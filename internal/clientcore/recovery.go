package clientcore

import "developer-mount/internal/protocol"

func (c *Client) RecoverHandles(handles []protocol.RecoverHandleSpec) (protocol.RecoverHandlesResp, error) {
	resp, err := requestDecode[protocol.RecoverHandlesResp](c, protocol.ChannelRecovery, protocol.OpcodeRecoverHandlesReq, protocol.RecoverHandlesReq{Handles: handles})
	if err != nil {
		return protocol.RecoverHandlesResp{}, err
	}
	c.applyRecoveredHandles(resp.Handles)
	return resp, nil
}

func (c *Client) RevalidateNodes(nodeIDs []uint64) (protocol.RevalidateResp, error) {
	resp, err := requestDecode[protocol.RevalidateResp](c, protocol.ChannelRecovery, protocol.OpcodeRevalidateReq, protocol.RevalidateReq{NodeIDs: nodeIDs})
	if err != nil {
		return protocol.RevalidateResp{}, err
	}
	tracked := make([]uint64, 0, len(resp.Entries))
	for _, entry := range resp.Entries {
		if entry.Exists {
			tracked = append(tracked, entry.NodeID, entry.Entry.ParentNodeID)
		}
	}
	c.trackNodeIDs(tracked...)
	return resp, nil
}

func (c *Client) ResubscribeWatches(watches []protocol.ResubscribeSpec) (protocol.ResubscribeResp, error) {
	resp, err := requestDecode[protocol.ResubscribeResp](c, protocol.ChannelRecovery, protocol.OpcodeResubscribeReq, protocol.ResubscribeReq{Watches: watches})
	if err != nil {
		return protocol.ResubscribeResp{}, err
	}
	c.applyResubscribedWatches(resp.Watches)
	return resp, nil
}
