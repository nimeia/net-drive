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
	"developer-mount/internal/winclientrecovery"
	"developer-mount/internal/winclientrelease"
	"developer-mount/internal/winclientruntime"
	"developer-mount/internal/winclientsmoke"
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
	CodeConfigValid            = "config.valid"
	CodeConfigInvalid          = "config.invalid"
	CodeWinFspBindingReady     = "winfsp.binding.ready"
	CodeWinFspBindingMissing   = "winfsp.binding.missing"
	CodeWinFspDispatcherReady  = "winfsp.dispatcher.ready"
	CodeWinFspDispatcherGap    = "winfsp.dispatcher.gap"
	CodeWinFspCallbackReady    = "winfsp.callback_bridge.ready"
	CodeWinFspServiceLoopReady = "winfsp.service_loop.ready"
	CodeWinFspNativeCallbacks  = "winfsp.native_callback_table"
	CodeExplorerRequestMatrix  = "smoke.explorer.request_matrix"
	CodeServerConnectOK        = "network.server_connect.ok"
	CodeServerConnectFailed    = "network.server_connect.failed"
	CodeStorePathReady         = "storage.store_path.ready"
	CodeStorePathMissing       = "storage.store_path.missing"
	CodeLogPathReady           = "storage.log_path.ready"
	CodeLogPathMissing         = "storage.log_path.missing"
	CodeRecoveryClean          = "recovery.clean"
	CodeRecoveryUnclean        = "recovery.unclean"
	CodeRuntimeHealthy         = "runtime.state.healthy"
	CodeRuntimeError           = "runtime.state.error"
	CodeExplorerSmokeDefined   = "smoke.explorer.defined"
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
	GeneratedAt    time.Time                    `json:"generated_at"`
	Config         winclient.Config             `json:"config"`
	Snapshot       winclientruntime.Snapshot    `json:"snapshot"`
	Binding        winfsp.BindingInfo           `json:"binding"`
	Recovery       winclientrecovery.State      `json:"recovery"`
	CallbackTable  winfsp.NativeCallbackTable   `json:"callback_table"`
	ExplorerMatrix winclientsmoke.RequestMatrix `json:"explorer_request_matrix"`
	SmokeScenarios []winclientsmoke.Scenario    `json:"smoke_scenarios,omitempty"`
	Checks         []CheckResult                `json:"checks"`
	Summary        Summary                      `json:"summary"`
	StorePath      string                       `json:"store_path,omitempty"`
	LogPath        string                       `json:"log_path,omitempty"`
	RecoveryPath   string                       `json:"recovery_path,omitempty"`
	LogTail        string                       `json:"log_tail,omitempty"`
	Notes          []string                     `json:"notes,omitempty"`
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
func (c Checker) Run(ctx context.Context, cfg winclient.Config, snapshot winclientruntime.Snapshot, storePath, logPath, recoveryPath string, recovery winclientrecovery.State, logTail string, smokeScenarios []winclientsmoke.Scenario) Report {
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.DialContext == nil {
		d := net.Dialer{Timeout: 1500 * time.Millisecond}
		c.DialContext = d.DialContext
	}
	cfg = cfg.Normalized()
	if len(smokeScenarios) == 0 {
		smokeScenarios = winclientsmoke.DefaultExplorerSmoke()
	}
	report := Report{GeneratedAt: c.Now(), Config: cfg, Snapshot: snapshot, StorePath: strings.TrimSpace(storePath), LogPath: strings.TrimSpace(logPath), RecoveryPath: strings.TrimSpace(recoveryPath), Recovery: recovery, LogTail: logTail, SmokeScenarios: smokeScenarios}
	if err := cfg.Validate(winclient.OperationMount); err != nil {
		report.Checks = append(report.Checks, newCheck(CodeConfigInvalid, "configuration", "Current profile configuration", StatusFail, err.Error(), "Fix the highlighted mount/profile fields and run self-check again."))
	} else {
		report.Checks = append(report.Checks, newCheck(CodeConfigValid, "configuration", "Current profile configuration", StatusPass, "mount configuration is structurally valid", ""))
	}
	binding, err := winfsp.Probe(winfsp.HostConfig{MountPoint: cfg.MountPoint, VolumePrefix: cfg.VolumePrefix, Backend: cfg.HostBackend})
	report.Binding = binding
	report.CallbackTable = winfsp.DefaultNativeCallbackTable(binding)
	report.ExplorerMatrix = winclientsmoke.DefaultExplorerRequestMatrix(report.CallbackTable)
	if err != nil {
		report.Checks = append(report.Checks, newCheck(CodeWinFspBindingMissing, "winfsp", "WinFsp binding", StatusFail, err.Error(), bindingRemediation(binding)))
	} else {
		report.Checks = append(report.Checks, newCheck(CodeWinFspBindingReady, "winfsp", "WinFsp binding", StatusPass, binding.Summary(), ""))
		if strings.Contains(strings.ToLower(binding.EffectiveBackend), "dispatcher") {
			status, code, remediation := StatusPass, CodeWinFspDispatcherReady, ""
			if !binding.DispatcherReady {
				status, code, remediation = StatusFail, CodeWinFspDispatcherGap, "Install a WinFsp build that exports dispatcher APIs, or switch host backend back to preflight."
			}
			report.Checks = append(report.Checks, newCheck(code, "winfsp", "Dispatcher APIs", status, defaultText(binding.DispatcherStatus, "dispatcher status unavailable"), remediation))
			report.Checks = append(report.Checks, newCheck(CodeWinFspCallbackReady, "winfsp", "Dispatcher callback bridge", boolStatus(binding.CallbackBridgeReady), defaultText(binding.CallbackBridgeStatus, "callback bridge status unavailable"), "Use dispatcher-v1 only after callback bridge status reports ready."))
			report.Checks = append(report.Checks, newCheck(CodeWinFspServiceLoopReady, "winfsp", "Dispatcher service loop", boolStatus(binding.ServiceLoopReady), defaultText(binding.ServiceLoopStatus, "service loop status unavailable"), "Verify the dispatcher service loop starts cleanly before running Explorer smoke."))
		}
	}
	callbackStatus, callbackRemediation := StatusPass, ""
	if report.CallbackTable.Preflight > 0 && !report.CallbackTable.Active {
		callbackStatus, callbackRemediation = StatusWarn, "Switch to dispatcher-v1 on a Windows host to exercise the native callback table."
	}
	if report.CallbackTable.MissingHotPathCount() > 0 {
		callbackStatus, callbackRemediation = StatusWarn, "Close the remaining Explorer hot-path callback gaps before claiming full native callback coverage."
	}
	report.Checks = append(report.Checks, newCheck(CodeWinFspNativeCallbacks, "winfsp", "Native callback table", callbackStatus, report.CallbackTable.Summary(), callbackRemediation))
	matrixStatus, matrixRemediation := StatusPass, ""
	if report.ExplorerMatrix.Gaps > 0 {
		matrixStatus, matrixRemediation = StatusWarn, "Review explorer-request-matrix.md and close the remaining callback gaps before broad Windows-host rollout."
	}
	report.Checks = append(report.Checks, newCheck(CodeExplorerRequestMatrix, "smoke", "Explorer request matrix", matrixStatus, report.ExplorerMatrix.Summary(), matrixRemediation))
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
	report.Checks = append(report.Checks, pathCheck("Client config store", storePath, CodeStorePathReady, CodeStorePathMissing), pathCheck("Client log file", logPath, CodeLogPathReady, CodeLogPathMissing), pathCheck("Crash recovery marker", recoveryPath, CodeStorePathReady, CodeStorePathMissing))
	if recovery.Dirty {
		report.Checks = append(report.Checks, newCheck(CodeRecoveryUnclean, "recovery", "Previous shutdown state", StatusWarn, recovery.Summary(), "Review the recovery marker, runtime log tail, and rerun Explorer smoke after a clean stop."))
	} else {
		report.Checks = append(report.Checks, newCheck(CodeRecoveryClean, "recovery", "Previous shutdown state", StatusPass, recovery.Summary(), ""))
	}
	report.Checks = append(report.Checks, runtimeCheck(snapshot), newCheck(CodeExplorerSmokeDefined, "smoke", "Explorer smoke manifest", StatusPass, fmt.Sprintf("%d scenarios prepared for Windows host validation", len(smokeScenarios)), "Run the smoke checklist on a Windows host after dispatcher-v1 reaches ready state."))
	if strings.Contains(strings.ToLower(binding.Backend), "dispatcher") || strings.Contains(strings.ToLower(binding.EffectiveBackend), "dispatcher") {
		report.Notes = append(report.Notes, "dispatcher-v1 now includes ABI bridge, service-loop scaffolding, a native callback coverage table, and an Explorer request matrix for Windows-host validation.")
	}
	report.Notes = append(report.Notes, "The diagnostics export now bundles an Explorer smoke checklist, the Explorer request matrix, the native callback table, and the current crash-recovery marker.")
	report.Summary = summarizeChecks(report.Checks)
	return report
}
func boolStatus(ok bool) Status {
	if ok {
		return StatusPass
	}
	return StatusWarn
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
	lines := []string{fmt.Sprintf("Generated: %s", r.GeneratedAt.Format(time.RFC3339)), fmt.Sprintf("Overall severity: %s", strings.ToUpper(string(r.Summary.OverallSeverity))), fmt.Sprintf("Check summary: pass=%d warn=%d fail=%d", r.Summary.Pass, r.Summary.Warn, r.Summary.Fail), fmt.Sprintf("Server: %s", r.Config.Addr), fmt.Sprintf("Mount point: %s", r.Config.MountPoint), fmt.Sprintf("Host backend: requested=%s effective=%s", defaultText(r.Config.HostBackend, "auto"), defaultText(r.Binding.EffectiveBackend, r.Binding.Backend)), fmt.Sprintf("Binding: %s", r.Binding.Summary()), fmt.Sprintf("Native callback table: %s", r.CallbackTable.Summary()), fmt.Sprintf("Explorer request matrix: %s", r.ExplorerMatrix.Summary()), fmt.Sprintf("Recovery: %s", r.Recovery.Summary())}
	if strings.TrimSpace(r.StorePath) != "" {
		lines = append(lines, fmt.Sprintf("Store path: %s", r.StorePath))
	}
	if strings.TrimSpace(r.LogPath) != "" {
		lines = append(lines, fmt.Sprintf("Log path: %s", r.LogPath))
	}
	if strings.TrimSpace(r.RecoveryPath) != "" {
		lines = append(lines, fmt.Sprintf("Recovery path: %s", r.RecoveryPath))
	}
	lines = append(lines, "", "Checks:")
	for _, check := range r.Checks {
		lines = append(lines, fmt.Sprintf("- [%s/%s] %s (%s): %s", strings.ToUpper(string(check.Status)), strings.ToUpper(string(check.Severity)), check.Name, check.Code, check.Detail))
		if check.Remediation != "" {
			lines = append(lines, "  remediation: "+check.Remediation)
		}
	}
	if len(r.SmokeScenarios) > 0 {
		lines = append(lines, "", fmt.Sprintf("Explorer smoke scenarios: %d", len(r.SmokeScenarios)))
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
	if len(report.SmokeScenarios) > 0 {
		if err := writeZipEntry(zw, "explorer-smoke.md", []byte(winclientsmoke.Markdown(report.SmokeScenarios))); err != nil {
			return "", err
		}
		smokePayload, err := winclientsmoke.JSON(report.SmokeScenarios)
		if err != nil {
			return "", err
		}
		if err := writeZipEntry(zw, "explorer-smoke.json", append(smokePayload, '\n')); err != nil {
			return "", err
		}
	}
	if err := writeZipEntry(zw, "explorer-request-matrix.md", []byte(report.ExplorerMatrix.Markdown())); err != nil {
		return "", err
	}
	matrixPayload, err := report.ExplorerMatrix.JSON()
	if err != nil {
		return "", err
	}
	if err := writeZipEntry(zw, "explorer-request-matrix.json", append(matrixPayload, '\n')); err != nil {
		return "", err
	}
	if err := writeZipEntry(zw, "winfsp-native-callbacks.md", []byte(report.CallbackTable.Markdown())); err != nil {
		return "", err
	}
	callbackPayload, err := report.CallbackTable.JSON()
	if err != nil {
		return "", err
	}
	if err := writeZipEntry(zw, "winfsp-native-callbacks.json", append(callbackPayload, '\n')); err != nil {
		return "", err
	}
	recoveryPayload, err := json.MarshalIndent(report.Recovery, "", "  ")
	if err == nil {
		if err := writeZipEntry(zw, "recovery.json", append(recoveryPayload, '\n')); err != nil {
			return "", err
		}
	}
	validation := winclientrelease.NewHostValidationRecord("", report.SmokeScenarios, report.CallbackTable, report.ExplorerMatrix)
	if err := writeZipEntry(zw, "windows-host-validation-template.md", []byte(validation.Markdown())); err != nil {
		return "", err
	}
	validationPayload, err := validation.JSON()
	if err != nil {
		return "", err
	}
	if err := writeZipEntry(zw, "windows-host-validation-template.json", append(validationPayload, '\n')); err != nil {
		return "", err
	}
	if err := writeZipEntry(zw, "windows-host-validation-result-template.md", []byte(validation.Markdown())); err != nil {
		return "", err
	}
	if err := writeZipEntry(zw, "windows-host-validation-result-template.json", append(validationPayload, '\n')); err != nil {
		return "", err
	}
	manifest := winclientrelease.NewManifest("", nil, report.CallbackTable, report.ExplorerMatrix, report.SmokeScenarios)
	closure := winclientrelease.NewReleaseClosure(manifest, validation)
	issues := winclientrelease.NewPreReleaseIssueList(manifest, validation, closure)
	patch := winclientrelease.NewValidationPatchTemplate(validation)
	if err := writeZipEntry(zw, "windows-host-backfill-patch-template.md", []byte(`# Windows Host Backfill Patch Template

Update the matching JSON file with first-pass Windows host results, then merge it into windows-host-validation-result-template.json.
`)); err != nil {
		return "", err
	}
	patchPayload, err := json.MarshalIndent(patch, "", "  ")
	if err != nil {
		return "", err
	}
	if err := writeZipEntry(zw, "windows-host-backfill-patch-template.json", append(patchPayload, byte(10))); err != nil {
		return "", err
	}
	if err := writeZipEntry(zw, "windows-release-closure-template.md", []byte(closure.Markdown())); err != nil {
		return "", err
	}
	closurePayload, err := closure.JSON()
	if err != nil {
		return "", err
	}
	if err := writeZipEntry(zw, "windows-release-closure-template.json", append(closurePayload, byte(10))); err != nil {
		return "", err
	}
	if err := writeZipEntry(zw, "windows-pre-release-issues.md", []byte(issues.Markdown())); err != nil {
		return "", err
	}
	issuePayload, err := issues.JSON()
	if err != nil {
		return "", err
	}
	if err := writeZipEntry(zw, "windows-pre-release-issues.json", append(issuePayload, byte(10))); err != nil {
		return "", err
	}
	intake := winclientrelease.NewValidationIntakeReport(manifest, validation)
	if err := writeZipEntry(zw, "windows-validation-intake-report.md", []byte(intake.Markdown())); err != nil {
		return "", err
	}
	intakePayload, err := intake.JSON()
	if err != nil {
		return "", err
	}
	if err := writeZipEntry(zw, "windows-validation-intake-report.json", append(intakePayload, byte(10))); err != nil {
		return "", err
	}
	fixPlan := winclientrelease.NewFirstPassFixPlan(manifest, validation, issues)
	if err := writeZipEntry(zw, "windows-first-pass-fix-plan.md", []byte(fixPlan.Markdown())); err != nil {
		return "", err
	}
	fixPlanPayload, err := fixPlan.JSON()
	if err != nil {
		return "", err
	}
	if err := writeZipEntry(zw, "windows-first-pass-fix-plan.json", append(fixPlanPayload, byte(10))); err != nil {
		return "", err
	}
	rc := winclientrelease.NewReleaseCandidate(manifest, validation, closure, issues)
	if err := writeZipEntry(zw, "windows-release-candidate.md", []byte(rc.Markdown())); err != nil {
		return "", err
	}
	rcPayload, err := rc.JSON()
	if err != nil {
		return "", err
	}
	if err := writeZipEntry(zw, "windows-release-candidate.json", append(rcPayload, byte(10))); err != nil {
		return "", err
	}
	finalRelease := winclientrelease.NewFinalRelease(manifest, validation, intake, closure, issues, rc)
	if err := writeZipEntry(zw, "windows-final-release.md", []byte(finalRelease.Markdown())); err != nil {
		return "", err
	}
	finalPayload, err := finalRelease.JSON()
	if err != nil {
		return "", err
	}
	if err := writeZipEntry(zw, "windows-final-release.json", append(finalPayload, byte(10))); err != nil {
		return "", err
	}
	if err := writeZipEntry(zw, "windows-final-signoff.md", []byte(finalRelease.SignoffMarkdown())); err != nil {
		return "", err
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
