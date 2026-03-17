package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"developer-mount/internal/server"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func pickFreeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr
}

func waitForTCP(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server did not listen on %s within %s", addr, timeout)
}

func waitForHTTPStatus(t *testing.T, url string, timeout time.Duration) []byte {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) // #nosec G107 -- local ephemeral test endpoint
		if err == nil {
			body, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr != nil {
				t.Fatalf("ReadAll(%s) error = %v", url, readErr)
			}
			if resp.StatusCode == http.StatusOK {
				return body
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("status endpoint %s did not respond within %s", url, timeout)
	return nil
}

func buildBinary(t *testing.T, repo, pkg, output string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", output, pkg)
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build %s error = %v\n%s", pkg, err, string(out))
	}
}

func TestServerAndClientBinarySmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping smoke test in short mode")
	}

	repo := repoRoot(t)
	binDir := t.TempDir()
	serverBin := filepath.Join(binDir, "devmount-server")
	clientBin := filepath.Join(binDir, "devmount-client")
	buildBinary(t, repo, "./cmd/devmount-server", serverBin)
	buildBinary(t, repo, "./cmd/devmount-client", clientBin)

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "smoke.txt"), []byte("smoke-data"), 0o644); err != nil {
		t.Fatalf("WriteFile(smoke.txt) error = %v", err)
	}
	serverAddr := pickFreeAddr(t)
	statusAddr := pickFreeAddr(t)
	configPath := filepath.Join(t.TempDir(), "devmount-smoke.json")
	cfg := server.ServerConfig{Addr: serverAddr, RootPath: root, AuthToken: "smoke-token", StatusAddr: statusAddr}
	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal(config) error = %v", err)
	}
	if err := os.WriteFile(configPath, cfgBytes, 0o644); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverCmd := exec.CommandContext(ctx, serverBin, "-config", configPath)
	serverCmd.Dir = repo
	var serverStdout, serverStderr bytes.Buffer
	serverCmd.Stdout = &serverStdout
	serverCmd.Stderr = &serverStderr
	if err := serverCmd.Start(); err != nil {
		t.Fatalf("server Start() error = %v", err)
	}
	defer func() {
		cancel()
		_ = serverCmd.Process.Kill()
		_, _ = serverCmd.Process.Wait()
	}()

	waitForTCP(t, serverAddr, 5*time.Second)
	statusBody := waitForHTTPStatus(t, fmt.Sprintf("http://%s/status", statusAddr), 5*time.Second)
	if !strings.Contains(string(statusBody), "devmount-server") {
		t.Fatalf("/status body = %s, want devmount-server", string(statusBody))
	}

	clientCtx, clientCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer clientCancel()
	clientCmd := exec.CommandContext(clientCtx, clientBin, "-addr", serverAddr, "-token", "smoke-token")
	clientCmd.Dir = repo
	output, err := clientCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("client CombinedOutput() error = %v\nstdout/stderr:\n%s\nserver stderr:\n%s", err, string(output), serverStderr.String())
	}
	outStr := string(output)
	for _, needle := range []string{"hello:", "auth:", "session:", "heartbeat:", "root entries"} {
		if !strings.Contains(outStr, needle) {
			t.Fatalf("client output missing %q\n%s", needle, outStr)
		}
	}
	if !strings.Contains(outStr, "smoke.txt") {
		t.Fatalf("client output missing smoke.txt\n%s", outStr)
	}
}
