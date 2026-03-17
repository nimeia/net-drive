package clientcore

import "developer-mount/internal/protocol"

func (c *Client) Lookup(parentNodeID uint64, name string) (protocol.LookupResp, error) {
	resp, err := requestDecode[protocol.LookupResp](c, protocol.ChannelMetadata, protocol.OpcodeLookupReq, protocol.LookupReq{
		ParentNodeID: parentNodeID,
		Name:         name,
	})
	if err != nil {
		return protocol.LookupResp{}, err
	}
	c.trackNodeIDs(parentNodeID, resp.Entry.NodeID)
	return resp, nil
}

func (c *Client) GetAttr(nodeID uint64) (protocol.GetAttrResp, error) {
	resp, err := requestDecode[protocol.GetAttrResp](c, protocol.ChannelMetadata, protocol.OpcodeGetAttrReq, protocol.GetAttrReq{NodeID: nodeID})
	if err != nil {
		return protocol.GetAttrResp{}, err
	}
	c.trackNodeIDs(nodeID, resp.Entry.ParentNodeID)
	return resp, nil
}

func (c *Client) OpenDir(nodeID uint64) (protocol.OpenDirResp, error) {
	resp, err := requestDecode[protocol.OpenDirResp](c, protocol.ChannelMetadata, protocol.OpcodeOpenDirReq, protocol.OpenDirReq{NodeID: nodeID})
	if err != nil {
		return protocol.OpenDirResp{}, err
	}
	c.updateDirCursor(TrackedDirCursor{DirCursorID: resp.DirCursorID, NodeID: nodeID})
	return resp, nil
}

func (c *Client) ReadDir(dirCursorID uint64, cookie uint64, maxEntries uint32) (protocol.ReadDirResp, error) {
	resp, err := requestDecode[protocol.ReadDirResp](c, protocol.ChannelMetadata, protocol.OpcodeReadDirReq, protocol.ReadDirReq{
		DirCursorID: dirCursorID,
		Cookie:      cookie,
		MaxEntries:  maxEntries,
	})
	if err != nil {
		return protocol.ReadDirResp{}, err
	}
	nodeIDs := make([]uint64, 0, len(resp.Entries))
	for _, entry := range resp.Entries {
		nodeIDs = append(nodeIDs, entry.NodeID)
	}
	c.trackNodeIDs(nodeIDs...)
	return resp, nil
}

func (c *Client) Rename(srcParentNodeID uint64, srcName string, dstParentNodeID uint64, dstName string, replace bool) (protocol.RenameResp, error) {
	resp, err := requestDecode[protocol.RenameResp](c, protocol.ChannelMetadata, protocol.OpcodeRenameReq, protocol.RenameReq{
		SrcParentNodeID: srcParentNodeID,
		SrcName:         srcName,
		DstParentNodeID: dstParentNodeID,
		DstName:         dstName,
		ReplaceExisting: replace,
	})
	if err != nil {
		return protocol.RenameResp{}, err
	}
	c.trackNodeIDs(srcParentNodeID, dstParentNodeID, resp.Entry.NodeID)
	return resp, nil
}
