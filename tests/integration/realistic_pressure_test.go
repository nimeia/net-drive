package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"developer-mount/internal/client"
	"developer-mount/internal/protocol"
)

type workloadMetrics struct {
	mu        sync.Mutex
	durations map[string][]time.Duration
	counts    map[string]int
}

func newWorkloadMetrics() *workloadMetrics {
	return &workloadMetrics{
		durations: make(map[string][]time.Duration),
		counts:    make(map[string]int),
	}
}

func (m *workloadMetrics) record(name string, started time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.durations[name] = append(m.durations[name], time.Since(started))
	m.counts[name]++
}

func percentile(durations []time.Duration, pct float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	copyDurations := append([]time.Duration(nil), durations...)
	sort.Slice(copyDurations, func(i, j int) bool { return copyDurations[i] < copyDurations[j] })
	idx := int(float64(len(copyDurations)-1) * pct)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(copyDurations) {
		idx = len(copyDurations) - 1
	}
	return copyDurations[idx]
}

func (m *workloadMetrics) log(t *testing.T, label string) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := make([]string, 0, len(m.durations))
	for key := range m.durations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		durations := m.durations[key]
		copyDurations := append([]time.Duration(nil), durations...)
		sort.Slice(copyDurations, func(i, j int) bool { return copyDurations[i] < copyDurations[j] })
		max := time.Duration(0)
		if len(copyDurations) > 0 {
			max = copyDurations[len(copyDurations)-1]
		}
		t.Logf("%s metric=%s count=%d p50=%s p95=%s max=%s", label, key, m.counts[key], percentile(copyDurations, 0.50), percentile(copyDurations, 0.95), max)
	}
}

func connectScenarioClient(t *testing.T, addr, clientInstanceID string) *client.Client {
	t.Helper()
	cli := client.New(addr)
	if err := cli.Connect(); err != nil {
		t.Fatalf("Connect(%s) error = %v", clientInstanceID, err)
	}
	if _, err := cli.Hello(); err != nil {
		_ = cli.Close()
		t.Fatalf("Hello(%s) error = %v", clientInstanceID, err)
	}
	if _, err := cli.Auth("devmount-dev-token"); err != nil {
		_ = cli.Close()
		t.Fatalf("Auth(%s) error = %v", clientInstanceID, err)
	}
	if _, err := cli.CreateSession(clientInstanceID, 30); err != nil {
		_ = cli.Close()
		t.Fatalf("CreateSession(%s) error = %v", clientInstanceID, err)
	}
	return cli
}

func TestRealisticMixedBrowseSaveWatchPressure(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"README.md":                          "# demo workspace\n",
		"src/main.go":                        "package main\nfunc main() {}\n",
		"src/lib/util.go":                    "package lib\nfunc Add(a, b int) int { return a + b }\n",
		"src/lib/util_test.go":               "package lib\n",
		"docs/architecture/overview.md":      "overview\n",
		"configs/devmount.json":              `{"root":"workspace"}`,
		"package.json":                       `{"name":"demo"}`,
		"scripts/bootstrap.sh":               "#!/usr/bin/env bash\necho bootstrap\n",
		"src/components/editor/profile.json": `{"hot":true}`,
	}
	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", rel, err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", rel, err)
		}
	}

	now := time.Date(2026, 3, 16, 15, 0, 0, 0, time.UTC)
	watcher, _ := newRecoveryEnv(t, root, 4096, now, "mixed-watcher")
	sub, err := watcher.Subscribe(watcher.RootNodeID, true)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	metrics := newWorkloadMetrics()
	start := time.Now()
	const browseWorkers = 4
	const browseIterations = 40
	const saveWorkers = 3
	const saveIterations = 10

	var wg sync.WaitGroup
	errCh := make(chan error, browseWorkers+saveWorkers+4)

	for i := 0; i < browseWorkers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			cli := connectScenarioClient(t, watcher.Addr, fmt.Sprintf("browse-%d", i))
			defer cli.Close()

			targets := []string{"README.md", "package.json", "src/main.go", "src/lib/util.go", "docs/architecture/overview.md"}
			dirs := []string{"", "src", "src/lib", "docs", "configs"}
			for iter := 0; iter < browseIterations; iter++ {
				name := targets[iter%len(targets)]
				dir := dirs[iter%len(dirs)]

				var parentID uint64 = cli.RootNodeID
				if dir != "" {
					segments := strings.Split(dir, "/")
					for _, segment := range segments {
						started := time.Now()
						lookupDir, err := cli.Lookup(parentID, segment)
						metrics.record("lookup_dir", started)
						if err != nil {
							errCh <- fmt.Errorf("browse-%d LookupDir(%s): %w", i, segment, err)
							return
						}
						parentID = lookupDir.Entry.NodeID
					}
				}

				started := time.Now()
				dirResp, err := cli.OpenDir(parentID)
				metrics.record("opendir", started)
				if err != nil {
					errCh <- fmt.Errorf("browse-%d OpenDir(%s): %w", i, dir, err)
					return
				}
				started = time.Now()
				listing, err := cli.ReadDir(dirResp.DirCursorID, 0, 64)
				metrics.record("readdir", started)
				if err != nil {
					errCh <- fmt.Errorf("browse-%d ReadDir(%s): %w", i, dir, err)
					return
				}
				if len(listing.Entries) == 0 {
					errCh <- fmt.Errorf("browse-%d ReadDir(%s): empty listing", i, dir)
					return
				}

				lookupParent := cli.RootNodeID
				base := filepath.Base(name)
				dirPart := filepath.Dir(name)
				if dirPart != "." {
					for _, segment := range strings.Split(filepath.ToSlash(dirPart), "/") {
						started = time.Now()
						lookupDir, err := cli.Lookup(lookupParent, segment)
						metrics.record("lookup_dir", started)
						if err != nil {
							errCh <- fmt.Errorf("browse-%d Lookup(%s): %w", i, segment, err)
							return
						}
						lookupParent = lookupDir.Entry.NodeID
					}
				}

				started = time.Now()
				lookup, err := cli.Lookup(lookupParent, base)
				metrics.record("lookup_file", started)
				if err != nil {
					errCh <- fmt.Errorf("browse-%d Lookup(%s): %w", i, name, err)
					return
				}
				started = time.Now()
				if _, err := cli.GetAttr(lookup.Entry.NodeID); err != nil {
					metrics.record("getattr", started)
					errCh <- fmt.Errorf("browse-%d GetAttr(%s): %w", i, name, err)
					return
				}
				metrics.record("getattr", started)

				started = time.Now()
				openResp, err := cli.OpenRead(lookup.Entry.NodeID)
				metrics.record("open_read", started)
				if err != nil {
					errCh <- fmt.Errorf("browse-%d OpenRead(%s): %w", i, name, err)
					return
				}
				started = time.Now()
				readResp, err := cli.Read(openResp.HandleID, 0, 256)
				metrics.record("read", started)
				if err != nil {
					errCh <- fmt.Errorf("browse-%d Read(%s): %w", i, name, err)
					return
				}
				if len(readResp.Data) == 0 {
					errCh <- fmt.Errorf("browse-%d Read(%s): empty payload", i, name)
					return
				}
				started = time.Now()
				if _, err := cli.CloseHandle(openResp.HandleID); err != nil {
					metrics.record("close_read", started)
					errCh <- fmt.Errorf("browse-%d CloseHandle(%s): %w", i, name, err)
					return
				}
				metrics.record("close_read", started)
			}
		}()
	}

	for i := 0; i < saveWorkers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			cli := connectScenarioClient(t, watcher.Addr, fmt.Sprintf("save-%d", i))
			defer cli.Close()

			for iter := 0; iter < saveIterations; iter++ {
				tmpName := fmt.Sprintf("save-%d-%02d.tmp", i, iter)
				finalName := fmt.Sprintf("save-%d-%02d.txt", i, iter)
				parts := []string{
					fmt.Sprintf("worker=%d\n", i),
					fmt.Sprintf("iter=%d\n", iter),
					"payload=line-1\n",
					"payload=line-2\n",
				}
				started := time.Now()
				createResp, err := cli.Create(cli.RootNodeID, tmpName, false)
				metrics.record("create", started)
				if err != nil {
					errCh <- fmt.Errorf("save-%d Create(%s): %w", i, tmpName, err)
					return
				}
				offset := int64(0)
				for _, part := range parts {
					started = time.Now()
					if _, err := cli.Write(createResp.HandleID, offset, []byte(part)); err != nil {
						metrics.record("write", started)
						errCh <- fmt.Errorf("save-%d Write(%s): %w", i, tmpName, err)
						return
					}
					metrics.record("write", started)
					offset += int64(len(part))
				}
				started = time.Now()
				if _, err := cli.Flush(createResp.HandleID); err != nil {
					metrics.record("flush", started)
					errCh <- fmt.Errorf("save-%d Flush(%s): %w", i, tmpName, err)
					return
				}
				metrics.record("flush", started)
				started = time.Now()
				if _, err := cli.CloseHandle(createResp.HandleID); err != nil {
					metrics.record("close_write", started)
					errCh <- fmt.Errorf("save-%d CloseHandle(%s): %w", i, tmpName, err)
					return
				}
				metrics.record("close_write", started)
				started = time.Now()
				if _, err := cli.Rename(cli.RootNodeID, tmpName, cli.RootNodeID, finalName, false); err != nil {
					metrics.record("rename", started)
					errCh <- fmt.Errorf("save-%d Rename(%s -> %s): %w", i, tmpName, finalName, err)
					return
				}
				metrics.record("rename", started)

				started = time.Now()
				lookup, err := cli.Lookup(cli.RootNodeID, finalName)
				metrics.record("lookup_saved", started)
				if err != nil {
					errCh <- fmt.Errorf("save-%d Lookup(%s): %w", i, finalName, err)
					return
				}
				started = time.Now()
				openResp, err := cli.OpenRead(lookup.Entry.NodeID)
				metrics.record("open_saved", started)
				if err != nil {
					errCh <- fmt.Errorf("save-%d OpenRead(%s): %w", i, finalName, err)
					return
				}
				started = time.Now()
				readResp, err := cli.Read(openResp.HandleID, 0, 256)
				metrics.record("read_saved", started)
				if err != nil {
					errCh <- fmt.Errorf("save-%d Read(%s): %w", i, finalName, err)
					return
				}
				started = time.Now()
				if _, err := cli.CloseHandle(openResp.HandleID); err != nil {
					metrics.record("close_saved", started)
					errCh <- fmt.Errorf("save-%d CloseHandle(saved %s): %w", i, finalName, err)
					return
				}
				metrics.record("close_saved", started)
				joined := strings.Join(parts, "")
				if got := string(readResp.Data); got != joined {
					errCh <- fmt.Errorf("save-%d Read(%s) = %q, want %q", i, finalName, got, joined)
					return
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		cli := connectScenarioClient(t, watcher.Addr, "heartbeat-worker")
		defer cli.Close()
		for i := 0; i < 48; i++ {
			started := time.Now()
			if _, err := cli.Heartbeat(); err != nil {
				metrics.record("heartbeat", started)
				errCh <- fmt.Errorf("heartbeat Heartbeat(iter=%d): %w", i, err)
				return
			}
			metrics.record("heartbeat", started)
			time.Sleep(3 * time.Millisecond)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		cli := connectScenarioClient(t, watcher.Addr, "resume-worker")
		sessionID := cli.SessionID
		addr := cli.Addr
		if err := cli.Close(); err != nil {
			errCh <- fmt.Errorf("resume Close(initial): %w", err)
			return
		}
		for i := 0; i < 6; i++ {
			started := time.Now()
			resumed := reconnectAndResume(t, addr, sessionID, "resume-worker")
			metrics.record("resume_connect", started)
			started = time.Now()
			lookup, err := resumed.Lookup(resumed.RootNodeID, "README.md")
			metrics.record("resume_lookup", started)
			if err != nil {
				errCh <- fmt.Errorf("resume Lookup(iter=%d): %w", i, err)
				return
			}
			started = time.Now()
			openResp, err := resumed.OpenRead(lookup.Entry.NodeID)
			metrics.record("resume_open", started)
			if err != nil {
				errCh <- fmt.Errorf("resume OpenRead(iter=%d): %w", i, err)
				return
			}
			started = time.Now()
			readResp, err := resumed.Read(openResp.HandleID, 0, 128)
			metrics.record("resume_read", started)
			if err != nil {
				errCh <- fmt.Errorf("resume Read(iter=%d): %w", i, err)
				return
			}
			if !strings.Contains(string(readResp.Data), "demo workspace") {
				errCh <- fmt.Errorf("resume Read(iter=%d): unexpected payload %q", i, string(readResp.Data))
				return
			}
			started = time.Now()
			if _, err := resumed.CloseHandle(openResp.HandleID); err != nil {
				metrics.record("resume_close", started)
				errCh <- fmt.Errorf("resume CloseHandle(iter=%d): %w", i, err)
				return
			}
			metrics.record("resume_close", started)
			if err := resumed.Close(); err != nil {
				errCh <- fmt.Errorf("resume Close(iter=%d): %w", i, err)
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}

	afterSeq := sub.StartSeq
	collected := make([]protocol.WatchEvent, 0, saveWorkers*saveIterations*4)
	deadline := time.Now().Add(3 * time.Second)
	for len(collected) < saveWorkers*saveIterations*4 && time.Now().Before(deadline) {
		startedPoll := time.Now()
		poll, err := watcher.PollEvents(sub.WatchID, afterSeq, 256)
		metrics.record("watch_poll", startedPoll)
		if err != nil {
			t.Fatalf("PollEvents() error = %v", err)
		}
		if len(poll.Events) == 0 {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		collected = append(collected, poll.Events...)
		afterSeq = poll.Events[len(poll.Events)-1].EventSeq
	}
	if len(collected) < saveWorkers*saveIterations*4 {
		t.Fatalf("watch events collected = %d, want at least %d", len(collected), saveWorkers*saveIterations*4)
	}

	for i := 0; i < saveWorkers; i++ {
		for iter := 0; iter < saveIterations; iter++ {
			name := fmt.Sprintf("save-%d-%02d.txt", i, iter)
			lookup, err := watcher.Lookup(watcher.RootNodeID, name)
			if err != nil {
				t.Fatalf("final Lookup(%s) error = %v", name, err)
			}
			openResp, err := watcher.OpenRead(lookup.Entry.NodeID)
			if err != nil {
				t.Fatalf("final OpenRead(%s) error = %v", name, err)
			}
			readResp, err := watcher.Read(openResp.HandleID, 0, 256)
			if err != nil {
				t.Fatalf("final Read(%s) error = %v", name, err)
			}
			if _, err := watcher.CloseHandle(openResp.HandleID); err != nil {
				t.Fatalf("final CloseHandle(%s) error = %v", name, err)
			}
			if !strings.Contains(string(readResp.Data), fmt.Sprintf("worker=%d", i)) {
				t.Fatalf("final Read(%s) missing worker marker: %q", name, string(readResp.Data))
			}
		}
	}

	elapsed := time.Since(start)
	metrics.log(t, "mixed-workload")
	t.Logf("mixed-workload elapsed=%s browse_ops=%d save_ops=%d watch_events=%d", elapsed, browseWorkers*browseIterations, saveWorkers*saveIterations, len(collected))
}
