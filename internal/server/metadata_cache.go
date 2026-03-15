package server

import (
	"sync/atomic"
	"time"

	"developer-mount/internal/protocol"
)

type attrCacheEntry struct {
	info      protocol.NodeInfo
	expiresAt time.Time
}

type negativeCacheEntry struct {
	expiresAt time.Time
}

type dirSnapshotEntry struct {
	entries   []protocol.DirEntry
	expiresAt time.Time
}

type smallFileCacheEntry struct {
	data      []byte
	expiresAt time.Time
}

type MetadataCacheStats struct {
	AttrHits                 uint64
	AttrMisses               uint64
	NegativeHits             uint64
	NegativeMisses           uint64
	DirSnapshotHits          uint64
	DirSnapshotMisses        uint64
	SmallFileHits            uint64
	SmallFileMisses          uint64
	SmallFilePrefetches      uint64
	RootPrefetches           uint64
	HotDirPrefetches         uint64
	HotFilePrefetches        uint64
	HighPriorityPrefetches   uint64
	NormalPriorityPrefetches uint64
}

type metadataCacheStats struct {
	attrHits                 atomic.Uint64
	attrMisses               atomic.Uint64
	negativeHits             atomic.Uint64
	negativeMisses           atomic.Uint64
	dirSnapshotHits          atomic.Uint64
	dirSnapshotMisses        atomic.Uint64
	smallFileHits            atomic.Uint64
	smallFileMisses          atomic.Uint64
	smallFilePrefetches      atomic.Uint64
	rootPrefetches           atomic.Uint64
	hotDirPrefetches         atomic.Uint64
	hotFilePrefetches        atomic.Uint64
	highPriorityPrefetches   atomic.Uint64
	normalPriorityPrefetches atomic.Uint64
}

func (s *metadataCacheStats) snapshot() MetadataCacheStats {
	return MetadataCacheStats{
		AttrHits:                 s.attrHits.Load(),
		AttrMisses:               s.attrMisses.Load(),
		NegativeHits:             s.negativeHits.Load(),
		NegativeMisses:           s.negativeMisses.Load(),
		DirSnapshotHits:          s.dirSnapshotHits.Load(),
		DirSnapshotMisses:        s.dirSnapshotMisses.Load(),
		SmallFileHits:            s.smallFileHits.Load(),
		SmallFileMisses:          s.smallFileMisses.Load(),
		SmallFilePrefetches:      s.smallFilePrefetches.Load(),
		RootPrefetches:           s.rootPrefetches.Load(),
		HotDirPrefetches:         s.hotDirPrefetches.Load(),
		HotFilePrefetches:        s.hotFilePrefetches.Load(),
		HighPriorityPrefetches:   s.highPriorityPrefetches.Load(),
		NormalPriorityPrefetches: s.normalPriorityPrefetches.Load(),
	}
}
