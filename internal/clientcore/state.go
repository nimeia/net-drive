package clientcore

import (
	"slices"
	"time"

	"developer-mount/internal/protocol"
)

type TrackedHandle struct {
	HandleID       uint64
	NodeID         uint64
	Writable       bool
	DeleteOnClose  bool
	LastKnownSize  int64
	LastWriteAt    time.Time
	LastFlushAt    time.Time
	LastTruncateAt time.Time
}

type TrackedDirCursor struct {
	DirCursorID uint64
	NodeID      uint64
}

type TrackedWatch struct {
	WatchID      uint64
	NodeID       uint64
	Recursive    bool
	LastAckedSeq uint64
	LastSeenSeq  uint64
}

type RecoverySnapshot struct {
	SessionID        uint64
	ClientInstanceID string
	Handles          []protocol.RecoverHandleSpec
	NodeIDs          []uint64
	Watches          []protocol.ResubscribeSpec
}

func (s RuntimeState) clone() RuntimeState {
	clone := s
	clone.GrantedFeatures = append([]string(nil), s.GrantedFeatures...)
	clone.ServerCapabilities = append([]string(nil), s.ServerCapabilities...)
	clone.TrackedHandles = make(map[uint64]TrackedHandle, len(s.TrackedHandles))
	for id, handle := range s.TrackedHandles {
		clone.TrackedHandles[id] = handle
	}
	clone.TrackedDirCursors = make(map[uint64]TrackedDirCursor, len(s.TrackedDirCursors))
	for id, cursor := range s.TrackedDirCursors {
		clone.TrackedDirCursors[id] = cursor
	}
	clone.TrackedWatches = make(map[uint64]TrackedWatch, len(s.TrackedWatches))
	for id, watch := range s.TrackedWatches {
		clone.TrackedWatches[id] = watch
	}
	clone.TrackedNodes = make(map[uint64]struct{}, len(s.TrackedNodes))
	for id := range s.TrackedNodes {
		clone.TrackedNodes[id] = struct{}{}
	}
	return clone
}

func (s RuntimeState) recoverySnapshot(sessionID uint64) RecoverySnapshot {
	handleIDs := make([]uint64, 0, len(s.TrackedHandles))
	for handleID := range s.TrackedHandles {
		handleIDs = append(handleIDs, handleID)
	}
	slices.Sort(handleIDs)
	handles := make([]protocol.RecoverHandleSpec, 0, len(handleIDs))
	for _, handleID := range handleIDs {
		handle := s.TrackedHandles[handleID]
		handles = append(handles, protocol.RecoverHandleSpec{
			PreviousHandleID: handle.HandleID,
			NodeID:           handle.NodeID,
			Writable:         handle.Writable,
			DeleteOnClose:    handle.DeleteOnClose,
		})
	}

	nodeIDs := make([]uint64, 0, len(s.TrackedNodes))
	for nodeID := range s.TrackedNodes {
		nodeIDs = append(nodeIDs, nodeID)
	}
	slices.Sort(nodeIDs)

	watchIDs := make([]uint64, 0, len(s.TrackedWatches))
	for watchID := range s.TrackedWatches {
		watchIDs = append(watchIDs, watchID)
	}
	slices.Sort(watchIDs)
	watches := make([]protocol.ResubscribeSpec, 0, len(watchIDs))
	for _, watchID := range watchIDs {
		watch := s.TrackedWatches[watchID]
		watches = append(watches, protocol.ResubscribeSpec{
			PreviousWatchID: watch.WatchID,
			NodeID:          watch.NodeID,
			Recursive:       watch.Recursive,
			AfterSeq:        max(watch.LastAckedSeq, watch.LastSeenSeq),
		})
	}

	return RecoverySnapshot{
		SessionID:        sessionID,
		ClientInstanceID: s.ClientInstanceID,
		Handles:          handles,
		NodeIDs:          nodeIDs,
		Watches:          watches,
	}
}

func (c *Client) trackNodeIDs(nodeIDs ...uint64) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	for _, nodeID := range nodeIDs {
		if nodeID == 0 {
			continue
		}
		c.state.TrackedNodes[nodeID] = struct{}{}
	}
}

func (c *Client) updateHandle(handle TrackedHandle) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.state.TrackedHandles[handle.HandleID] = handle
	if handle.NodeID != 0 {
		c.state.TrackedNodes[handle.NodeID] = struct{}{}
	}
}

func (c *Client) setHandleDeleteOnClose(handleID uint64, enabled bool) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	handle, ok := c.state.TrackedHandles[handleID]
	if !ok {
		return
	}
	handle.DeleteOnClose = enabled
	c.state.TrackedHandles[handleID] = handle
}

func (c *Client) markHandleWrite(handleID uint64, size int64) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	handle, ok := c.state.TrackedHandles[handleID]
	if !ok {
		return
	}
	handle.LastKnownSize = size
	handle.LastWriteAt = time.Now().UTC()
	c.state.TrackedHandles[handleID] = handle
}

func (c *Client) markHandleFlush(handleID uint64) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	handle, ok := c.state.TrackedHandles[handleID]
	if !ok {
		return
	}
	handle.LastFlushAt = time.Now().UTC()
	c.state.TrackedHandles[handleID] = handle
}

func (c *Client) markHandleTruncate(handleID uint64, size int64) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	handle, ok := c.state.TrackedHandles[handleID]
	if !ok {
		return
	}
	handle.LastKnownSize = size
	handle.LastTruncateAt = time.Now().UTC()
	c.state.TrackedHandles[handleID] = handle
}

func (c *Client) removeHandle(handleID uint64) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	delete(c.state.TrackedHandles, handleID)
}

func (c *Client) updateDirCursor(cursor TrackedDirCursor) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.state.TrackedDirCursors[cursor.DirCursorID] = cursor
	if cursor.NodeID != 0 {
		c.state.TrackedNodes[cursor.NodeID] = struct{}{}
	}
}

func (c *Client) updateWatch(watch TrackedWatch) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.state.TrackedWatches[watch.WatchID] = watch
	if watch.NodeID != 0 {
		c.state.TrackedNodes[watch.NodeID] = struct{}{}
	}
}

func (c *Client) markWatchSeen(watchID, latestSeq uint64) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	watch, ok := c.state.TrackedWatches[watchID]
	if !ok {
		return
	}
	watch.LastSeenSeq = max(watch.LastSeenSeq, latestSeq)
	c.state.TrackedWatches[watchID] = watch
}

func (c *Client) markWatchAcked(watchID, ackedSeq uint64) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	watch, ok := c.state.TrackedWatches[watchID]
	if !ok {
		return
	}
	watch.LastAckedSeq = max(watch.LastAckedSeq, ackedSeq)
	watch.LastSeenSeq = max(watch.LastSeenSeq, ackedSeq)
	c.state.TrackedWatches[watchID] = watch
}

func (c *Client) applyRecoveredHandles(results []protocol.RecoverHandleResult) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	for _, result := range results {
		previous, hadPrevious := c.state.TrackedHandles[result.PreviousHandleID]
		if result.PreviousHandleID != 0 {
			delete(c.state.TrackedHandles, result.PreviousHandleID)
		}
		if result.Error != "" || result.HandleID == 0 {
			continue
		}
		writable := previous.Writable
		if !hadPrevious {
			writable = lookupRecoveredWritable(result.NodeID, c.state.TrackedHandles)
		}
		c.state.TrackedHandles[result.HandleID] = TrackedHandle{
			HandleID:      result.HandleID,
			NodeID:        result.NodeID,
			Writable:      writable,
			DeleteOnClose: result.DeleteOnClose || previous.DeleteOnClose,
			LastKnownSize: result.Size,
		}
		if result.NodeID != 0 {
			c.state.TrackedNodes[result.NodeID] = struct{}{}
		}
	}
}

func lookupRecoveredWritable(nodeID uint64, handles map[uint64]TrackedHandle) bool {
	for _, handle := range handles {
		if handle.NodeID == nodeID {
			return handle.Writable
		}
	}
	return false
}

func (c *Client) applyResubscribedWatches(results []protocol.ResubscribeResult) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	for _, result := range results {
		previous, ok := c.state.TrackedWatches[result.PreviousWatchID]
		if ok {
			delete(c.state.TrackedWatches, result.PreviousWatchID)
		}
		if result.Error != "" || result.WatchID == 0 {
			continue
		}
		c.state.TrackedWatches[result.WatchID] = TrackedWatch{
			WatchID:      result.WatchID,
			NodeID:       previous.NodeID,
			Recursive:    previous.Recursive,
			LastAckedSeq: max(previous.LastAckedSeq, result.AckedSeq),
			LastSeenSeq:  max(previous.LastSeenSeq, result.StartSeq),
		}
		if previous.NodeID != 0 {
			c.state.TrackedNodes[previous.NodeID] = struct{}{}
		}
	}
}

func (c *Client) setHeartbeatResult(err error) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.state.LastHeartbeatAt = time.Now().UTC()
	if err != nil {
		c.state.LastHeartbeatError = err.Error()
		return
	}
	c.state.LastHeartbeatError = ""
}
