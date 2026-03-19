package server

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"developer-mount/internal/protocol"
)

type watchSubscription struct {
	ID        uint64
	NodeID    uint64
	RelPath   string
	Recursive bool
	AckedSeq  uint64
}

type journalBroker struct {
	backend   *metadataBackend
	now       func() time.Time
	retention int

	nextSeq     atomic.Uint64
	nextWatchID atomic.Uint64

	mu      observedRWMutex
	events  []protocol.WatchEvent
	watches map[uint64]*watchSubscription
}

func newJournalBroker(backend *metadataBackend, now func() time.Time, retention int) *journalBroker {
	if retention <= 0 {
		retention = 256
	}
	j := &journalBroker{
		backend:   backend,
		now:       now,
		retention: retention,
		watches:   make(map[uint64]*watchSubscription),
	}
	j.nextWatchID.Store(5000)
	return j
}

func (j *journalBroker) latestSeq() uint64 {
	j.mu.RLock()
	defer j.mu.RUnlock()
	if len(j.events) == 0 {
		return 0
	}
	return j.events[len(j.events)-1].EventSeq
}

func (j *journalBroker) Subscribe(nodeID uint64, recursive bool) (uint64, uint64, error) {
	relPath, err := j.backend.RelPathByNodeID(nodeID)
	if err != nil {
		return 0, 0, err
	}
	watchID := j.nextWatchID.Add(1)
	startSeq := j.latestSeq()
	j.mu.Lock()
	j.watches[watchID] = &watchSubscription{ID: watchID, NodeID: nodeID, RelPath: relPath, Recursive: recursive, AckedSeq: startSeq}
	j.mu.Unlock()
	return watchID, startSeq, nil
}

func (j *journalBroker) Poll(watchID, afterSeq uint64, maxEvents uint32) (protocol.PollEventsResp, error) {
	watch, err := j.watchByID(watchID)
	if err != nil {
		return protocol.PollEventsResp{}, err
	}
	if maxEvents == 0 {
		maxEvents = 128
	}

	j.mu.RLock()
	defer j.mu.RUnlock()
	resp := protocol.PollEventsResp{LatestSeq: latestSeqFromSlice(j.events), AckedSeq: watch.AckedSeq}
	if len(j.events) == 0 {
		return resp, nil
	}
	oldestSeq := j.events[0].EventSeq
	if afterSeq+1 < oldestSeq {
		resp.Overflow = true
		resp.NeedsResync = true
		return resp, nil
	}
	for _, evt := range j.events {
		if evt.EventSeq <= afterSeq {
			continue
		}
		if !watchMatches(watch, evt) {
			continue
		}
		resp.Events = append(resp.Events, evt)
		if uint32(len(resp.Events)) >= maxEvents {
			break
		}
	}
	return resp, nil
}

func (j *journalBroker) Ack(watchID, eventSeq uint64) (uint64, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	watch, ok := j.watches[watchID]
	if !ok {
		return 0, fmt.Errorf("watch not found")
	}
	if eventSeq > watch.AckedSeq {
		watch.AckedSeq = eventSeq
	}
	return watch.AckedSeq, nil
}

func (j *journalBroker) Resync(watchID uint64) (protocol.ResyncResp, error) {
	watch, err := j.watchByID(watchID)
	if err != nil {
		return protocol.ResyncResp{}, err
	}
	entries, err := j.backend.SnapshotSubtree(watch.NodeID, watch.Recursive)
	if err != nil {
		return protocol.ResyncResp{}, err
	}
	return protocol.ResyncResp{WatchID: watchID, SnapshotSeq: j.latestSeq(), Entries: entries}, nil
}

func (j *journalBroker) Append(event protocol.WatchEvent) {
	event.EventSeq = j.nextSeq.Add(1)
	event.EventTime = protocol.NowRFC3339(j.now())
	if event.Name == "" && event.Path != "" {
		event.Name = filepath.Base(event.Path)
	}
	if event.OldName == "" && event.OldPath != "" {
		event.OldName = filepath.Base(event.OldPath)
	}
	j.mu.Lock()
	j.events = append(j.events, event)
	if len(j.events) > j.retention {
		j.events = append([]protocol.WatchEvent(nil), j.events[len(j.events)-j.retention:]...)
	}
	j.mu.Unlock()
}

func (j *journalBroker) watchByID(watchID uint64) (*watchSubscription, error) {
	j.mu.RLock()
	watch, ok := j.watches[watchID]
	j.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("watch not found")
	}
	copyWatch := *watch
	return &copyWatch, nil
}

func latestSeqFromSlice(events []protocol.WatchEvent) uint64 {
	if len(events) == 0 {
		return 0
	}
	return events[len(events)-1].EventSeq
}

func watchMatches(watch *watchSubscription, evt protocol.WatchEvent) bool {
	return pathMatchesWatch(watch.RelPath, watch.Recursive, evt.Path) || pathMatchesWatch(watch.RelPath, watch.Recursive, evt.OldPath)
}

func pathMatchesWatch(root string, recursive bool, path string) bool {
	if path == "" {
		return false
	}
	root = cleanEventPath(root)
	path = cleanEventPath(path)
	if root == "" {
		if recursive {
			return true
		}
		parent := cleanEventPath(filepath.ToSlash(filepath.Dir(path)))
		return parent == "." || parent == ""
	}
	if recursive {
		return path == root || strings.HasPrefix(path, root+"/")
	}
	if path == root {
		return true
	}
	parent := cleanEventPath(filepath.ToSlash(filepath.Dir(path)))
	return parent == root
}

func cleanEventPath(path string) string {
	path = filepath.ToSlash(path)
	path = strings.TrimPrefix(path, "./")
	if path == "." {
		return ""
	}
	return strings.Trim(path, "/")
}

func (j *journalBroker) Resubscribe(specs []protocol.ResubscribeSpec) ([]protocol.ResubscribeResult, error) {
	results := make([]protocol.ResubscribeResult, 0, len(specs))
	for _, spec := range specs {
		startSeq := spec.AfterSeq
		j.mu.RLock()
		oldWatch, ok := j.watches[spec.PreviousWatchID]
		j.mu.RUnlock()
		if ok {
			if spec.NodeID == 0 {
				spec.NodeID = oldWatch.NodeID
			}
			if startSeq < oldWatch.AckedSeq {
				startSeq = oldWatch.AckedSeq
			}
		}
		relPath, err := j.backend.RelPathByNodeID(spec.NodeID)
		if err != nil {
			results = append(results, protocol.ResubscribeResult{PreviousWatchID: spec.PreviousWatchID, Error: err.Error()})
			continue
		}
		watchID := j.nextWatchID.Add(1)
		j.mu.Lock()
		j.watches[watchID] = &watchSubscription{ID: watchID, NodeID: spec.NodeID, RelPath: relPath, Recursive: spec.Recursive, AckedSeq: startSeq}
		j.mu.Unlock()
		results = append(results, protocol.ResubscribeResult{PreviousWatchID: spec.PreviousWatchID, WatchID: watchID, StartSeq: startSeq, AckedSeq: startSeq})
	}
	return results, nil
}
