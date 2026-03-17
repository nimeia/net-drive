package server

import (
	"fmt"
	"testing"
	"time"

	"developer-mount/internal/protocol"
)

func BenchmarkJournalPoll(b *testing.B) {
	root := b.TempDir()
	backend, err := newMetadataBackend(root)
	if err != nil {
		b.Fatalf("newMetadataBackend() error = %v", err)
	}
	j := newJournalBroker(backend, time.Now, 4096)
	watchID, _, err := j.Subscribe(backend.RootNodeID(), true)
	if err != nil {
		b.Fatalf("Subscribe() error = %v", err)
	}
	for i := 0; i < 1024; i++ {
		j.Append(protocol.WatchEvent{EventType: protocol.EventCreate, Path: fmt.Sprintf("file-%d.txt", i)})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := j.Poll(watchID, 0, 256)
		if err != nil {
			b.Fatalf("Poll() error = %v", err)
		}
		if len(resp.Events) == 0 {
			b.Fatalf("Poll() returned no events")
		}
	}
}
