package winclientdiag

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"developer-mount/internal/winclient"
	"developer-mount/internal/winclientruntime"
	"developer-mount/internal/winfsp"
)

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

type CheckResult struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Detail string `json:"detail"`
}

type Report struct {
	GeneratedAt time.Time                 `json:"generated_at"`
	Config      winclient.Config          `json:"config"`
	Snapshot    winclientruntime.Snapshot `json:"snapshot"`
	Binding     winfsp.BindingInfo        `json:"binding"`
	Checks      []CheckResult             `json:"checks"`
	StorePath   string                    `json:"store_path,omitempty"`
	LogPath     string                    `json:"log_path,omitempty"`
	LogTail     string                    `json:"log_tail,omitempty"`
	Notes       []string                  `json:"notes,omitempty"`
}

type Checker struct {
	DialTimeout time.Duration
	DialContext func(ctx context.Context, network, address string) (net.Conn, error)
	Now         func() time.Time
}

func NewChecker() Checker {
	d := net.Dialer{Timeout: 1500 * time.Millisecond}
	return Checker{DialTimeout: 1500 * time.Millisecond, DialContext: d.DialContext, Now: time.Now}
}

func (c Checker) Run(ctx context.Context, cfg winclient.Config, snapshot winclientruntime.Snapshot, storePath, logPath, logTail string) Report {
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.DialContext == nil {
		d := net.Dialer{Timeout: 1500 * time.Millisecond}
		c.DialContext = d.DialContext
	}
	cfg = cfg.Normalized()
	report := Report{GeneratedAt: c.Now(), Config: cfg, Snapshot: snapshot, StorePath: strings.TrimSpace(storePath), LogPath: strings.TrimSpace(logPath), LogTail: logTail}
	if err := cfg.Validate(winclient.OperationMount); err != nil {
		report.Checks = append(report.Checks, CheckResult{Name: "config", Status: StatusFail, Detail: err.Error()})
	} else {
		report.Checks = append(report.Checks, CheckResult{Name: "config", Status: StatusPass, Detail: "mount configuration is structurally valid"})
	}
	binding, err := winfsp.Probe(winfsp.HostConfig{MountPoint: cfg.MountPoint, VolumePrefix: cfg.VolumePrefix, Backend: cfg.HostBackend})
	report.Binding = binding
	if err != nil {
		report.Checks = append(report.Checks, CheckResult{Name: "winfsp-binding", Status: StatusFail, Detail: err.Error()})
	} else {
		report.Checks = append(report.Checks, CheckResult{Name: "winfsp-binding", Status: StatusPass, Detail: binding.Summary()})
	}
	if strings.TrimSpace(cfg.Addr) == "" {
		report.Checks = append(report.Checks, CheckResult{Name: "server-connect", Status: StatusFail, Detail: "server address is empty"})
	} else {
		dialCtx, cancel := context.WithTimeout(ctx, c.DialTimeout)
		conn, dialErr := c.DialContext(dialCtx, "tcp", cfg.Addr)
		cancel()
		if dialErr != nil {
			report.Checks = append(report.Checks, CheckResult{Name: "server-connect", Status: StatusFail, Detail: dialErr.Error()})
		} else {
			_ = conn.Close()
			report.Checks = append(report.Checks, CheckResult{Name: "server-connect", Status: StatusPass, Detail: "TCP handshake succeeded"})
		}
	}
	report.Checks = append(report.Checks, pathCheck("store-path", storePath))
	report.Checks = append(report.Checks, pathCheck("log-path", logPath))
	if strings.Contains(strings.ToLower(binding.Backend), "dispatcher") || strings.Contains(strings.ToLower(binding.EffectiveBackend), "dispatcher") {
		report.Notes = append(report.Notes, "dispatcher-v1 currently validates WinFsp dispatcher API availability and backend selection, but the full callback ABI bridge is still an explicit scaffold boundary on this branch.")
	}
	return report
}

func pathCheck(name, path string) CheckResult {
	path = strings.TrimSpace(path)
	if path == "" {
		return CheckResult{Name: name, Status: StatusWarn, Detail: "path is empty"}
	}
	if _, err := os.Stat(path); err == nil {
		return CheckResult{Name: name, Status: StatusPass, Detail: path}
	} else if os.IsNotExist(err) {
		parent := filepath.Dir(path)
		if parent == "" {
			parent = path
		}
		if _, parentErr := os.Stat(parent); parentErr == nil {
			return CheckResult{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("%s does not exist yet, but parent directory is available", path)}
		}
		return CheckResult{Name: name, Status: StatusFail, Detail: err.Error()}
	} else {
		return CheckResult{Name: name, Status: StatusFail, Detail: err.Error()}
	}
}

func (r Report) Text() string {
	lines := []string{
		fmt.Sprintf("Generated: %s", r.GeneratedAt.Format(time.RFC3339)),
		fmt.Sprintf("Server: %s", r.Config.Addr),
		fmt.Sprintf("Mount point: %s", r.Config.MountPoint),
		fmt.Sprintf("Host backend: requested=%s effective=%s", defaultText(r.Config.HostBackend, "auto"), defaultText(r.Binding.EffectiveBackend, r.Binding.Backend)),
		fmt.Sprintf("Binding: %s", r.Binding.Summary()),
		"",
		"Checks:",
	}
	for _, check := range r.Checks {
		lines = append(lines, fmt.Sprintf("- [%s] %s: %s", strings.ToUpper(string(check.Status)), check.Name, check.Detail))
	}
	if len(r.Notes) > 0 {
		lines = append(lines, "", "Notes:")
		for _, note := range r.Notes {
			lines = append(lines, "- "+note)
		}
	}
	return strings.Join(lines, "\n")
}

func DefaultExportPath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve user cache dir: %w", err)
	}
	name := fmt.Sprintf("developer-mount-diagnostics-%s.zip", time.Now().Format("20060102-150405"))
	return filepath.Join(cacheDir, "developer-mount", name), nil
}

func Export(path string, report Report) (string, error) {
	if strings.TrimSpace(path) == "" {
		var err error
		path, err = DefaultExportPath()
		if err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create diagnostics directory: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create diagnostics zip: %w", err)
	}
	defer file.Close()
	zw := zip.NewWriter(file)
	if err := writeZipEntry(zw, "report.txt", []byte(report.Text()+"\n")); err != nil {
		return "", err
	}
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode diagnostics report: %w", err)
	}
	if err := writeZipEntry(zw, "report.json", append(payload, '\n')); err != nil {
		return "", err
	}
	if strings.TrimSpace(report.LogTail) != "" {
		if err := writeZipEntry(zw, "log-tail.txt", []byte(report.LogTail)); err != nil {
			return "", err
		}
	}
	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("close diagnostics zip: %w", err)
	}
	return path, nil
}

func writeZipEntry(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return fmt.Errorf("create zip entry %s: %w", name, err)
	}
	_, err = w.Write(data)
	return err
}

func defaultText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
