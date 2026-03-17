package winclient

import (
	"fmt"
	"strconv"
	"strings"
)

type Operation string

const (
	OperationVolume  Operation = "volume"
	OperationGetAttr Operation = "getattr"
	OperationReadDir Operation = "readdir"
	OperationRead    Operation = "read"
)

type Config struct {
	Addr             string
	Token            string
	ClientInstanceID string
	LeaseSeconds     uint32
	MountPoint       string
	VolumePrefix     string
	Path             string
	Offset           int64
	Length           uint32
	MaxEntries       uint32
}

func DefaultConfig() Config {
	return Config{
		Addr:             "127.0.0.1:17890",
		Token:            "devmount-dev-token",
		ClientInstanceID: "win32-test-ui",
		LeaseSeconds:     30,
		MountPoint:       "M:",
		VolumePrefix:     "devmount",
		Path:             "/",
		Offset:           0,
		Length:           64,
		MaxEntries:       32,
	}
}

func Operations() []Operation {
	return []Operation{OperationVolume, OperationGetAttr, OperationReadDir, OperationRead}
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
	if strings.TrimSpace(c.VolumePrefix) == "" {
		c.VolumePrefix = defaults.VolumePrefix
	}
	if strings.TrimSpace(c.Path) == "" {
		c.Path = defaults.Path
	}
	if !strings.HasPrefix(c.Path, "/") {
		c.Path = "/" + c.Path
	}
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
	if strings.TrimSpace(c.VolumePrefix) == "" {
		return fmt.Errorf("volume prefix is required")
	}
	if strings.TrimSpace(c.Path) == "" {
		return fmt.Errorf("path is required")
	}
	if !strings.HasPrefix(c.Path, "/") {
		return fmt.Errorf("path must start with '/'")
	}
	switch op {
	case OperationVolume, OperationGetAttr, OperationReadDir, OperationRead:
	default:
		return fmt.Errorf("unsupported operation %q", op)
	}
	return nil
}

func BuildCLIPreview(config Config, op Operation) string {
	config = config.Normalized()
	args := []string{
		"devmount-winfsp.exe",
		"-addr", quoteIfNeeded(config.Addr),
		"-token", quoteIfNeeded(config.Token),
		"-client-instance", quoteIfNeeded(config.ClientInstanceID),
		"-op", string(op),
		"-path", quoteIfNeeded(config.Path),
		"-mount-point", quoteIfNeeded(config.MountPoint),
		"-volume-prefix", quoteIfNeeded(config.VolumePrefix),
	}
	switch op {
	case OperationRead:
		args = append(args, "-offset", strconv.FormatInt(config.Offset, 10), "-length", strconv.FormatUint(uint64(config.Length), 10))
	case OperationReadDir:
		args = append(args, "-max-entries", strconv.FormatUint(uint64(config.MaxEntries), 10))
	}
	return strings.Join(args, " ")
}

func quoteIfNeeded(s string) string {
	if s == "" {
		return `""`
	}
	if strings.IndexFunc(s, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '"'
	}) == -1 {
		return s
	}
	escaped := strings.ReplaceAll(s, `"`, `\"`)
	return `"` + escaped + `"`
}
