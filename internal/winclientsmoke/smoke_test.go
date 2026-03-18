package winclientsmoke

import (
	"strings"
	"testing"
)

func TestDefaultExplorerSmokeMarkdown(t *testing.T) {
	scenarios := DefaultExplorerSmoke()
	if len(scenarios) < 4 {
		t.Fatalf("len(DefaultExplorerSmoke()) = %d", len(scenarios))
	}
	md := Markdown(scenarios)
	if !strings.Contains(md, "# Windows Explorer Smoke") || !strings.Contains(md, "explorer-mount-visible") {
		t.Fatalf("Markdown() output missing expected content: %s", md)
	}
	payload, err := JSON(scenarios)
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	if !strings.Contains(string(payload), "explorer-diagnostics") {
		t.Fatalf("JSON() missing scenario id: %s", string(payload))
	}
}
