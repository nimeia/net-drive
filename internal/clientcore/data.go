package clientcore

import "developer-mount/internal/protocol"

func (c *Client) OpenRead(nodeID uint64) (protocol.OpenResp, error) {
	return c.open(nodeID, false, false)
}

func (c *Client) OpenWrite(nodeID uint64, truncate bool) (protocol.OpenResp, error) {
	return c.open(nodeID, true, truncate)
}

func (c *Client) open(nodeID uint64, writable, truncate bool) (protocol.OpenResp, error) {
	resp, err := requestDecode[protocol.OpenResp](c, protocol.ChannelData, protocol.OpcodeOpenReq, protocol.OpenReq{
		NodeID:   nodeID,
		Writable: writable,
		Truncate: truncate,
	})
	if err != nil {
		return protocol.OpenResp{}, err
	}
	c.updateHandle(TrackedHandle{HandleID: resp.HandleID, NodeID: nodeID, Writable: writable, LastKnownSize: resp.Size})
	return resp, nil
}

func (c *Client) Create(parentNodeID uint64, name string, overwrite bool) (protocol.CreateResp, error) {
	resp, err := requestDecode[protocol.CreateResp](c, protocol.ChannelData, protocol.OpcodeCreateReq, protocol.CreateReq{
		ParentNodeID: parentNodeID,
		Name:         name,
		Overwrite:    overwrite,
	})
	if err != nil {
		return protocol.CreateResp{}, err
	}
	c.trackNodeIDs(parentNodeID, resp.Entry.NodeID)
	c.updateHandle(TrackedHandle{HandleID: resp.HandleID, NodeID: resp.Entry.NodeID, Writable: true, LastKnownSize: resp.Entry.Size})
	return resp, nil
}

func (c *Client) Read(handleID uint64, offset int64, length uint32) (protocol.ReadResp, error) {
	return requestDecode[protocol.ReadResp](c, protocol.ChannelData, protocol.OpcodeReadReq, protocol.ReadReq{HandleID: handleID, Offset: offset, Length: length})
}

func (c *Client) Write(handleID uint64, offset int64, data []byte) (protocol.WriteResp, error) {
	resp, err := requestDecode[protocol.WriteResp](c, protocol.ChannelData, protocol.OpcodeWriteReq, protocol.WriteReq{HandleID: handleID, Offset: offset, Data: data})
	if err != nil {
		return protocol.WriteResp{}, err
	}
	c.markHandleWrite(handleID, resp.NewSize)
	return resp, nil
}

func (c *Client) Flush(handleID uint64) (protocol.FlushResp, error) {
	resp, err := requestDecode[protocol.FlushResp](c, protocol.ChannelData, protocol.OpcodeFlushReq, protocol.FlushReq{HandleID: handleID})
	if err != nil {
		return protocol.FlushResp{}, err
	}
	c.markHandleFlush(handleID)
	return resp, nil
}

func (c *Client) Truncate(handleID uint64, size int64) (protocol.TruncateResp, error) {
	resp, err := requestDecode[protocol.TruncateResp](c, protocol.ChannelData, protocol.OpcodeTruncateReq, protocol.TruncateReq{HandleID: handleID, Size: size})
	if err != nil {
		return protocol.TruncateResp{}, err
	}
	c.markHandleTruncate(handleID, resp.Size)
	return resp, nil
}

func (c *Client) SetDeleteOnClose(handleID uint64, enabled bool) (protocol.SetDeleteOnCloseResp, error) {
	resp, err := requestDecode[protocol.SetDeleteOnCloseResp](c, protocol.ChannelData, protocol.OpcodeSetDeleteOnCloseReq, protocol.SetDeleteOnCloseReq{HandleID: handleID, DeleteOnClose: enabled})
	if err != nil {
		return protocol.SetDeleteOnCloseResp{}, err
	}
	c.setHandleDeleteOnClose(handleID, enabled)
	return resp, nil
}

func (c *Client) CloseHandle(handleID uint64) (protocol.CloseResp, error) {
	resp, err := requestDecode[protocol.CloseResp](c, protocol.ChannelData, protocol.OpcodeCloseReq, protocol.CloseReq{HandleID: handleID})
	if err != nil {
		return protocol.CloseResp{}, err
	}
	c.removeHandle(handleID)
	return resp, nil
}
