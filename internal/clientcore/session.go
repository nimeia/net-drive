package clientcore

import (
	"time"

	"developer-mount/internal/protocol"
)

func (c *Client) Hello() (protocol.HelloResp, error) {
	state := c.SnapshotState()
	resp, err := requestDecode[protocol.HelloResp](c, protocol.ChannelControl, protocol.OpcodeHelloReq, protocol.HelloReq{
		ClientName:                state.ClientName,
		ClientVersion:             state.ClientVersion,
		SupportedProtocolVersions: []uint8{protocol.Version},
		Capabilities:              protocol.DefaultCapabilities(),
	})
	if err != nil {
		return protocol.HelloResp{}, err
	}
	c.stateMu.Lock()
	c.state.ServerName = resp.ServerName
	c.state.ServerVersion = resp.ServerVersion
	c.state.SelectedProtocolVersion = resp.SelectedProtocolVersion
	c.state.ServerCapabilities = append([]string(nil), resp.Capabilities.Features...)
	c.stateMu.Unlock()
	return resp, nil
}

func (c *Client) Auth(token string) (protocol.AuthResp, error) {
	resp, err := requestDecode[protocol.AuthResp](c, protocol.ChannelControl, protocol.OpcodeAuthReq, protocol.AuthReq{
		Scheme: "dev-token",
		Token:  token,
	})
	if err != nil {
		return protocol.AuthResp{}, err
	}
	c.stateMu.Lock()
	c.state.PrincipalID = resp.PrincipalID
	c.state.DisplayName = resp.DisplayName
	c.state.GrantedFeatures = append([]string(nil), resp.GrantedFeature...)
	c.stateMu.Unlock()
	return resp, nil
}

func (c *Client) CreateSession(clientInstanceID string, leaseSeconds uint32) (protocol.CreateSessionResp, error) {
	resp, err := requestDecode[protocol.CreateSessionResp](c, protocol.ChannelControl, protocol.OpcodeCreateSessionReq, protocol.CreateSessionReq{
		ClientInstanceID:      clientInstanceID,
		RequestedLeaseSeconds: leaseSeconds,
		MountName:             c.SnapshotState().MountName,
	})
	if err != nil {
		return protocol.CreateSessionResp{}, err
	}
	c.SessionID = resp.SessionID
	c.stateMu.Lock()
	c.state.ClientInstanceID = clientInstanceID
	c.state.LeaseSeconds = resp.LeaseSeconds
	c.state.SessionExpiresAt = resp.ExpiresAt
	c.state.SessionState = resp.State
	c.stateMu.Unlock()
	return resp, nil
}

func (c *Client) ResumeSession(sessionID uint64, clientInstanceID string) (protocol.ResumeSessionResp, error) {
	prev := c.SessionID
	c.SessionID = sessionID
	resp, err := requestDecode[protocol.ResumeSessionResp](c, protocol.ChannelControl, protocol.OpcodeResumeSessionReq, protocol.ResumeSessionReq{
		SessionID:           sessionID,
		ClientInstanceID:    clientInstanceID,
		LastKnownServerTime: protocol.NowRFC3339(time.Now()),
	})
	if err != nil {
		c.SessionID = prev
		return protocol.ResumeSessionResp{}, err
	}
	c.SessionID = resp.SessionID
	c.stateMu.Lock()
	c.state.ClientInstanceID = clientInstanceID
	c.state.LeaseSeconds = resp.LeaseSeconds
	c.state.SessionExpiresAt = resp.ExpiresAt
	c.state.SessionState = resp.State
	c.stateMu.Unlock()
	return resp, nil
}

func (c *Client) Heartbeat() (protocol.HeartbeatResp, error) {
	resp, err := requestDecode[protocol.HeartbeatResp](c, protocol.ChannelControl, protocol.OpcodeHeartbeatReq, protocol.HeartbeatReq{
		SessionID:  c.SessionID,
		ClientTime: protocol.NowRFC3339(time.Now()),
	})
	c.setHeartbeatResult(err)
	if err != nil {
		return protocol.HeartbeatResp{}, err
	}
	c.stateMu.Lock()
	c.state.SessionExpiresAt = resp.ExpiresAt
	c.state.SessionState = resp.State
	c.stateMu.Unlock()
	return resp, nil
}
