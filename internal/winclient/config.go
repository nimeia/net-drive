package winclient

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type Operation string

const (
	OperationMount       Operation = "mount"
	OperationVolume      Operation = "volume"
	OperationGetAttr     Operation = "getattr"
	OperationReadDir     Operation = "readdir"
	OperationRead        Operation = "read"
	OperationMaterialize Operation = "materialize"
)

const (
	HostBackendAuto         = "auto"
	HostBackendPreflight    = "preflight"
	HostBackendDispatcherV1 = "dispatcher-v1"
)

type Config struct {
	Addr             string
	Token            string
	ClientInstanceID string
	LeaseSeconds     uint32
	MountPoint       string
	VolumePrefix     string
	Path             string
	LocalPath        string
	HostBackend      string
	Offset           int64
	Length           uint32
	MaxEntries       uint32
}

func DefaultConfig() Config {
	return Config{Addr: "127.0.0.1:17890", Token: "devmount-dev-token", ClientInstanceID: "win32-test-ui", LeaseSeconds: 30, MountPoint: defaultMountPoint(), VolumePrefix: "devmount", Path: "/", LocalPath: "devmount-local", HostBackend: HostBackendAuto, Offset: 0, Length: 64, MaxEntries: 32}
}

func Operations() []Operation {
	return []Operation{OperationVolume, OperationGetAttr, OperationReadDir, OperationRead, OperationMaterialize}
}
func HostBackendOptions() []string {
	return []string{HostBackendAuto, HostBackendPreflight, HostBackendDispatcherV1}
}
func NormalizeHostBackend(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", HostBackendAuto:
		return HostBackendAuto
	case HostBackendPreflight:
		return HostBackendPreflight
	case HostBackendDispatcherV1:
		return HostBackendDispatcherV1
	default:
		return value
	}
}

func NormalizeMountPoint(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && isAlpha(value[0]) && value[1] == ':' {
		drive := strings.ToUpper(value[:1]) + ":"
		if len(value) == 2 {
			return drive
		}
		if len(value) == 3 && isPathSeparator(value[2]) {
			return drive
		}
	}
	return value
}

func ValidateMountPoint(value string) error {
	value = NormalizeMountPoint(value)
	if value == "" {
		return fmt.Errorf("mount point is required")
	}
	if len(value) == 2 && isAlpha(value[0]) && value[1] == ':' {
		return nil
	}
	if isLikelyAbsoluteMountDir(value) {
		return nil
	}
	return fmt.Errorf("mount point %q is invalid; expected a drive letter like X: or an absolute directory path", value)
}

func ResolveMountPointForStart(value string) (string, bool) {
	value = NormalizeMountPoint(value)
	if !isDriveLetterMountPoint(value) {
		return value, false
	}
	if !driveRootExists(value + `\`) {
		return value, false
	}
	if next := nextAvailableDriveLetter(value); next != "" && next != value {
		return next, true
	}
	return value, false
}

func (c Config) Normalized() Config {
	defaults := DefaultConfig()
	if strings.TrimSpace(c.Addr) == "" {
		c.Addr = defaults.Addr
	}
	if strings.TrimSpace(c.Token) == "" {
		c.Token = defaults.Token
	}
	if strings.TrimSpace(c.ClientInstanceID) == "" {
		c.ClientInstanceID = defaults.ClientInstanceID
	}
	if c.LeaseSeconds == 0 {
		c.LeaseSeconds = defaults.LeaseSeconds
	}
	if strings.TrimSpace(c.MountPoint) == "" {
		c.MountPoint = defaults.MountPoint
	}
	c.MountPoint = NormalizeMountPoint(c.MountPoint)
	if strings.TrimSpace(c.VolumePrefix) == "" {
		c.VolumePrefix = defaults.VolumePrefix
	}
	if strings.TrimSpace(c.Path) == "" {
		c.Path = defaults.Path
	}
	if !strings.HasPrefix(c.Path, "/") {
		c.Path = "/" + c.Path
	}
	if strings.TrimSpace(c.LocalPath) == "" {
		c.LocalPath = defaults.LocalPath
	}
	c.HostBackend = NormalizeHostBackend(c.HostBackend)
	if c.Length == 0 {
		c.Length = defaults.Length
	}
	if c.MaxEntries == 0 {
		c.MaxEntries = defaults.MaxEntries
	}
	return c
}

func (c Config) Validate(op Operation) error {
	c = c.Normalized()
	if strings.TrimSpace(c.Addr) == "" {
		return fmt.Errorf("server address is required")
	}
	if strings.TrimSpace(c.Token) == "" {
		return fmt.Errorf("token is required")
	}
	if strings.TrimSpace(c.ClientInstanceID) == "" {
		return fmt.Errorf("client instance id is required")
	}
	if c.LeaseSeconds == 0 {
		return fmt.Errorf("lease seconds must be greater than 0")
	}
	if err := ValidateMountPoint(c.MountPoint); err != nil {
		return err
	}
	if strings.TrimSpace(c.VolumePrefix) == "" {
		return fmt.Errorf("volume prefix is required")
	}
	if strings.TrimSpace(c.Path) == "" {
		return fmt.Errorf("path is required")
	}
	if !strings.HasPrefix(c.Path, "/") {
		return fmt.Errorf("path must start with '/'")
	}
	if c.HostBackend != HostBackendAuto && c.HostBackend != HostBackendPreflight && c.HostBackend != HostBackendDispatcherV1 {
		return fmt.Errorf("unsupported host backend %q", c.HostBackend)
	}
	if op == OperationMaterialize && strings.TrimSpace(c.LocalPath) == "" {
		return fmt.Errorf("local path is required for materialize")
	}
	switch op {
	case OperationMount, OperationVolume, OperationGetAttr, OperationReadDir, OperationRead, OperationMaterialize:
	default:
		return fmt.Errorf("unsupported operation %q", op)
	}
	return nil
}

func BuildCLIPreview(config Config, op Operation) string {
	config = config.Normalized()
	args := []string{"devmount-winfsp.exe", "-addr", quoteIfNeeded(config.Addr), "-token", quoteIfNeeded(config.Token), "-client-instance", quoteIfNeeded(config.ClientInstanceID), "-op", string(op), "-path", quoteIfNeeded(config.Path), "-mount-point", quoteIfNeeded(config.MountPoint), "-volume-prefix", quoteIfNeeded(config.VolumePrefix), "-host-backend", quoteIfNeeded(config.HostBackend)}
	switch op {
	case OperationRead:
		args = append(args, "-offset", strconv.FormatInt(config.Offset, 10), "-length", strconv.FormatUint(uint64(config.Length), 10))
	case OperationReadDir:
		args = append(args, "-max-entries", strconv.FormatUint(uint64(config.MaxEntries), 10))
	case OperationMaterialize:
		args = append(args, "-local-path", quoteIfNeeded(config.LocalPath), "-length", strconv.FormatUint(uint64(config.Length), 10), "-max-entries", strconv.FormatUint(uint64(config.MaxEntries), 10))
	}
	return strings.Join(args, " ")
}

func quoteIfNeeded(s string) string {
	if s == "" {
		return `""`
	}
	if strings.IndexFunc(s, func(r rune) bool { return r == ' ' || r == '\t' || r == '"' }) == -1 {
		return s
	}
	escaped := strings.ReplaceAll(s, `"`, `\\"`)
	return `"` + escaped + `"`
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isPathSeparator(b byte) bool {
	return b == '\\' || b == '/'
}

func isLikelyAbsoluteMountDir(value string) bool {
	if len(value) >= 3 && isAlpha(value[0]) && value[1] == ':' && isPathSeparator(value[2]) {
		return true
	}
	return strings.HasPrefix(value, `\\`) || strings.HasPrefix(value, `/`)
}

var driveRootExists = func(root string) bool {
	_, err := os.Stat(root)
	return err == nil
}

func defaultMountPoint() string {
	if runtime.GOOS != "windows" {
		return "M:"
	}
	for _, letter := range candidateDriveLetters() {
		root := string(letter) + `:\`
		if !driveRootExists(root) {
			return string(letter) + ":"
		}
	}
	return "M:"
}

func candidateDriveLetters() []byte {
	return []byte{'Z', 'Y', 'X', 'W', 'V', 'U', 'T', 'S', 'R', 'Q', 'P', 'O', 'N', 'M', 'L', 'K', 'J', 'I', 'H', 'G', 'F', 'E', 'D'}
}

func isDriveLetterMountPoint(value string) bool {
	return len(value) == 2 && isAlpha(value[0]) && value[1] == ':'
}

func nextAvailableDriveLetter(current string) string {
	current = NormalizeMountPoint(current)
	letters := candidateDriveLetters()
	start := -1
	for i, letter := range letters {
		if strings.EqualFold(string(letter)+":", current) {
			start = i
			break
		}
	}
	if start == -1 {
		return defaultMountPoint()
	}
	for offset := 1; offset < len(letters); offset++ {
		letter := letters[(start+offset)%len(letters)]
		root := string(letter) + `:\`
		if !driveRootExists(root) {
			return string(letter) + ":"
		}
	}
	return ""
}
