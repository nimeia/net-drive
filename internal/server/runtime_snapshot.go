package server

import "time"

type MetadataRuntimeSnapshot struct {
	Nodes          int                         `json:"nodes"`
	NodePaths      int                         `json:"node_paths"`
	DirCursors     int                         `json:"dir_cursors"`
	Handles        int                         `json:"handles"`
	AttrCache      int                         `json:"attr_cache"`
	NegativeCache  int                         `json:"negative_cache"`
	DirSnapshots   int                         `json:"dir_snapshots"`
	SmallFileCache int                         `json:"small_file_cache"`
	Locks          RWLockWaitSnapshot          `json:"locks"`
	Diagnostics    MetadataDiagnosticsSnapshot `json:"diagnostics"`
}

type SessionRuntimeSnapshot struct {
	Total   int                `json:"total"`
	Active  int                `json:"active"`
	Expired int                `json:"expired"`
	Locks   RWLockWaitSnapshot `json:"locks"`
}

type JournalRuntimeSnapshot struct {
	Events          int                `json:"events"`
	Watches         int                `json:"watches"`
	LatestSeq       uint64             `json:"latest_seq"`
	OldestSeq       uint64             `json:"oldest_seq"`
	MaxWatchBacklog int                `json:"max_watch_backlog"`
	TotalBacklog    int                `json:"total_backlog"`
	Locks           RWLockWaitSnapshot `json:"locks"`
}

type RuntimeSnapshot struct {
	At       time.Time               `json:"at"`
	Metadata MetadataRuntimeSnapshot `json:"metadata"`
	Sessions SessionRuntimeSnapshot  `json:"sessions"`
	Journal  JournalRuntimeSnapshot  `json:"journal"`
	Control  ControlRuntimeSnapshot  `json:"control"`
	Faults   FaultLogRuntimeSnapshot `json:"faults"`
}

func (b *metadataBackend) RuntimeSnapshot() MetadataRuntimeSnapshot {
	if b == nil {
		return MetadataRuntimeSnapshot{}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pruneExpiredCachesLocked(b.now())
	return MetadataRuntimeSnapshot{
		Nodes:          len(b.nodesByID),
		NodePaths:      len(b.nodesByPath),
		DirCursors:     len(b.cursors),
		Handles:        len(b.handles),
		AttrCache:      len(b.attrCache),
		NegativeCache:  len(b.negativeCache),
		DirSnapshots:   len(b.dirSnapshots),
		SmallFileCache: len(b.smallFileCache),
		Locks:          b.mu.Snapshot(),
		Diagnostics:    b.diag.snapshot(),
	}
}

func (m *SessionManager) RuntimeSnapshot(now time.Time) SessionRuntimeSnapshot {
	if m == nil {
		return SessionRuntimeSnapshot{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := SessionRuntimeSnapshot{Total: len(m.sessions), Locks: m.mu.Snapshot()}
	for _, s := range m.sessions {
		if s.State == "active" && !now.After(s.ExpiresAt) {
			out.Active++
		} else {
			out.Expired++
		}
	}
	return out
}

func (j *journalBroker) RuntimeSnapshot() JournalRuntimeSnapshot {
	if j == nil {
		return JournalRuntimeSnapshot{}
	}
	j.mu.RLock()
	defer j.mu.RUnlock()
	out := JournalRuntimeSnapshot{Events: len(j.events), Watches: len(j.watches), Locks: j.mu.Snapshot()}
	if len(j.events) > 0 {
		out.OldestSeq = j.events[0].EventSeq
		out.LatestSeq = j.events[len(j.events)-1].EventSeq
	}
	for _, watch := range j.watches {
		backlog := 0
		for _, evt := range j.events {
			if evt.EventSeq <= watch.AckedSeq {
				continue
			}
			if !watchMatches(watch, evt) {
				continue
			}
			backlog++
		}
		out.TotalBacklog += backlog
		if backlog > out.MaxWatchBacklog {
			out.MaxWatchBacklog = backlog
		}
	}
	return out
}

func (s *Server) SnapshotRuntime(now time.Time) RuntimeSnapshot {
	if s == nil {
		return RuntimeSnapshot{At: now}
	}
	return RuntimeSnapshot{
		At:       now,
		Metadata: s.Backend.RuntimeSnapshot(),
		Sessions: s.SessionManager.RuntimeSnapshot(now),
		Journal:  s.Journal.RuntimeSnapshot(),
		Control:  s.Control.Snapshot(),
		Faults:   s.Faults.Snapshot(),
	}
}
