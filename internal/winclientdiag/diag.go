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
type Severity string

const (
	StatusPass      Status   = "pass"
	StatusWarn      Status   = "warn"
	StatusFail      Status   = "fail"
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

const (
	CodeConfigValid           = "config.valid"
	CodeConfigInvalid         = "config.invalid"
	CodeWinFspBindingReady    = "winfsp.binding.ready"
	CodeWinFspBindingMissing  = "winfsp.binding.missing"
	CodeWinFspDispatcherReady = "winfsp.dispatcher.ready"
	CodeWinFspDispatcherGap   = "winfsp.dispatcher.gap"
	CodeServerConnectOK       = "network.server_connect.ok"
	CodeServerConnectFailed   = "network.server_connect.failed"
	CodeStorePathReady        = "storage.store_path.ready"
	CodeStorePathMissing      = "storage.store_path.missing"
	CodeLogPathReady          = "storage.log_path.ready"
	CodeLogPathMissing        = "storage.log_path.missing"
	CodeRuntimeHealthy        = "runtime.state.healthy"
	CodeRuntimeError          = "runtime.state.error"
)

type CheckResult struct {
	Code        string   `json:"code"`
	Category    string   `json:"category"`
	Name        string   `json:"name"`
	Status      Status   `json:"status"`
	Severity    Severity `json:"severity"`
	Detail      string   `json:"detail"`
	Remediation string   `json:"remediation,omitempty"`
}

type Summary struct {
	Pass            int      `json:"pass"`
	Warn            int      `json:"warn"`
	Fail            int      `json:"fail"`
	OverallSeverity Severity `json:"overall_severity"`
}

type Report struct {
	GeneratedAt time.Time                 `json:"generated_at"`
	Config      winclient.Config          `json:"config"`
	Snapshot    winclientruntime.Snapshot `json:"snapshot"`
	Binding     winfsp.BindingInfo        `json:"binding"`
	Checks      []CheckResult             `json:"checks"`
	Summary     Summary                   `json:"summary"`
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
		report.Checks = append(report.Checks, newCheck(CodeConfigInvalid, "configuration", "Current profile configuration", StatusFail, err.Error(), "Fix the highlighted mount/profile fields and run self-check again."))
	} else {
		report.Checks = append(report.Checks, newCheck(CodeConfigValid, "configuration", "Current profile configuration", StatusPass, "mount configuration is structurally valid", ""))
	}
	binding, err := winfsp.Probe(winfsp.HostConfig{MountPoint: cfg.MountPoint, VolumePrefix: cfg.VolumePrefix, Backend: cfg.HostBackend})
	report.Binding = binding
	if err != nil {
		report.Checks = append(report.Checks, newCheck(CodeWinFspBindingMissing, "winfsp", "WinFsp binding", StatusFail, err.Error(), bindingRemediation(binding)))
	} else {
		report.Checks = append(report.Checks, newCheck(CodeWinFspBindingReady, "winfsp", "WinFsp binding", StatusPass, binding.Summary(), ""))
		if strings.Contains(strings.ToLower(binding.EffectiveBackend), "dispatcher") {
			status, code, remediation := StatusPass, CodeWinFspDispatcherReady, ""
			if !binding.DispatcherReady {
				status, code, remediation = StatusFail, CodeWinFspDispatcherGap, "Install a WinFsp build that exports dispatcher APIs, or switch host backend back to preflight."
			}
			report.Checks = append(report.Checks, newCheck(code, "winfsp", "Dispatcher callback bridge", status, defaultText(binding.DispatcherStatus, "dispatcher status unavailable"), remediation))
		}
	}
	if strings.TrimSpace(cfg.Addr) == "" {
		report.Checks = append(report.Checks, newCheck(CodeServerConnectFailed, "network", "Server TCP connect", StatusFail, "server address is empty", "Enter a reachable devmount-server address before starting the mount."))
	} else {
		dialCtx, cancel := context.WithTimeout(ctx, c.DialTimeout)
		conn, dialErr := c.DialContext(dialCtx, "tcp", cfg.Addr)
		cancel()
		if dialErr != nil {
			report.Checks = append(report.Checks, newCheck(CodeServerConnectFailed, "network", "Server TCP connect", StatusFail, dialErr.Error(), "Verify the server is running, the address is correct, and firewalls allow the connection."))
		} else {
			_ = conn.Close()
			report.Checks = append(report.Checks, newCheck(CodeServerConnectOK, "network", "Server TCP connect", StatusPass, "TCP handshake succeeded", ""))
		}
	}
	report.Checks = append(report.Checks, pathCheck("Client config store", storePath, CodeStorePathReady, CodeStorePathMissing))
	report.Checks = append(report.Checks, pathCheck("Client log file", logPath, CodeLogPathReady, CodeLogPathMissing))
	report.Checks = append(report.Checks, runtimeCheck(snapshot))
	if strings.Contains(strings.ToLower(binding.Backend), "dispatcher") || strings.Contains(strings.ToLower(binding.EffectiveBackend), "dispatcher") {
		report.Notes = append(report.Notes, "dispatcher-v1 now includes a first callback bridge for volume/getattr/open/readdir/read/close lifecycle warmup and host state reporting.")
	}
	report.Summary = summarizeChecks(report.Checks)
	return report
}

func newCheck(code, category, name string, status Status, detail, remediation string) CheckResult {
	return CheckResult{Code: code, Category: category, Name: name, Status: status, Severity: severityForStatus(status), Detail: strings.TrimSpace(detail), Remediation: strings.TrimSpace(remediation)}
}
func severityForStatus(status Status) Severity {
	switch status {
	case StatusFail:
		return SeverityError
	case StatusWarn:
		return SeverityWarning
	default:
		return SeverityInfo
	}
}
func summarizeChecks(checks []CheckResult) Summary {
	s := Summary{OverallSeverity: SeverityInfo}
	for _, c := range checks {
		switch c.Status {
		case StatusPass:
			s.Pass++
		case StatusWarn:
			s.Warn++
		case StatusFail:
			s.Fail++
		}
	}
	if s.Fail > 0 {
		s.OverallSeverity = SeverityError
	} else if s.Warn > 0 {
		s.OverallSeverity = SeverityWarning
	}
	return s
}
func bindingRemediation(binding winfsp.BindingInfo) string {
	if strings.TrimSpace(binding.Note) != "" {
		return binding.Note
	}
	if strings.Contains(strings.ToLower(binding.DispatcherStatus), "find") || strings.Contains(strings.ToLower(binding.DispatcherStatus), "unavailable") {
		return "Install a WinFsp build that includes dispatcher APIs, or switch host backend to preflight."
	}
	return "Install WinFsp Developer components and retry the self-check."
}
func runtimeCheck(snapshot winclientruntime.Snapshot) CheckResult {
	detail := fmt.Sprintf("phase=%s status=%s", snapshot.Phase, defaultText(snapshot.StatusText, "-"))
	if strings.TrimSpace(snapshot.LastError) != "" || snapshot.Phase == winclientruntime.PhaseError {
		if snapshot.LastError != "" {
			detail += "; last_error=" + snapshot.LastError
		}
		return newCheck(CodeRuntimeError, "runtime", "Mount runtime state", StatusWarn, detail, "Use Start Mount again after fixing the connection/binding error, or inspect the log tail for the previous failure.")
	}
	return newCheck(CodeRuntimeHealthy, "runtime", "Mount runtime state", StatusPass, detail, "")
}
func pathCheck(name, path, passCode, missingCode string) CheckResult {
	path = strings.TrimSpace(path)
	if path == "" {
		return newCheck(missingCode, "storage", name, StatusWarn, "path is empty", "Configure a writable path so diagnostics and logs can be persisted locally.")
	}
	if _, err := os.Stat(path); err == nil {
		return newCheck(passCode, "storage", name, StatusPass, path, "")
	} else if os.IsNotExist(err) {
		parent := filepath.Dir(path)
		if parent == "" {
			parent = path
		}
		if _, parentErr := os.Stat(parent); parentErr == nil {
			return newCheck(missingCode, "storage", name, StatusWarn, fmt.Sprintf("%s does not exist yet, but parent directory is available", path), "The file will be created on the next save/export operation.")
		}
		return newCheck(missingCode, "storage", name, StatusFail, err.Error(), "Create the parent directory or choose a writable path.")
	} else {
		return newCheck(missingCode, "storage", name, StatusFail, err.Error(), "Verify the path is writable and not blocked by permissions or antivirus.")
	}
}
func (r Report) Text() string {
	lines := []string{fmt.Sprintf("Generated: %s", r.GeneratedAt.Format(time.RFC3339)), fmt.Sprintf("Overall severity: %s", strings.ToUpper(string(r.Summary.OverallSeverity))), fmt.Sprintf("Check summary: pass=%d warn=%d fail=%d", r.Summary.Pass, r.Summary.Warn, r.Summary.Fail), fmt.Sprintf("Server: %s", r.Config.Addr), fmt.Sprintf("Mount point: %s", r.Config.MountPoint), fmt.Sprintf("Host backend: requested=%s effective=%s", defaultText(r.Config.HostBackend, "auto"), defaultText(r.Binding.EffectiveBackend, r.Binding.Backend)), fmt.Sprintf("Binding: %s", r.Binding.Summary())}
	if strings.TrimSpace(r.StorePath) != "" {
		lines = append(lines, fmt.Sprintf("Store path: %s", r.StorePath))
	}
	if strings.TrimSpace(r.LogPath) != "" {
		lines = append(lines, fmt.Sprintf("Log path: %s", r.LogPath))
	}
	lines = append(lines, "", "Checks:")
	for _, check := range r.Checks {
		lines = append(lines, fmt.Sprintf("- [%s/%s] %s (%s): %s", strings.ToUpper(string(check.Status)), strings.ToUpper(string(check.Severity)), check.Name, check.Code, check.Detail))
		if check.Remediation != "" {
			lines = append(lines, "  remediation: "+check.Remediation)
		}
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
	return filepath.Join(cacheDir, "developer-mount", fmt.Sprintf("developer-mount-diagnostics-%s.zip", time.Now().Format("20060102-150405"))), nil
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
