package clientcore

import "developer-mount/internal/protocol"

func (c *Client) Subscribe(nodeID uint64, recursive bool) (protocol.SubscribeResp, error) {
	resp, err := requestDecode[protocol.SubscribeResp](c, protocol.ChannelEvents, protocol.OpcodeSubscribeReq, protocol.SubscribeReq{NodeID: nodeID, Recursive: recursive})
	if err != nil {
		return protocol.SubscribeResp{}, err
	}
	c.updateWatch(TrackedWatch{WatchID: resp.WatchID, NodeID: nodeID, Recursive: recursive, LastSeenSeq: resp.StartSeq, LastAckedSeq: resp.StartSeq})
	return resp, nil
}

func (c *Client) PollEvents(watchID, afterSeq uint64, maxEvents uint32) (protocol.PollEventsResp, error) {
	resp, err := requestDecode[protocol.PollEventsResp](c, protocol.ChannelEvents, protocol.OpcodePollEventsReq, protocol.PollEventsReq{WatchID: watchID, AfterSeq: afterSeq, MaxEvents: maxEvents})
	if err != nil {
		return protocol.PollEventsResp{}, err
	}
	c.markWatchSeen(watchID, resp.LatestSeq)
	nodeIDs := make([]uint64, 0, len(resp.Events))
	for _, event := range resp.Events {
		nodeIDs = append(nodeIDs, event.NodeID, event.ParentNodeID, event.OldParentNodeID)
	}
	c.trackNodeIDs(nodeIDs...)
	return resp, nil
}

func (c *Client) AckEvents(watchID, eventSeq uint64) (protocol.AckEventsResp, error) {
	resp, err := requestDecode[protocol.AckEventsResp](c, protocol.ChannelEvents, protocol.OpcodeAckEventsReq, protocol.AckEventsReq{WatchID: watchID, EventSeq: eventSeq})
	if err != nil {
		return protocol.AckEventsResp{}, err
	}
	c.markWatchAcked(watchID, resp.AckedSeq)
	return resp, nil
}

func (c *Client) Resync(watchID uint64) (protocol.ResyncResp, error) {
	resp, err := requestDecode[protocol.ResyncResp](c, protocol.ChannelEvents, protocol.OpcodeResyncReq, protocol.ResyncReq{WatchID: watchID})
	if err != nil {
		return protocol.ResyncResp{}, err
	}
	nodeIDs := make([]uint64, 0, len(resp.Entries))
	for _, entry := range resp.Entries {
		nodeIDs = append(nodeIDs, entry.Entry.NodeID, entry.Entry.ParentNodeID)
	}
	c.trackNodeIDs(nodeIDs...)
	c.markWatchSeen(watchID, resp.SnapshotSeq)
	return resp, nil
}
