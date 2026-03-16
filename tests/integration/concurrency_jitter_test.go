package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"developer-mount/internal/client"
	"developer-mount/internal/protocol"
)

func TestMultiClientConcurrentCreateWriteRenameAndWatch(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	watcher, _ := newRecoveryEnv(t, root, 512, now, "watcher-client")
	sub, err := watcher.Subscribe(watcher.RootNodeID, true)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	const workerCount = 6
	var wg sync.WaitGroup
	errCh := make(chan error, workerCount)

	for i := 0; i < workerCount; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			cli := client.New(watcher.Addr)
			if err := cli.Connect(); err != nil {
				errCh <- fmt.Errorf("client-%d Connect(): %w", i, err)
				return
			}
			defer cli.Close()
			if _, err := cli.Hello(); err != nil {
				errCh <- fmt.Errorf("client-%d Hello(): %w", i, err)
				return
			}
			if _, err := cli.Auth("devmount-dev-token"); err != nil {
				errCh <- fmt.Errorf("client-%d Auth(): %w", i, err)
				return
			}
			if _, err := cli.CreateSession(fmt.Sprintf("worker-%d", i), 30); err != nil {
				errCh <- fmt.Errorf("client-%d CreateSession(): %w", i, err)
				return
			}

			tmpName := fmt.Sprintf("worker-%d.tmp", i)
			finalName := fmt.Sprintf("worker-%d.txt", i)
			createResp, err := cli.Create(cli.RootNodeID, tmpName, false)
			if err != nil {
				errCh <- fmt.Errorf("client-%d Create(%s): %w", i, tmpName, err)
				return
			}
			payload := []byte(fmt.Sprintf("payload-from-worker-%d", i))
			if _, err := cli.Write(createResp.HandleID, 0, payload[:8]); err != nil {
				errCh <- fmt.Errorf("client-%d Write(chunk-1): %w", i, err)
				return
			}
			if _, err := cli.Write(createResp.HandleID, 8, payload[8:]); err != nil {
				errCh <- fmt.Errorf("client-%d Write(chunk-2): %w", i, err)
				return
			}
			if _, err := cli.Flush(createResp.HandleID); err != nil {
				errCh <- fmt.Errorf("client-%d Flush(): %w", i, err)
				return
			}
			if _, err := cli.CloseHandle(createResp.HandleID); err != nil {
				errCh <- fmt.Errorf("client-%d CloseHandle(): %w", i, err)
				return
			}
			if _, err := cli.Rename(cli.RootNodeID, tmpName, cli.RootNodeID, finalName, false); err != nil {
				errCh <- fmt.Errorf("client-%d Rename(%s->%s): %w", i, tmpName, finalName, err)
				return
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}

	afterSeq := sub.StartSeq
	collected := make([]protocol.WatchEvent, 0, workerCount*4)
	deadline := time.Now().Add(3 * time.Second)
	for len(collected) < workerCount*4 && time.Now().Before(deadline) {
		poll, err := watcher.PollEvents(sub.WatchID, afterSeq, 128)
		if err != nil {
			t.Fatalf("PollEvents() error = %v", err)
		}
		if len(poll.Events) == 0 {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		collected = append(collected, poll.Events...)
		afterSeq = poll.Events[len(poll.Events)-1].EventSeq
	}
	if len(collected) < workerCount*4 {
		t.Fatalf("collected %d watch events, want at least %d", len(collected), workerCount*4)
	}

	for i := 0; i < workerCount; i++ {
		name := fmt.Sprintf("worker-%d.txt", i)
		lookup, err := watcher.Lookup(watcher.RootNodeID, name)
		if err != nil {
			t.Fatalf("Lookup(%s) error = %v", name, err)
		}
		openResp, err := watcher.OpenRead(lookup.Entry.NodeID)
		if err != nil {
			t.Fatalf("OpenRead(%s) error = %v", name, err)
		}
		readResp, err := watcher.Read(openResp.HandleID, 0, 128)
		if err != nil {
			t.Fatalf("Read(%s) error = %v", name, err)
		}
		if _, err := watcher.CloseHandle(openResp.HandleID); err != nil {
			t.Fatalf("CloseHandle(%s) error = %v", name, err)
		}
		if got, want := string(readResp.Data), fmt.Sprintf("payload-from-worker-%d", i); got != want {
			t.Fatalf("Read(%s) = %q, want %q", name, got, want)
		}
	}
}

func TestHeartbeatInterleavesWithFileOperations(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 3, 16, 13, 0, 0, 0, time.UTC)
	cli, _ := newRecoveryEnv(t, root, 128, now, "interleave-client")

	createResp, err := cli.Create(cli.RootNodeID, "interleave.txt", false)
	if err != nil {
		t.Fatalf("Create(interleave.txt) error = %v", err)
	}

	const iterations = 24
	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	payloads := make([]string, 0, iterations)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			if _, err := cli.Heartbeat(); err != nil {
				errCh <- fmt.Errorf("Heartbeat(iter=%d): %w", i, err)
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		offset := int64(0)
		for i := 0; i < iterations; i++ {
			chunk := fmt.Sprintf("chunk-%02d|", i)
			payloads = append(payloads, chunk)
			if _, err := cli.Write(createResp.HandleID, offset, []byte(chunk)); err != nil {
				errCh <- fmt.Errorf("Write(iter=%d): %w", i, err)
				return
			}
			offset += int64(len(chunk))
			if i%6 == 5 {
				if _, err := cli.Flush(createResp.HandleID); err != nil {
					errCh <- fmt.Errorf("Flush(iter=%d): %w", i, err)
					return
				}
			}
		}
	}()

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := cli.Flush(createResp.HandleID); err != nil {
		t.Fatalf("Flush(final) error = %v", err)
	}
	if _, err := cli.CloseHandle(createResp.HandleID); err != nil {
		t.Fatalf("CloseHandle() error = %v", err)
	}

	lookup, err := cli.Lookup(cli.RootNodeID, "interleave.txt")
	if err != nil {
		t.Fatalf("Lookup(interleave.txt) error = %v", err)
	}
	readHandle, err := cli.OpenRead(lookup.Entry.NodeID)
	if err != nil {
		t.Fatalf("OpenRead(interleave.txt) error = %v", err)
	}
	readResp, err := cli.Read(readHandle.HandleID, 0, 4096)
	if err != nil {
		t.Fatalf("Read(interleave.txt) error = %v", err)
	}
	if _, err := cli.CloseHandle(readHandle.HandleID); err != nil {
		t.Fatalf("CloseHandle(read-handle) error = %v", err)
	}
	want := strings.Join(payloads, "")
	if got := string(readResp.Data); got != want {
		t.Fatalf("Read(interleave.txt) = %q, want %q", got, want)
	}
}

func TestConnectionJitterRepeatedResumeAndRead(t *testing.T) {
	root := t.TempDir()
	content := []byte("jitter-hello")
	if err := os.WriteFile(filepath.Join(root, "stable.txt"), content, 0o644); err != nil {
		t.Fatalf("WriteFile(stable.txt) error = %v", err)
	}
	now := time.Date(2026, 3, 16, 14, 0, 0, 0, time.UTC)
	cli, _ := newRecoveryEnv(t, root, 128, now, "jitter-client")
	sessionID := cli.SessionID
	addr := cli.Addr
	if err := cli.Close(); err != nil {
		t.Fatalf("Close(initial client) error = %v", err)
	}

	for i := 0; i < 5; i++ {
		resumed := reconnectAndResume(t, addr, sessionID, "jitter-client")
		if _, err := resumed.Heartbeat(); err != nil {
			t.Fatalf("Heartbeat(iter=%d) error = %v", i, err)
		}
		lookup, err := resumed.Lookup(resumed.RootNodeID, "stable.txt")
		if err != nil {
			t.Fatalf("Lookup(iter=%d) error = %v", i, err)
		}
		openResp, err := resumed.OpenRead(lookup.Entry.NodeID)
		if err != nil {
			t.Fatalf("OpenRead(iter=%d) error = %v", i, err)
		}
		readResp, err := resumed.Read(openResp.HandleID, 0, 64)
		if err != nil {
			t.Fatalf("Read(iter=%d) error = %v", i, err)
		}
		if got := string(readResp.Data); got != string(content) {
			t.Fatalf("Read(iter=%d) = %q, want %q", i, got, string(content))
		}
		if _, err := resumed.CloseHandle(openResp.HandleID); err != nil {
			t.Fatalf("CloseHandle(iter=%d) error = %v", i, err)
		}
		if err := resumed.Close(); err != nil {
			t.Fatalf("Close(iter=%d) error = %v", i, err)
		}
		time.Sleep(time.Duration(i+1) * 5 * time.Millisecond)
	}
}
