package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"developer-mount/internal/client"
	"developer-mount/internal/protocol"
	"developer-mount/internal/server"
	"developer-mount/internal/transport"
)

type metricStore struct {
	mu        sync.Mutex
	durations map[string][]time.Duration
	counts    map[string]int64
}

func newMetricStore() *metricStore {
	return &metricStore{durations: map[string][]time.Duration{}, counts: map[string]int64{}}
}

func (m *metricStore) add(name string, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.durations[name] = append(m.durations[name], d)
	m.counts[name]++
}

type metricSummary struct {
	Name  string
	Count int64
	P50   time.Duration
	P95   time.Duration
	Max   time.Duration
}

func percentile(ds []time.Duration, p float64) time.Duration {
	if len(ds) == 0 {
		return 0
	}
	cp := append([]time.Duration(nil), ds...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := int(float64(len(cp)-1) * p)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

func (m *metricStore) summarize() []metricSummary {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := make([]string, 0, len(m.durations))
	for k := range m.durations {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]metricSummary, 0, len(keys))
	for _, k := range keys {
		cp := append([]time.Duration(nil), m.durations[k]...)
		sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
		max := time.Duration(0)
		if len(cp) > 0 {
			max = cp[len(cp)-1]
		}
		out = append(out, metricSummary{Name: k, Count: m.counts[k], P50: percentile(cp, 0.50), P95: percentile(cp, 0.95), Max: max})
	}
	return out
}

type sample struct {
	At                        time.Time
	Goroutines                int
	HeapAlloc                 uint64
	HeapObjects               uint64
	SessionsTotal             int
	SessionsActive            int
	SessionsExpired           int
	Nodes                     int
	NodePaths                 int
	DirCursors                int
	Handles                   int
	AttrCache                 int
	NegativeCache             int
	DirSnapshots              int
	SmallFileCache            int
	WatchEvents               int64
	Watches                   int
	Events                    int
	LatestSeq                 uint64
	OldestSeq                 uint64
	MaxBacklog                int
	TotalBacklog              int
	MetadataReadAcquires      uint64
	MetadataReadWaitOver50us  uint64
	MetadataReadWaitOver1ms   uint64
	MetadataReadTotalWaitNS   uint64
	MetadataReadMaxWaitNS     uint64
	MetadataWriteAcquires     uint64
	MetadataWriteWaitOver50us uint64
	MetadataWriteWaitOver1ms  uint64
	MetadataWriteTotalWaitNS  uint64
	MetadataWriteMaxWaitNS    uint64
	SessionReadAcquires       uint64
	SessionReadWaitOver50us   uint64
	SessionReadWaitOver1ms    uint64
	SessionReadTotalWaitNS    uint64
	SessionReadMaxWaitNS      uint64
	SessionWriteAcquires      uint64
	SessionWriteWaitOver50us  uint64
	SessionWriteWaitOver1ms   uint64
	SessionWriteTotalWaitNS   uint64
	SessionWriteMaxWaitNS     uint64
	JournalReadAcquires       uint64
	JournalReadWaitOver50us   uint64
	JournalReadWaitOver1ms    uint64
	JournalReadTotalWaitNS    uint64
	JournalReadMaxWaitNS      uint64
	JournalWriteAcquires      uint64
	JournalWriteWaitOver50us  uint64
	JournalWriteWaitOver1ms   uint64
	JournalWriteTotalWaitNS   uint64
	JournalWriteMaxWaitNS     uint64
	ControlHelloCount         uint64
	ControlHelloErrors        uint64
	ControlHelloMaxWaitNS     uint64
	ControlAuthCount          uint64
	ControlAuthErrors         uint64
	ControlAuthMaxWaitNS      uint64
	ControlCreateCount        uint64
	ControlCreateErrors       uint64
	ControlCreateMaxWaitNS    uint64
	ControlResumeCount        uint64
	ControlResumeErrors       uint64
	ControlResumeMaxWaitNS    uint64
	ControlHeartbeatCount     uint64
	ControlHeartbeatErrors    uint64
	ControlHeartbeatMaxWaitNS uint64
	FaultSuppressedNetClosed  uint64
	FaultSuppressedEOF        uint64
	FaultSuppressedUnexpected uint64
	FaultSuppressedBrokenPipe uint64
	FaultSuppressedConnReset  uint64
	FaultLogged               uint64
	Errors                    int64
}

type reportData struct {
	Duration       time.Duration
	BaseGoroutines int
	EndGoroutines  int
	Errors         int64
	WatchEvents    int64
	Samples        []sample
	Metrics        []metricSummary
}

func main() {
	duration := flag.Duration("duration", 3*time.Minute, "soak duration")
	sampleInterval := flag.Duration("sample-interval", time.Second, "runtime sample interval")
	browseWorkers := flag.Int("browse-workers", 4, "browse worker count")
	saveWorkers := flag.Int("save-workers", 3, "save worker count")
	heartbeatWorkers := flag.Int("heartbeat-workers", 2, "heartbeat worker count")
	resumeWorkers := flag.Int("resume-workers", 2, "resume worker count")
	enableSlowClient := flag.Bool("fault-slow-client", true, "enable partial-frame fault injection")
	enableHalfClose := flag.Bool("fault-half-close", true, "enable half-close fault injection")
	enableDelayedWrite := flag.Bool("fault-delayed-write", true, "enable delayed byte-wise frame writes")
	csvPath := flag.String("csv", filepath.ToSlash(filepath.Join("dist", "stress", "sampled-soak-samples.csv")), "csv output path")
	reportPath := flag.String("report", filepath.ToSlash(filepath.Join("dist", "stress", "sampled-soak-report.md")), "markdown report path")
	dryRun := flag.Bool("dry-run", false, "print planned configuration and exit")
	flag.Parse()

	if *browseWorkers+*saveWorkers+*heartbeatWorkers+*resumeWorkers <= 0 {
		panic("at least one worker must be enabled")
	}
	if *dryRun {
		fmt.Printf("duration=%s\n", *duration)
		fmt.Printf("sample_interval=%s\n", *sampleInterval)
		fmt.Printf("workers browse=%d save=%d heartbeat=%d resume=%d\n", *browseWorkers, *saveWorkers, *heartbeatWorkers, *resumeWorkers)
		fmt.Printf("faults slow_client=%v half_close=%v delayed_write=%v\n", *enableSlowClient, *enableHalfClose, *enableDelayedWrite)
		fmt.Printf("csv=%s\n", *csvPath)
		fmt.Printf("report=%s\n", *reportPath)
		return
	}

	must(os.MkdirAll(filepath.Dir(*csvPath), 0o755))
	must(os.MkdirAll(filepath.Dir(*reportPath), 0o755))

	root, err := os.MkdirTemp("", "devmount-sampled-soak-*")
	must(err)
	defer os.RemoveAll(root)
	seedWorkspace(root)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	must(err)
	srv := server.New(ln.Addr().String())
	srv.RootPath = root
	srv.JournalRetention = 8192
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	time.Sleep(50 * time.Millisecond)

	addr := ln.Addr().String()
	baseG := runtime.NumGoroutine()
	metrics := newMetricStore()
	var errorsSeen atomic.Int64
	var watchEvents atomic.Int64

	watcher, err := connectClient(addr, "sampled-soak-watcher", metrics)
	must(err)
	defer watcher.Close()
	sub, err := watcher.Subscribe(watcher.RootNodeID, true)
	must(err)

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()
	startedAt := time.Now()
	var wg sync.WaitGroup
	spawn := func(fn func(context.Context) error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "worker error: %v\n", err)
				errorsSeen.Add(1)
			}
		}()
	}

	for i := 0; i < *browseWorkers; i++ {
		id := fmt.Sprintf("browse-%d", i)
		spawn(func(ctx context.Context) error { return browseWorker(ctx, addr, id, metrics) })
	}
	for i := 0; i < *saveWorkers; i++ {
		id := fmt.Sprintf("save-%d", i)
		spawn(func(ctx context.Context) error { return saveWorker(ctx, addr, id, metrics) })
	}
	for i := 0; i < *heartbeatWorkers; i++ {
		id := fmt.Sprintf("heartbeat-%d", i)
		spawn(func(ctx context.Context) error { return heartbeatWorker(ctx, addr, id, metrics) })
	}
	for i := 0; i < *resumeWorkers; i++ {
		id := fmt.Sprintf("resume-%d", i)
		spawn(func(ctx context.Context) error { return resumeWorker(ctx, addr, id, metrics) })
	}
	spawn(func(ctx context.Context) error {
		return watchWorker(ctx, watcher, sub.WatchID, sub.StartSeq, metrics, &watchEvents)
	})
	if *enableSlowClient {
		spawn(func(ctx context.Context) error { return slowFrameWorker(ctx, addr, metrics) })
	}
	if *enableHalfClose {
		spawn(func(ctx context.Context) error { return halfCloseWorker(ctx, addr, metrics) })
	}
	if *enableDelayedWrite {
		spawn(func(ctx context.Context) error { return delayedFrameWorker(ctx, addr, metrics) })
	}

	ticker := time.NewTicker(*sampleInterval)
	defer ticker.Stop()
	samples := make([]sample, 0, int(duration.Seconds())+8)

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-ticker.C:
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			snap := srv.SnapshotRuntime(time.Now())
			samples = append(samples, sample{
				At:                        snap.At,
				Goroutines:                runtime.NumGoroutine(),
				HeapAlloc:                 ms.HeapAlloc,
				HeapObjects:               ms.HeapObjects,
				SessionsTotal:             snap.Sessions.Total,
				SessionsActive:            snap.Sessions.Active,
				SessionsExpired:           snap.Sessions.Expired,
				Nodes:                     snap.Metadata.Nodes,
				NodePaths:                 snap.Metadata.NodePaths,
				DirCursors:                snap.Metadata.DirCursors,
				Handles:                   snap.Metadata.Handles,
				AttrCache:                 snap.Metadata.AttrCache,
				NegativeCache:             snap.Metadata.NegativeCache,
				DirSnapshots:              snap.Metadata.DirSnapshots,
				SmallFileCache:            snap.Metadata.SmallFileCache,
				WatchEvents:               watchEvents.Load(),
				Watches:                   snap.Journal.Watches,
				Events:                    snap.Journal.Events,
				LatestSeq:                 snap.Journal.LatestSeq,
				OldestSeq:                 snap.Journal.OldestSeq,
				MaxBacklog:                snap.Journal.MaxWatchBacklog,
				TotalBacklog:              snap.Journal.TotalBacklog,
				MetadataReadAcquires:      snap.Metadata.Locks.Read.Acquires,
				MetadataReadWaitOver50us:  snap.Metadata.Locks.Read.WaitOver50us,
				MetadataReadWaitOver1ms:   snap.Metadata.Locks.Read.WaitOver1ms,
				MetadataReadTotalWaitNS:   uint64(snap.Metadata.Locks.Read.TotalWait),
				MetadataReadMaxWaitNS:     uint64(snap.Metadata.Locks.Read.MaxWait),
				MetadataWriteAcquires:     snap.Metadata.Locks.Write.Acquires,
				MetadataWriteWaitOver50us: snap.Metadata.Locks.Write.WaitOver50us,
				MetadataWriteWaitOver1ms:  snap.Metadata.Locks.Write.WaitOver1ms,
				MetadataWriteTotalWaitNS:  uint64(snap.Metadata.Locks.Write.TotalWait),
				MetadataWriteMaxWaitNS:    uint64(snap.Metadata.Locks.Write.MaxWait),
				SessionReadAcquires:       snap.Sessions.Locks.Read.Acquires,
				SessionReadWaitOver50us:   snap.Sessions.Locks.Read.WaitOver50us,
				SessionReadWaitOver1ms:    snap.Sessions.Locks.Read.WaitOver1ms,
				SessionReadTotalWaitNS:    uint64(snap.Sessions.Locks.Read.TotalWait),
				SessionReadMaxWaitNS:      uint64(snap.Sessions.Locks.Read.MaxWait),
				SessionWriteAcquires:      snap.Sessions.Locks.Write.Acquires,
				SessionWriteWaitOver50us:  snap.Sessions.Locks.Write.WaitOver50us,
				SessionWriteWaitOver1ms:   snap.Sessions.Locks.Write.WaitOver1ms,
				SessionWriteTotalWaitNS:   uint64(snap.Sessions.Locks.Write.TotalWait),
				SessionWriteMaxWaitNS:     uint64(snap.Sessions.Locks.Write.MaxWait),
				JournalReadAcquires:       snap.Journal.Locks.Read.Acquires,
				JournalReadWaitOver50us:   snap.Journal.Locks.Read.WaitOver50us,
				JournalReadWaitOver1ms:    snap.Journal.Locks.Read.WaitOver1ms,
				JournalReadTotalWaitNS:    uint64(snap.Journal.Locks.Read.TotalWait),
				JournalReadMaxWaitNS:      uint64(snap.Journal.Locks.Read.MaxWait),
				JournalWriteAcquires:      snap.Journal.Locks.Write.Acquires,
				JournalWriteWaitOver50us:  snap.Journal.Locks.Write.WaitOver50us,
				JournalWriteWaitOver1ms:   snap.Journal.Locks.Write.WaitOver1ms,
				JournalWriteTotalWaitNS:   uint64(snap.Journal.Locks.Write.TotalWait),
				JournalWriteMaxWaitNS:     uint64(snap.Journal.Locks.Write.MaxWait),
				ControlHelloCount:         snap.Control.Hello.Count,
				ControlHelloErrors:        snap.Control.Hello.Errors,
				ControlHelloMaxWaitNS:     uint64(snap.Control.Hello.MaxWait),
				ControlAuthCount:          snap.Control.Auth.Count,
				ControlAuthErrors:         snap.Control.Auth.Errors,
				ControlAuthMaxWaitNS:      uint64(snap.Control.Auth.MaxWait),
				ControlCreateCount:        snap.Control.CreateSession.Count,
				ControlCreateErrors:       snap.Control.CreateSession.Errors,
				ControlCreateMaxWaitNS:    uint64(snap.Control.CreateSession.MaxWait),
				ControlResumeCount:        snap.Control.ResumeSession.Count,
				ControlResumeErrors:       snap.Control.ResumeSession.Errors,
				ControlResumeMaxWaitNS:    uint64(snap.Control.ResumeSession.MaxWait),
				ControlHeartbeatCount:     snap.Control.Heartbeat.Count,
				ControlHeartbeatErrors:    snap.Control.Heartbeat.Errors,
				ControlHeartbeatMaxWaitNS: uint64(snap.Control.Heartbeat.MaxWait),
				FaultSuppressedNetClosed:  snap.Faults.SuppressedNetClosed,
				FaultSuppressedEOF:        snap.Faults.SuppressedEOF,
				FaultSuppressedUnexpected: snap.Faults.SuppressedUnexpectedEOF,
				FaultSuppressedBrokenPipe: snap.Faults.SuppressedBrokenPipe,
				FaultSuppressedConnReset:  snap.Faults.SuppressedConnReset,
				FaultLogged:               snap.Faults.Logged,
				Errors:                    errorsSeen.Load(),
			})
		}
	}

	wg.Wait()
	_ = ln.Close()
	select {
	case err := <-serveErr:
		if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			fmt.Fprintf(os.Stderr, "serve error: %v\n", err)
		}
	default:
	}
	time.Sleep(100 * time.Millisecond)
	endG := runtime.NumGoroutine()

	must(writeCSV(*csvPath, samples))
	must(writeReport(*reportPath, reportData{Duration: time.Since(startedAt), BaseGoroutines: baseG, EndGoroutines: endG, Errors: errorsSeen.Load(), WatchEvents: watchEvents.Load(), Samples: samples, Metrics: metrics.summarize()}))

	fmt.Printf("sampled soak finished: duration=%s errors=%d watch_events=%d goroutines_before=%d goroutines_after=%d\n", time.Since(startedAt), errorsSeen.Load(), watchEvents.Load(), baseG, endG)
	fmt.Printf("csv=%s\nreport=%s\n", *csvPath, *reportPath)
}

func seedWorkspace(root string) {
	files := map[string]string{
		"README.md":                          "# demo workspace\n",
		"src/main.go":                        "package main\nfunc main() {}\n",
		"src/lib/util.go":                    "package lib\nfunc Add(a, b int) int { return a + b }\n",
		"src/lib/util_test.go":               "package lib\n",
		"docs/architecture/overview.md":      strings.Repeat("overview\n", 64),
		"configs/devmount.json":              `{"root":"workspace"}`,
		"package.json":                       `{"name":"demo"}`,
		"scripts/bootstrap.sh":               "#!/usr/bin/env bash\necho bootstrap\n",
		"src/components/editor/profile.json": `{"hot":true}`,
	}
	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		must(os.MkdirAll(filepath.Dir(abs), 0o755))
		must(os.WriteFile(abs, []byte(content), 0o644))
	}
}

func connectClient(addr, clientInstanceID string, metrics *metricStore) (*client.Client, error) {
	cli := client.New(addr)
	if err := cli.Connect(); err != nil {
		return nil, err
	}
	started := time.Now()
	if _, err := cli.Hello(); err != nil {
		metrics.add("control_hello", time.Since(started))
		_ = cli.Close()
		return nil, err
	}
	metrics.add("control_hello", time.Since(started))
	started = time.Now()
	if _, err := cli.Auth("devmount-dev-token"); err != nil {
		metrics.add("control_auth", time.Since(started))
		_ = cli.Close()
		return nil, err
	}
	metrics.add("control_auth", time.Since(started))
	started = time.Now()
	if _, err := cli.CreateSession(clientInstanceID, 300); err != nil {
		metrics.add("control_create_session", time.Since(started))
		_ = cli.Close()
		return nil, err
	}
	metrics.add("control_create_session", time.Since(started))
	return cli, nil
}

func resumeClient(addr string, sessionID uint64, clientInstanceID string, metrics *metricStore) (*client.Client, error) {
	cli := client.New(addr)
	if err := cli.Connect(); err != nil {
		return nil, err
	}
	started := time.Now()
	if _, err := cli.Hello(); err != nil {
		metrics.add("control_hello", time.Since(started))
		_ = cli.Close()
		return nil, err
	}
	metrics.add("control_hello", time.Since(started))
	started = time.Now()
	if _, err := cli.Auth("devmount-dev-token"); err != nil {
		metrics.add("control_auth", time.Since(started))
		_ = cli.Close()
		return nil, err
	}
	metrics.add("control_auth", time.Since(started))
	started = time.Now()
	if _, err := cli.ResumeSession(sessionID, clientInstanceID); err != nil {
		metrics.add("control_resume_session", time.Since(started))
		_ = cli.Close()
		return nil, err
	}
	metrics.add("control_resume_session", time.Since(started))
	return cli, nil
}

func browseWorker(ctx context.Context, addr, id string, metrics *metricStore) error {
	cli, err := connectClient(addr, id, metrics)
	if err != nil {
		return err
	}
	defer cli.Close()
	targets := []string{"README.md", "package.json", "src/main.go", "src/lib/util.go", "docs/architecture/overview.md"}
	dirs := []string{"", "src", "src/lib", "docs", "configs"}
	iter := 0
	for ctx.Err() == nil {
		name := targets[iter%len(targets)]
		dir := dirs[iter%len(dirs)]
		var parentID uint64 = cli.RootNodeID
		if dir != "" {
			for _, segment := range strings.Split(dir, "/") {
				started := time.Now()
				lookupDir, err := cli.Lookup(parentID, segment)
				metrics.add("lookup_dir", time.Since(started))
				if err != nil {
					return fmt.Errorf("%s lookup_dir(%s): %w", id, segment, err)
				}
				parentID = lookupDir.Entry.NodeID
			}
		}
		started := time.Now()
		dirResp, err := cli.OpenDir(parentID)
		metrics.add("opendir", time.Since(started))
		if err != nil {
			return fmt.Errorf("%s opendir(%s): %w", id, dir, err)
		}
		started = time.Now()
		listing, err := cli.ReadDir(dirResp.DirCursorID, 0, 64)
		metrics.add("readdir", time.Since(started))
		if err != nil {
			return fmt.Errorf("%s readdir(%s): %w", id, dir, err)
		}
		if len(listing.Entries) == 0 {
			return fmt.Errorf("%s readdir(%s): empty listing", id, dir)
		}
		lookupParent := cli.RootNodeID
		base := filepath.Base(name)
		dirPart := filepath.Dir(name)
		if dirPart != "." {
			for _, segment := range strings.Split(filepath.ToSlash(dirPart), "/") {
				started = time.Now()
				lookupDir, err := cli.Lookup(lookupParent, segment)
				metrics.add("lookup_dir", time.Since(started))
				if err != nil {
					return fmt.Errorf("%s lookup(%s): %w", id, segment, err)
				}
				lookupParent = lookupDir.Entry.NodeID
			}
		}
		started = time.Now()
		lookup, err := cli.Lookup(lookupParent, base)
		metrics.add("lookup_file", time.Since(started))
		if err != nil {
			return fmt.Errorf("%s lookup_file(%s): %w", id, name, err)
		}
		started = time.Now()
		if _, err := cli.GetAttr(lookup.Entry.NodeID); err != nil {
			metrics.add("getattr", time.Since(started))
			return fmt.Errorf("%s getattr(%s): %w", id, name, err)
		}
		metrics.add("getattr", time.Since(started))
		started = time.Now()
		openResp, err := cli.OpenRead(lookup.Entry.NodeID)
		metrics.add("open_read", time.Since(started))
		if err != nil {
			return fmt.Errorf("%s open_read(%s): %w", id, name, err)
		}
		started = time.Now()
		readResp, err := cli.Read(openResp.HandleID, 0, 256)
		metrics.add("read", time.Since(started))
		if err != nil {
			return fmt.Errorf("%s read(%s): %w", id, name, err)
		}
		if len(readResp.Data) == 0 {
			return fmt.Errorf("%s read(%s): empty payload", id, name)
		}
		started = time.Now()
		if _, err := cli.CloseHandle(openResp.HandleID); err != nil {
			metrics.add("close_read", time.Since(started))
			return fmt.Errorf("%s close_read(%s): %w", id, name, err)
		}
		metrics.add("close_read", time.Since(started))
		iter++
	}
	return nil
}

func saveWorker(ctx context.Context, addr, id string, metrics *metricStore) error {
	cli, err := connectClient(addr, id, metrics)
	if err != nil {
		return err
	}
	defer cli.Close()
	iter := 0
	for ctx.Err() == nil {
		tmpName := fmt.Sprintf("%s-%04d.tmp", id, iter)
		finalName := fmt.Sprintf("%s-%04d.txt", id, iter)
		parts := []string{fmt.Sprintf("worker=%s\n", id), fmt.Sprintf("iter=%d\n", iter), "payload=line-1\n", "payload=line-2\n"}
		started := time.Now()
		createResp, err := cli.Create(cli.RootNodeID, tmpName, false)
		metrics.add("create", time.Since(started))
		if err != nil {
			return fmt.Errorf("%s create(%s): %w", id, tmpName, err)
		}
		offset := int64(0)
		for _, part := range parts {
			started = time.Now()
			if _, err := cli.Write(createResp.HandleID, offset, []byte(part)); err != nil {
				metrics.add("write", time.Since(started))
				return fmt.Errorf("%s write(%s): %w", id, tmpName, err)
			}
			metrics.add("write", time.Since(started))
			offset += int64(len(part))
		}
		started = time.Now()
		if _, err := cli.Flush(createResp.HandleID); err != nil {
			metrics.add("flush", time.Since(started))
			return fmt.Errorf("%s flush(%s): %w", id, tmpName, err)
		}
		metrics.add("flush", time.Since(started))
		started = time.Now()
		if _, err := cli.CloseHandle(createResp.HandleID); err != nil {
			metrics.add("close_write", time.Since(started))
			return fmt.Errorf("%s close_write(%s): %w", id, tmpName, err)
		}
		metrics.add("close_write", time.Since(started))
		started = time.Now()
		if _, err := cli.Rename(cli.RootNodeID, tmpName, cli.RootNodeID, finalName, false); err != nil {
			metrics.add("rename", time.Since(started))
			return fmt.Errorf("%s rename(%s->%s): %w", id, tmpName, finalName, err)
		}
		metrics.add("rename", time.Since(started))
		started = time.Now()
		lookup, err := cli.Lookup(cli.RootNodeID, finalName)
		metrics.add("lookup_saved", time.Since(started))
		if err != nil {
			return fmt.Errorf("%s lookup_saved(%s): %w", id, finalName, err)
		}
		started = time.Now()
		openResp, err := cli.OpenRead(lookup.Entry.NodeID)
		metrics.add("open_saved", time.Since(started))
		if err != nil {
			return fmt.Errorf("%s open_saved(%s): %w", id, finalName, err)
		}
		started = time.Now()
		readResp, err := cli.Read(openResp.HandleID, 0, 4096)
		metrics.add("read_saved", time.Since(started))
		if err != nil {
			return fmt.Errorf("%s read_saved(%s): %w", id, finalName, err)
		}
		if !strings.Contains(string(readResp.Data), fmt.Sprintf("worker=%s", id)) {
			return fmt.Errorf("%s read_saved(%s): verification failed", id, finalName)
		}
		started = time.Now()
		if _, err := cli.CloseHandle(openResp.HandleID); err != nil {
			metrics.add("close_saved", time.Since(started))
			return fmt.Errorf("%s close_saved(%s): %w", id, finalName, err)
		}
		metrics.add("close_saved", time.Since(started))
		iter++
	}
	return nil
}

func heartbeatWorker(ctx context.Context, addr, id string, metrics *metricStore) error {
	cli, err := connectClient(addr, id, metrics)
	if err != nil {
		return err
	}
	defer cli.Close()
	for ctx.Err() == nil {
		started := time.Now()
		if _, err := cli.Heartbeat(); err != nil {
			metrics.add("heartbeat", time.Since(started))
			return fmt.Errorf("%s heartbeat: %w", id, err)
		}
		metrics.add("heartbeat", time.Since(started))
		time.Sleep(20 * time.Millisecond)
	}
	return nil
}

func resumeWorker(ctx context.Context, addr, id string, metrics *metricStore) error {
	cli, err := connectClient(addr, id, metrics)
	if err != nil {
		return err
	}
	sessionID := cli.SessionID
	_ = cli.Close()
	for ctx.Err() == nil {
		started := time.Now()
		resumed, err := resumeClient(addr, sessionID, id, metrics)
		metrics.add("resume_connect", time.Since(started))
		if err != nil {
			return fmt.Errorf("%s resume_connect: %w", id, err)
		}
		started = time.Now()
		lookup, err := resumed.Lookup(resumed.RootNodeID, "README.md")
		metrics.add("resume_lookup", time.Since(started))
		if err != nil {
			_ = resumed.Close()
			return fmt.Errorf("%s resume_lookup: %w", id, err)
		}
		started = time.Now()
		openResp, err := resumed.OpenRead(lookup.Entry.NodeID)
		metrics.add("resume_open", time.Since(started))
		if err != nil {
			_ = resumed.Close()
			return fmt.Errorf("%s resume_open: %w", id, err)
		}
		started = time.Now()
		readResp, err := resumed.Read(openResp.HandleID, 0, 256)
		metrics.add("resume_read", time.Since(started))
		if err != nil {
			_ = resumed.Close()
			return fmt.Errorf("%s resume_read: %w", id, err)
		}
		if !strings.Contains(string(readResp.Data), "demo workspace") {
			_ = resumed.Close()
			return fmt.Errorf("%s resume_read: verification failed", id)
		}
		started = time.Now()
		if _, err := resumed.CloseHandle(openResp.HandleID); err != nil {
			metrics.add("resume_close", time.Since(started))
			_ = resumed.Close()
			return fmt.Errorf("%s resume_close: %w", id, err)
		}
		metrics.add("resume_close", time.Since(started))
		_ = resumed.Close()
		time.Sleep(15 * time.Millisecond)
	}
	return nil
}

func watchWorker(ctx context.Context, cli *client.Client, watchID, after uint64, metrics *metricStore, counter *atomic.Int64) error {
	for ctx.Err() == nil {
		started := time.Now()
		poll, err := cli.PollEvents(watchID, after, 512)
		metrics.add("watch_poll", time.Since(started))
		if err != nil {
			return err
		}
		if len(poll.Events) == 0 {
			time.Sleep(25 * time.Millisecond)
			continue
		}
		counter.Add(int64(len(poll.Events)))
		after = poll.Events[len(poll.Events)-1].EventSeq
		if _, err := cli.AckEvents(watchID, after); err != nil {
			return err
		}
	}
	return nil
}

func slowFrameWorker(ctx context.Context, addr string, metrics *metricStore) error {
	reqID := uint64(900000)
	for ctx.Err() == nil {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return err
		}
		frame, err := transport.EncodeFrame(protocol.Header{Channel: protocol.ChannelControl, Opcode: protocol.OpcodeHelloReq, Flags: protocol.FlagRequest, RequestID: reqID}, protocol.HelloReq{ClientName: "fault-slow", ClientVersion: "0", SupportedProtocolVersions: []uint8{protocol.Version}, Capabilities: protocol.DefaultCapabilities()})
		if err != nil {
			_ = conn.Close()
			return err
		}
		limit := len(frame) / 3
		if limit < 8 {
			limit = len(frame) / 2
		}
		started := time.Now()
		_, err = conn.Write(frame[:limit])
		metrics.add("fault_partial_write", time.Since(started))
		_ = conn.Close()
		if err != nil {
			return err
		}
		reqID++
		time.Sleep(35 * time.Millisecond)
	}
	return nil
}

func halfCloseWorker(ctx context.Context, addr string, metrics *metricStore) error {
	reqID := uint64(910000)
	for ctx.Err() == nil {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return err
		}
		frame, err := transport.EncodeFrame(protocol.Header{Channel: protocol.ChannelControl, Opcode: protocol.OpcodeHelloReq, Flags: protocol.FlagRequest, RequestID: reqID}, protocol.HelloReq{ClientName: "fault-halfclose", ClientVersion: "0", SupportedProtocolVersions: []uint8{protocol.Version}, Capabilities: protocol.DefaultCapabilities()})
		if err != nil {
			_ = conn.Close()
			return err
		}
		started := time.Now()
		_, err = conn.Write(frame)
		metrics.add("fault_halfclose_write", time.Since(started))
		if err != nil {
			_ = conn.Close()
			return err
		}
		if tcp, ok := conn.(*net.TCPConn); ok {
			_ = tcp.CloseWrite()
		}
		_ = conn.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
		_, _, _ = transport.DecodeFrame(conn)
		_ = conn.Close()
		reqID++
		time.Sleep(40 * time.Millisecond)
	}
	return nil
}

func delayedFrameWorker(ctx context.Context, addr string, metrics *metricStore) error {
	reqID := uint64(920000)
	for ctx.Err() == nil {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return err
		}
		frame, err := transport.EncodeFrame(protocol.Header{Channel: protocol.ChannelControl, Opcode: protocol.OpcodeHelloReq, Flags: protocol.FlagRequest, RequestID: reqID}, protocol.HelloReq{ClientName: "fault-delayed", ClientVersion: "0", SupportedProtocolVersions: []uint8{protocol.Version}, Capabilities: protocol.DefaultCapabilities()})
		if err != nil {
			_ = conn.Close()
			return err
		}
		started := time.Now()
		for _, b := range frame {
			if _, err := conn.Write([]byte{b}); err != nil {
				_ = conn.Close()
				return err
			}
			time.Sleep(2 * time.Millisecond)
		}
		metrics.add("fault_delayed_write", time.Since(started))
		_ = conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		_, _, _ = transport.DecodeFrame(conn)
		_ = conn.Close()
		reqID++
		time.Sleep(25 * time.Millisecond)
	}
	return nil
}

func writeCSV(path string, samples []sample) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	header := []string{"at", "goroutines", "heap_alloc", "heap_objects", "sessions_total", "sessions_active", "sessions_expired", "nodes", "node_paths", "dir_cursors", "handles", "attr_cache", "negative_cache", "dir_snapshots", "small_file_cache", "watch_events", "watches", "events", "latest_seq", "oldest_seq", "max_backlog", "total_backlog", "metadata_read_acquires", "metadata_read_wait_over_50us", "metadata_read_wait_over_1ms", "metadata_read_total_wait_ns", "metadata_read_max_wait_ns", "metadata_write_acquires", "metadata_write_wait_over_50us", "metadata_write_wait_over_1ms", "metadata_write_total_wait_ns", "metadata_write_max_wait_ns", "session_read_acquires", "session_read_wait_over_50us", "session_read_wait_over_1ms", "session_read_total_wait_ns", "session_read_max_wait_ns", "session_write_acquires", "session_write_wait_over_50us", "session_write_wait_over_1ms", "session_write_total_wait_ns", "session_write_max_wait_ns", "journal_read_acquires", "journal_read_wait_over_50us", "journal_read_wait_over_1ms", "journal_read_total_wait_ns", "journal_read_max_wait_ns", "journal_write_acquires", "journal_write_wait_over_50us", "journal_write_wait_over_1ms", "journal_write_total_wait_ns", "journal_write_max_wait_ns", "control_hello_count", "control_hello_errors", "control_hello_max_wait_ns", "control_auth_count", "control_auth_errors", "control_auth_max_wait_ns", "control_create_count", "control_create_errors", "control_create_max_wait_ns", "control_resume_count", "control_resume_errors", "control_resume_max_wait_ns", "control_heartbeat_count", "control_heartbeat_errors", "control_heartbeat_max_wait_ns", "fault_suppressed_net_closed", "fault_suppressed_eof", "fault_suppressed_unexpected_eof", "fault_suppressed_broken_pipe", "fault_suppressed_conn_reset", "fault_logged", "errors"}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, s := range samples {
		row := []string{s.At.Format(time.RFC3339), fmt.Sprint(s.Goroutines), fmt.Sprint(s.HeapAlloc), fmt.Sprint(s.HeapObjects), fmt.Sprint(s.SessionsTotal), fmt.Sprint(s.SessionsActive), fmt.Sprint(s.SessionsExpired), fmt.Sprint(s.Nodes), fmt.Sprint(s.NodePaths), fmt.Sprint(s.DirCursors), fmt.Sprint(s.Handles), fmt.Sprint(s.AttrCache), fmt.Sprint(s.NegativeCache), fmt.Sprint(s.DirSnapshots), fmt.Sprint(s.SmallFileCache), fmt.Sprint(s.WatchEvents), fmt.Sprint(s.Watches), fmt.Sprint(s.Events), fmt.Sprint(s.LatestSeq), fmt.Sprint(s.OldestSeq), fmt.Sprint(s.MaxBacklog), fmt.Sprint(s.TotalBacklog), fmt.Sprint(s.MetadataReadAcquires), fmt.Sprint(s.MetadataReadWaitOver50us), fmt.Sprint(s.MetadataReadWaitOver1ms), fmt.Sprint(s.MetadataReadTotalWaitNS), fmt.Sprint(s.MetadataReadMaxWaitNS), fmt.Sprint(s.MetadataWriteAcquires), fmt.Sprint(s.MetadataWriteWaitOver50us), fmt.Sprint(s.MetadataWriteWaitOver1ms), fmt.Sprint(s.MetadataWriteTotalWaitNS), fmt.Sprint(s.MetadataWriteMaxWaitNS), fmt.Sprint(s.SessionReadAcquires), fmt.Sprint(s.SessionReadWaitOver50us), fmt.Sprint(s.SessionReadWaitOver1ms), fmt.Sprint(s.SessionReadTotalWaitNS), fmt.Sprint(s.SessionReadMaxWaitNS), fmt.Sprint(s.SessionWriteAcquires), fmt.Sprint(s.SessionWriteWaitOver50us), fmt.Sprint(s.SessionWriteWaitOver1ms), fmt.Sprint(s.SessionWriteTotalWaitNS), fmt.Sprint(s.SessionWriteMaxWaitNS), fmt.Sprint(s.JournalReadAcquires), fmt.Sprint(s.JournalReadWaitOver50us), fmt.Sprint(s.JournalReadWaitOver1ms), fmt.Sprint(s.JournalReadTotalWaitNS), fmt.Sprint(s.JournalReadMaxWaitNS), fmt.Sprint(s.JournalWriteAcquires), fmt.Sprint(s.JournalWriteWaitOver50us), fmt.Sprint(s.JournalWriteWaitOver1ms), fmt.Sprint(s.JournalWriteTotalWaitNS), fmt.Sprint(s.JournalWriteMaxWaitNS), fmt.Sprint(s.ControlHelloCount), fmt.Sprint(s.ControlHelloErrors), fmt.Sprint(s.ControlHelloMaxWaitNS), fmt.Sprint(s.ControlAuthCount), fmt.Sprint(s.ControlAuthErrors), fmt.Sprint(s.ControlAuthMaxWaitNS), fmt.Sprint(s.ControlCreateCount), fmt.Sprint(s.ControlCreateErrors), fmt.Sprint(s.ControlCreateMaxWaitNS), fmt.Sprint(s.ControlResumeCount), fmt.Sprint(s.ControlResumeErrors), fmt.Sprint(s.ControlResumeMaxWaitNS), fmt.Sprint(s.ControlHeartbeatCount), fmt.Sprint(s.ControlHeartbeatErrors), fmt.Sprint(s.ControlHeartbeatMaxWaitNS), fmt.Sprint(s.FaultSuppressedNetClosed), fmt.Sprint(s.FaultSuppressedEOF), fmt.Sprint(s.FaultSuppressedUnexpected), fmt.Sprint(s.FaultSuppressedBrokenPipe), fmt.Sprint(s.FaultSuppressedConnReset), fmt.Sprint(s.FaultLogged), fmt.Sprint(s.Errors)}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return w.Error()
}

func writeReport(path string, data reportData) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintf(f, "# Iter 47 sampled soak report\n\n")
	fmt.Fprintf(f, "- duration: %s\n", data.Duration)
	fmt.Fprintf(f, "- errors: %d\n", data.Errors)
	fmt.Fprintf(f, "- watch events: %d\n", data.WatchEvents)
	fmt.Fprintf(f, "- goroutines before: %d\n", data.BaseGoroutines)
	fmt.Fprintf(f, "- goroutines after: %d\n\n", data.EndGoroutines)
	if len(data.Samples) > 0 {
		first := data.Samples[0]
		last := data.Samples[len(data.Samples)-1]
		peakG := first.Goroutines
		peakHeap := first.HeapAlloc
		peakHandles := first.Handles
		peakBacklog := first.MaxBacklog
		peakReadWait50us := first.MetadataReadWaitOver50us
		peakReadWait1ms := first.MetadataReadWaitOver1ms
		peakWriteWait50us := first.MetadataWriteWaitOver50us
		peakWriteWait1ms := first.MetadataWriteWaitOver1ms
		for _, s := range data.Samples {
			if s.Goroutines > peakG {
				peakG = s.Goroutines
			}
			if s.HeapAlloc > peakHeap {
				peakHeap = s.HeapAlloc
			}
			if s.Handles > peakHandles {
				peakHandles = s.Handles
			}
			if s.MaxBacklog > peakBacklog {
				peakBacklog = s.MaxBacklog
			}
			if s.MetadataReadWaitOver50us > peakReadWait50us {
				peakReadWait50us = s.MetadataReadWaitOver50us
			}
			if s.MetadataReadWaitOver1ms > peakReadWait1ms {
				peakReadWait1ms = s.MetadataReadWaitOver1ms
			}
			if s.MetadataWriteWaitOver50us > peakWriteWait50us {
				peakWriteWait50us = s.MetadataWriteWaitOver50us
			}
			if s.MetadataWriteWaitOver1ms > peakWriteWait1ms {
				peakWriteWait1ms = s.MetadataWriteWaitOver1ms
			}
		}
		fmt.Fprintf(f, "## runtime snapshot summary\n\n")
		fmt.Fprintf(f, "- samples: %d\n", len(data.Samples))
		fmt.Fprintf(f, "- goroutines: first=%d peak=%d last=%d\n", first.Goroutines, peakG, last.Goroutines)
		fmt.Fprintf(f, "- heap alloc bytes: first=%d peak=%d last=%d\n", first.HeapAlloc, peakHeap, last.HeapAlloc)
		fmt.Fprintf(f, "- active sessions: first=%d last=%d\n", first.SessionsActive, last.SessionsActive)
		fmt.Fprintf(f, "- handles: first=%d peak=%d last=%d\n", first.Handles, peakHandles, last.Handles)
		fmt.Fprintf(f, "- max watch backlog: first=%d peak=%d last=%d\n", first.MaxBacklog, peakBacklog, last.MaxBacklog)
		fmt.Fprintf(f, "- metadata read lock waits >50us: first=%d peak=%d last=%d\n", first.MetadataReadWaitOver50us, peakReadWait50us, last.MetadataReadWaitOver50us)
		fmt.Fprintf(f, "- metadata read lock waits >1ms: first=%d peak=%d last=%d\n", first.MetadataReadWaitOver1ms, peakReadWait1ms, last.MetadataReadWaitOver1ms)
		fmt.Fprintf(f, "- metadata write lock waits >50us: first=%d peak=%d last=%d\n", first.MetadataWriteWaitOver50us, peakWriteWait50us, last.MetadataWriteWaitOver50us)
		fmt.Fprintf(f, "- metadata write lock waits >1ms: first=%d peak=%d last=%d\n", first.MetadataWriteWaitOver1ms, peakWriteWait1ms, last.MetadataWriteWaitOver1ms)
		fmt.Fprintf(f, "- session write lock waits >1ms: first=%d last=%d\n", first.SessionWriteWaitOver1ms, last.SessionWriteWaitOver1ms)
		fmt.Fprintf(f, "- journal write lock waits >1ms: first=%d last=%d\n", first.JournalWriteWaitOver1ms, last.JournalWriteWaitOver1ms)
		fmt.Fprintf(f, "- control hello count/errors/max: %d/%d/%s\n", last.ControlHelloCount, last.ControlHelloErrors, time.Duration(last.ControlHelloMaxWaitNS))
		fmt.Fprintf(f, "- control auth count/errors/max: %d/%d/%s\n", last.ControlAuthCount, last.ControlAuthErrors, time.Duration(last.ControlAuthMaxWaitNS))
		fmt.Fprintf(f, "- control create count/errors/max: %d/%d/%s\n", last.ControlCreateCount, last.ControlCreateErrors, time.Duration(last.ControlCreateMaxWaitNS))
		fmt.Fprintf(f, "- control resume count/errors/max: %d/%d/%s\n", last.ControlResumeCount, last.ControlResumeErrors, time.Duration(last.ControlResumeMaxWaitNS))
		fmt.Fprintf(f, "- control heartbeat count/errors/max: %d/%d/%s\n", last.ControlHeartbeatCount, last.ControlHeartbeatErrors, time.Duration(last.ControlHeartbeatMaxWaitNS))
		fmt.Fprintf(f, "- fault log counters suppressed(net_closed/eof/unexpected/broken_pipe/conn_reset)=%d/%d/%d/%d/%d logged=%d\n\n", last.FaultSuppressedNetClosed, last.FaultSuppressedEOF, last.FaultSuppressedUnexpected, last.FaultSuppressedBrokenPipe, last.FaultSuppressedConnReset, last.FaultLogged)
	}
	fmt.Fprintf(f, "## latency summary\n\n")
	fmt.Fprintf(f, "| metric | count | p50 | p95 | max |\n")
	fmt.Fprintf(f, "|---|---:|---:|---:|---:|\n")
	for _, m := range data.Metrics {
		fmt.Fprintf(f, "| %s | %d | %s | %s | %s |\n", m.Name, m.Count, m.P50, m.P95, m.Max)
	}
	return nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
