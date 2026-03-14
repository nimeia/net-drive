package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"developer-mount/internal/protocol"
	"developer-mount/internal/transport"
)

type Server struct {
	Addr           string
	AuthToken      string
	SessionManager *SessionManager
	Now            func() time.Time
}

type connectionState struct {
	helloDone     bool
	authenticated bool
	principalID   string
	serverVersion string
	serverName    string
}

func New(addr string) *Server {
	return &Server{
		Addr:           addr,
		AuthToken:      "devmount-dev-token",
		SessionManager: NewSessionManager(),
		Now:            time.Now,
	}
}

func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	return s.Serve(ln)
}

func (s *Server) Serve(ln net.Listener) error {
	log.Printf("devmount server listening on %s", ln.Addr())
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	state := connectionState{serverName: "devmount-server", serverVersion: "0.1.0"}
	for {
		header, payload, err := transport.DecodeFrame(conn)
		if err != nil {
			if !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) {
				log.Printf("decode frame error: %v", err)
			}
			return
		}
		if header.Channel != protocol.ChannelControl {
			s.writeError(conn, header, protocol.ErrUnsupportedOp, "only control channel is implemented", false, nil)
			continue
		}
		s.dispatch(conn, &state, header, payload)
	}
}

func (s *Server) dispatch(conn net.Conn, state *connectionState, header protocol.Header, payload []byte) {
	switch header.Opcode {
	case protocol.OpcodeHelloReq:
		req, err := transport.DecodePayload[protocol.HelloReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid hello payload", false, nil)
			return
		}
		selected := uint8(0)
		for _, version := range req.SupportedProtocolVersions {
			if version == protocol.Version {
				selected = version
				break
			}
		}
		if selected == 0 {
			s.writeError(conn, header, protocol.ErrUnsupportedVersion, "no compatible protocol version", false, nil)
			return
		}
		state.helloDone = true
		resp := protocol.HelloResp{
			ServerName:              state.serverName,
			ServerVersion:           state.serverVersion,
			SelectedProtocolVersion: selected,
			ServerTime:              protocol.NowRFC3339(s.Now()),
			Capabilities:            protocol.DefaultCapabilities(),
		}
		s.writeResponse(conn, header, protocol.OpcodeHelloResp, 0, resp)
	case protocol.OpcodeAuthReq:
		if !state.helloDone {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "hello required before auth", false, nil)
			return
		}
		req, err := transport.DecodePayload[protocol.AuthReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid auth payload", false, nil)
			return
		}
		if req.Scheme != "dev-token" || req.Token != s.AuthToken {
			s.writeError(conn, header, protocol.ErrAuthRequired, "authentication required", false, map[string]any{"operation": "Auth"})
			return
		}
		state.authenticated = true
		state.principalID = "developer"
		resp := protocol.AuthResp{Authenticated: true, PrincipalID: state.principalID, DisplayName: "Developer", GrantedFeature: []string{"session-create", "heartbeat"}}
		s.writeResponse(conn, header, protocol.OpcodeAuthResp, 0, resp)
	case protocol.OpcodeCreateSessionReq:
		if !state.authenticated {
			s.writeError(conn, header, protocol.ErrAuthRequired, "authentication required", false, map[string]any{"operation": "CreateSession"})
			return
		}
		req, err := transport.DecodePayload[protocol.CreateSessionReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid create-session payload", false, nil)
			return
		}
		lease := req.RequestedLeaseSeconds
		if lease == 0 {
			lease = 30
		}
		if lease > 300 {
			lease = 300
		}
		now := s.Now()
		session := s.SessionManager.Create(state.principalID, req.ClientInstanceID, lease, now)
		resp := protocol.CreateSessionResp{SessionID: session.ID, LeaseSeconds: session.LeaseSeconds, ExpiresAt: protocol.NowRFC3339(session.ExpiresAt), State: session.State}
		s.writeResponse(conn, header, protocol.OpcodeCreateSessionResp, session.ID, resp)
	case protocol.OpcodeResumeSessionReq:
		s.writeError(conn, header, protocol.ErrUnsupportedOp, "resume session is reserved but not implemented", false, map[string]any{"operation": "ResumeSession"})
	case protocol.OpcodeHeartbeatReq:
		req, err := transport.DecodePayload[protocol.HeartbeatReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid heartbeat payload", false, nil)
			return
		}
		now := s.Now()
		session, found, active := s.SessionManager.Heartbeat(req.SessionID, now)
		if !found {
			s.writeError(conn, header, protocol.ErrSessionNotFound, "session not found", false, nil)
			return
		}
		if !active {
			s.writeError(conn, header, protocol.ErrSessionExpired, "session expired", false, nil)
			return
		}
		resp := protocol.HeartbeatResp{SessionID: session.ID, ServerTime: protocol.NowRFC3339(now), ExpiresAt: protocol.NowRFC3339(session.ExpiresAt), State: session.State}
		s.writeResponse(conn, header, protocol.OpcodeHeartbeatResp, session.ID, resp)
	default:
		s.writeError(conn, header, protocol.ErrUnsupportedOp, fmt.Sprintf("unsupported opcode %d", header.Opcode), false, nil)
	}
}

func (s *Server) writeResponse(conn net.Conn, reqHeader protocol.Header, opcode protocol.Opcode, sessionID uint64, payload any) {
	header := protocol.Header{Channel: reqHeader.Channel, Opcode: opcode, Flags: protocol.FlagResponse, RequestID: reqHeader.RequestID, SessionID: sessionID}
	frame, err := transport.EncodeFrame(header, payload)
	if err != nil {
		log.Printf("encode response failed: %v", err)
		return
	}
	_, _ = conn.Write(frame)
}

func (s *Server) writeError(conn net.Conn, reqHeader protocol.Header, code protocol.ErrorCode, message string, retryable bool, details map[string]any) {
	header := protocol.Header{Channel: reqHeader.Channel, Opcode: protocol.OpcodeErrorResp, Flags: protocol.FlagResponse | protocol.FlagError, RequestID: reqHeader.RequestID, SessionID: reqHeader.SessionID}
	frame, err := transport.EncodeFrame(header, protocol.ErrorResp{Code: code, Message: message, Retryable: retryable, Details: details})
	if err != nil {
		log.Printf("encode error response failed: %v", err)
		return
	}
	_, _ = conn.Write(frame)
}
