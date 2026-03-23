package winclientproduct

import (
	"strings"
	"testing"

	"developer-mount/internal/winclientdiag"
)

func TestFriendlyRuntimeError(t *testing.T) {
	if got := FriendlyRuntimeError("dial tcp 127.0.0.1:1: connect: connection refused"); !strings.Contains(got, "无法连接") {
		t.Fatalf("got %q", got)
	}
}

func TestSupportConsolePathUsesCompanionExe(t *testing.T) {
	got := SupportConsolePath(`C:\dist\devmount-client-win32.exe`)
	if !strings.HasSuffix(strings.ToLower(got), `devmount-support-console.exe`) {
		t.Fatalf("SupportConsolePath() = %q", got)
	}
}

func TestHelpSummaryIncludesSupportConsolePath(t *testing.T) {
	report := winclientdiag.Report{}
	report.Summary.Pass = 3
	summary := HelpSummary(report, `C:\logs\client.log`, `C:\dist\devmount-support-console.exe`)
	if !strings.Contains(summary, "devmount-support-console.exe") {
		t.Fatalf("HelpSummary() missing support console path: %s", summary)
	}
}
