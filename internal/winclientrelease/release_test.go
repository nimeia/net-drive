package winclientrelease

import (
	"strings"
	"testing"

	"developer-mount/internal/winclientsmoke"
	"developer-mount/internal/winfsp"
)

func TestManifestMarkdownChecklist(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{EffectiveBackend: "winfsp-dispatcher-v1", DispatcherReady: true, CallbackBridgeReady: true, ServiceLoopReady: true})
	matrix := winclientsmoke.DefaultExplorerRequestMatrix(table)
	manifest := NewManifest("1.2.3", []Artifact{{Name: "client", Kind: "exe", Path: "dist/client.exe"}}, table, matrix, winclientsmoke.DefaultExplorerSmoke())
	md := manifest.MarkdownChecklist()
	for _, want := range []string{"windows release validation", "1.2.3", "native callback", "explorer smoke", "backfill windows-host-validation-result-template.json"} {
		if !strings.Contains(strings.ToLower(md), want) {
			t.Fatalf("checklist missing %q", want)
		}
	}
}
