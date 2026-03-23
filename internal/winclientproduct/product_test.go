package winclientproduct

import (
	"strings"
	"testing"

	"developer-mount/internal/winclient"
	"developer-mount/internal/winclientdiag"
	"developer-mount/internal/winclientruntime"
	"developer-mount/internal/winclientstore"
)

func TestFriendlyRuntimeError(t *testing.T) {
	cases := map[string]string{
		"dial tcp 127.0.0.1:1: connect: connection refused": "无法连接到服务端，请检查地址、网络或防火墙设置。",
		"WinFsp dispatcher unavailable":                     "当前设备缺少可用的 WinFsp 运行环境，请先安装或修复 WinFsp。",
	}
	for input, want := range cases {
		if got := FriendlyRuntimeError(input); got != want {
			t.Fatalf("FriendlyRuntimeError(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestHomeSummaryIncludesReadOnlyMode(t *testing.T) {
	state := winclientstore.DefaultState()
	state.ActiveProfile = "dev"
	state.Profiles["dev"] = winclient.Config{Addr: "127.0.0.1:17890", MountPoint: "X:", Path: "/repo"}.Normalized()
	state.WorkspaceMeta["dev"] = winclientstore.WorkspaceMeta{DisplayName: "研发空间"}
	summary := HomeSummary(winclientruntime.Snapshot{Phase: winclientruntime.PhaseMounted, ActiveProfile: "dev", MountPoint: "X:", RemotePath: "/repo", ServerAddr: "127.0.0.1:17890"}, state)
	if !strings.Contains(summary, "访问模式：只读挂载") {
		t.Fatalf("missing read-only note: %s", summary)
	}
	if !strings.Contains(summary, "研发空间") {
		t.Fatalf("missing display name: %s", summary)
	}
}

func TestWorkspacesSummaryRendersFlags(t *testing.T) {
	state := winclientstore.DefaultState()
	state.ActiveProfile = "dev"
	state.Profiles["dev"] = winclient.DefaultConfig()
	state.WorkspaceMeta["dev"] = winclientstore.WorkspaceMeta{DisplayName: "工程", AutoMount: true, LastUsedAt: "2026-03-22T17:02:03Z"}
	summary := WorkspacesSummary(state)
	if !strings.Contains(summary, "当前") || !strings.Contains(summary, "开机自动连接") {
		t.Fatalf("missing flags: %s", summary)
	}
}

func TestHelpSummaryIncludesSupportConsolePath(t *testing.T) {
	report := winclientdiag.Report{}
	report.Summary.Pass = 3
	report.Checks = []winclientdiag.CheckResult{{Name: "WinFsp binding", Detail: "ready"}}
	summary := HelpSummary(report, `C:\logs\client.log`, `C:\dist\devmount-support-console.exe`)
	if !strings.Contains(summary, "devmount-support-console.exe") {
		t.Fatalf("missing path: %s", summary)
	}
}
