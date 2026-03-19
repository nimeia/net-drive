package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"developer-mount/internal/protocol"
	"developer-mount/internal/transport"
)

type Server struct {
	Addr             string
	RootPath         string
	AuthToken        string
	SessionManager   *SessionManager
	Backend          *metadataBackend
	Journal          *journalBroker
	JournalRetention int
	Now              func() time.Time
	StartedAt        time.Time
	Audit            *AuditLogger
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
		Addr:             addr,
		RootPath:         ".",
		AuthToken:        "devmount-dev-token",
		SessionManager:   NewSessionManager(),
		JournalRetention: 256,
		Now:              time.Now,
		StartedAt:        time.Now(),
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
	if s.Backend == nil {
		backend, err := newMetadataBackend(s.RootPath)
		if err != nil {
			return err
		}
		s.Backend = backend
	}
	if s.Journal == nil {
		s.Journal = newJournalBroker(s.Backend, s.Now, s.JournalRetention)
	}
	log.Printf("devmount server listening on %s root=%s", ln.Addr(), s.Backend.rootPath)
	_ = s.audit("server", "listen", "success", 0, 0, "", map[string]any{"addr": ln.Addr().String(), "root": s.Backend.rootPath})
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
	state := connectionState{serverName: "devmount-server", serverVersion: "0.5.0"}
	for {
		header, payload, err := transport.DecodeFrame(conn)
		if err != nil {
			if !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				log.Printf("decode frame error: %v", err)
			}
			return
		}
		s.dispatch(conn, &state, header, payload)
	}
}

func (s *Server) dispatch(conn net.Conn, state *connectionState, header protocol.Header, payload []byte) {
	switch header.Channel {
	case protocol.ChannelControl:
		s.dispatchControl(conn, state, header, payload)
	case protocol.ChannelMetadata:
		if !s.requireActiveSession(conn, header) {
			return
		}
		s.dispatchMetadata(conn, header, payload)
	case protocol.ChannelData:
		if !s.requireActiveSession(conn, header) {
			return
		}
		s.dispatchData(conn, header, payload)
	case protocol.ChannelEvents:
		if !s.requireActiveSession(conn, header) {
			return
		}
		s.dispatchEvents(conn, header, payload)
	case protocol.ChannelRecovery:
		if !s.requireActiveSession(conn, header) {
			return
		}
		s.dispatchRecovery(conn, header, payload)
	default:
		s.writeError(conn, header, protocol.ErrUnsupportedOp, "channel not implemented", false, nil)
	}
}

func (s *Server) dispatchControl(conn net.Conn, state *connectionState, header protocol.Header, payload []byte) {
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
		resp := protocol.AuthResp{Authenticated: true, PrincipalID: state.principalID, DisplayName: "Developer", GrantedFeature: protocol.DefaultCapabilities().Features}
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
		req, err := transport.DecodePayload[protocol.ResumeSessionReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid resume-session payload", false, nil)
			return
		}
		now := s.Now()
		session, found, active, matched := s.SessionManager.Resume(req.SessionID, req.ClientInstanceID, now)
		if !found {
			s.writeError(conn, header, protocol.ErrSessionNotFound, "session not found", false, nil)
			return
		}
		if !matched {
			s.writeError(conn, header, protocol.ErrAccessDenied, "client instance mismatch", false, map[string]any{"operation": "ResumeSession"})
			return
		}
		if !active {
			s.writeError(conn, header, protocol.ErrSessionExpired, "session expired", false, nil)
			return
		}
		state.authenticated = true
		state.principalID = session.PrincipalID
		resp := protocol.ResumeSessionResp{SessionID: session.ID, LeaseSeconds: session.LeaseSeconds, ExpiresAt: protocol.NowRFC3339(session.ExpiresAt), State: session.State}
		s.writeResponse(conn, header, protocol.OpcodeResumeSessionResp, session.ID, resp)
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
		s.writeError(conn, header, protocol.ErrUnsupportedOp, fmt.Sprintf("unsupported control opcode %d", header.Opcode), false, nil)
	}
}

func (s *Server) dispatchMetadata(conn net.Conn, header protocol.Header, payload []byte) {
	switch header.Opcode {
	case protocol.OpcodeLookupReq:
		req, err := transport.DecodePayload[protocol.LookupReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid lookup payload", false, nil)
			return
		}
		entry, err := s.Backend.Lookup(req.ParentNodeID, req.Name)
		if err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		s.writeResponse(conn, header, protocol.OpcodeLookupResp, header.SessionID, protocol.LookupResp{Entry: entry})
	case protocol.OpcodeGetAttrReq:
		req, err := transport.DecodePayload[protocol.GetAttrReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid getattr payload", false, nil)
			return
		}
		entry, err := s.Backend.GetAttr(req.NodeID)
		if err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		s.writeResponse(conn, header, protocol.OpcodeGetAttrResp, header.SessionID, protocol.GetAttrResp{Entry: entry})
	case protocol.OpcodeOpenDirReq:
		req, err := transport.DecodePayload[protocol.OpenDirReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid opendir payload", false, nil)
			return
		}
		cursorID, err := s.Backend.OpenDir(req.NodeID)
		if err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		s.writeResponse(conn, header, protocol.OpcodeOpenDirResp, header.SessionID, protocol.OpenDirResp{DirCursorID: cursorID})
	case protocol.OpcodeReadDirReq:
		req, err := transport.DecodePayload[protocol.ReadDirReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid readdir payload", false, nil)
			return
		}
		resp, err := s.Backend.ReadDir(req.DirCursorID, req.Cookie, req.MaxEntries)
		if err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		s.writeResponse(conn, header, protocol.OpcodeReadDirResp, header.SessionID, resp)
	case protocol.OpcodeRenameReq:
		req, err := transport.DecodePayload[protocol.RenameReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid rename payload", false, nil)
			return
		}
		srcParentRec, err := s.Backend.nodeByID(req.SrcParentNodeID)
		if err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		dstParentRec, err := s.Backend.nodeByID(req.DstParentNodeID)
		if err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		srcRel := filepath.ToSlash(stringsJoin(srcParentRec.relPath, req.SrcName))
		dstRel := filepath.ToSlash(stringsJoin(dstParentRec.relPath, req.DstName))
		entry, err := s.Backend.RenamePath(req.SrcParentNodeID, req.SrcName, req.DstParentNodeID, req.DstName, req.ReplaceExisting)
		if err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		s.Journal.Append(protocol.WatchEvent{EventType: protocol.EventRenameFrom, NodeID: entry.NodeID, ParentNodeID: req.SrcParentNodeID, OldParentNodeID: req.SrcParentNodeID, Name: req.SrcName, OldName: req.SrcName, Path: srcRel, OldPath: srcRel})
		s.Journal.Append(protocol.WatchEvent{EventType: protocol.EventRenameTo, NodeID: entry.NodeID, ParentNodeID: req.DstParentNodeID, OldParentNodeID: req.SrcParentNodeID, Name: req.DstName, OldName: req.SrcName, Path: dstRel, OldPath: srcRel})
		s.writeResponse(conn, header, protocol.OpcodeRenameResp, header.SessionID, protocol.RenameResp{Entry: entry})
	default:
		s.writeError(conn, header, protocol.ErrUnsupportedOp, fmt.Sprintf("unsupported metadata opcode %d", header.Opcode), false, nil)
	}
}

func (s *Server) dispatchData(conn net.Conn, header protocol.Header, payload []byte) {
	switch header.Opcode {
	case protocol.OpcodeOpenReq:
		req, err := transport.DecodePayload[protocol.OpenReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid open payload", false, nil)
			return
		}
		handleID, size, err := s.Backend.OpenFile(req.NodeID, req.Writable, req.Truncate)
		if err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		s.writeResponse(conn, header, protocol.OpcodeOpenResp, header.SessionID, protocol.OpenResp{HandleID: handleID, Size: size})
	case protocol.OpcodeCreateReq:
		req, err := transport.DecodePayload[protocol.CreateReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid create payload", false, nil)
			return
		}
		entry, handleID, err := s.Backend.CreateFile(req.ParentNodeID, req.Name, req.Overwrite)
		if err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		relPath, _ := s.Backend.RelPathByNodeID(entry.NodeID)
		s.Journal.Append(protocol.WatchEvent{EventType: protocol.EventCreate, NodeID: entry.NodeID, ParentNodeID: entry.ParentNodeID, Name: entry.Name, Path: relPath})
		s.writeResponse(conn, header, protocol.OpcodeCreateResp, header.SessionID, protocol.CreateResp{Entry: entry, HandleID: handleID})
	case protocol.OpcodeReadReq:
		req, err := transport.DecodePayload[protocol.ReadReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid read payload", false, nil)
			return
		}
		data, eof, err := s.Backend.ReadFile(req.HandleID, req.Offset, req.Length)
		if err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		s.writeResponse(conn, header, protocol.OpcodeReadResp, header.SessionID, protocol.ReadResp{Data: data, EOF: eof, Offset: req.Offset})
	case protocol.OpcodeWriteReq:
		req, err := transport.DecodePayload[protocol.WriteReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid write payload", false, nil)
			return
		}
		written, newSize, err := s.Backend.WriteFile(req.HandleID, req.Offset, req.Data)
		if err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		s.writeResponse(conn, header, protocol.OpcodeWriteResp, header.SessionID, protocol.WriteResp{BytesWritten: written, NewSize: newSize})
	case protocol.OpcodeFlushReq:
		req, err := transport.DecodePayload[protocol.FlushReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid flush payload", false, nil)
			return
		}
		snapshot, err := s.Backend.HandleSnapshot(req.HandleID)
		if err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		if err := s.Backend.FlushHandle(req.HandleID); err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		s.Journal.Append(protocol.WatchEvent{EventType: protocol.EventContentChanged, NodeID: snapshot.NodeID, ParentNodeID: snapshot.ParentNodeID, Name: snapshot.Name, Path: filepath.ToSlash(snapshot.RelPath)})
		s.writeResponse(conn, header, protocol.OpcodeFlushResp, header.SessionID, protocol.FlushResp{Flushed: true})
	case protocol.OpcodeTruncateReq:
		req, err := transport.DecodePayload[protocol.TruncateReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid truncate payload", false, nil)
			return
		}
		snapshot, err := s.Backend.HandleSnapshot(req.HandleID)
		if err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		size, err := s.Backend.TruncateHandle(req.HandleID, req.Size)
		if err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		s.Journal.Append(protocol.WatchEvent{EventType: protocol.EventContentChanged, NodeID: snapshot.NodeID, ParentNodeID: snapshot.ParentNodeID, Name: snapshot.Name, Path: filepath.ToSlash(snapshot.RelPath)})
		s.writeResponse(conn, header, protocol.OpcodeTruncateResp, header.SessionID, protocol.TruncateResp{Size: size})
	case protocol.OpcodeSetDeleteOnCloseReq:
		req, err := transport.DecodePayload[protocol.SetDeleteOnCloseReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid delete-on-close payload", false, nil)
			return
		}
		if err := s.Backend.SetDeleteOnClose(req.HandleID, req.DeleteOnClose); err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		s.writeResponse(conn, header, protocol.OpcodeSetDeleteOnCloseResp, header.SessionID, protocol.SetDeleteOnCloseResp{DeleteOnClose: req.DeleteOnClose})
	case protocol.OpcodeCloseReq:
		req, err := transport.DecodePayload[protocol.CloseReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid close payload", false, nil)
			return
		}
		snapshot, snapErr := s.Backend.HandleSnapshot(req.HandleID)
		if err := s.Backend.CloseHandle(req.HandleID); err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		if snapErr == nil && snapshot.DeleteOnClose {
			s.Journal.Append(protocol.WatchEvent{EventType: protocol.EventDelete, NodeID: snapshot.NodeID, ParentNodeID: snapshot.ParentNodeID, Name: snapshot.Name, Path: filepath.ToSlash(snapshot.RelPath)})
		}
		s.writeResponse(conn, header, protocol.OpcodeCloseResp, header.SessionID, protocol.CloseResp{Closed: true})
	default:
		s.writeError(conn, header, protocol.ErrUnsupportedOp, fmt.Sprintf("unsupported data opcode %d", header.Opcode), false, nil)
	}
}

func (s *Server) dispatchEvents(conn net.Conn, header protocol.Header, payload []byte) {
	switch header.Opcode {
	case protocol.OpcodeSubscribeReq:
		req, err := transport.DecodePayload[protocol.SubscribeReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid subscribe payload", false, nil)
			return
		}
		watchID, startSeq, err := s.Journal.Subscribe(req.NodeID, req.Recursive)
		if err != nil {
			s.writeBackendError(conn, header, err)
			return
		}
		s.writeResponse(conn, header, protocol.OpcodeSubscribeResp, header.SessionID, protocol.SubscribeResp{WatchID: watchID, StartSeq: startSeq})
	case protocol.OpcodePollEventsReq:
		req, err := transport.DecodePayload[protocol.PollEventsReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid poll events payload", false, nil)
			return
		}
		resp, err := s.Journal.Poll(req.WatchID, req.AfterSeq, req.MaxEvents)
		if err != nil {
			s.writeWatchError(conn, header, err)
			return
		}
		for i := range resp.Events {
			resp.Events[i].WatchID = req.WatchID
		}
		s.writeResponse(conn, header, protocol.OpcodePollEventsResp, header.SessionID, resp)
	case protocol.OpcodeAckEventsReq:
		req, err := transport.DecodePayload[protocol.AckEventsReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid ack events payload", false, nil)
			return
		}
		acked, err := s.Journal.Ack(req.WatchID, req.EventSeq)
		if err != nil {
			s.writeWatchError(conn, header, err)
			return
		}
		s.writeResponse(conn, header, protocol.OpcodeAckEventsResp, header.SessionID, protocol.AckEventsResp{WatchID: req.WatchID, AckedSeq: acked})
	case protocol.OpcodeResyncReq:
		req, err := transport.DecodePayload[protocol.ResyncReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid resync payload", false, nil)
			return
		}
		resp, err := s.Journal.Resync(req.WatchID)
		if err != nil {
			s.writeWatchError(conn, header, err)
			return
		}
		s.writeResponse(conn, header, protocol.OpcodeResyncResp, header.SessionID, resp)
	default:
		s.writeError(conn, header, protocol.ErrUnsupportedOp, fmt.Sprintf("unsupported events opcode %d", header.Opcode), false, nil)
	}
}

func (s *Server) dispatchRecovery(conn net.Conn, header protocol.Header, payload []byte) {
	switch header.Opcode {
	case protocol.OpcodeRecoverHandlesReq:
		req, err := transport.DecodePayload[protocol.RecoverHandlesReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid recover handles payload", false, nil)
			return
		}
		results := make([]protocol.RecoverHandleResult, 0, len(req.Handles))
		for _, spec := range req.Handles {
			handleID, size, err := s.Backend.RecoverFileHandle(spec.NodeID, spec.Writable, spec.DeleteOnClose)
			result := protocol.RecoverHandleResult{PreviousHandleID: spec.PreviousHandleID, NodeID: spec.NodeID, HandleID: handleID, Size: size, DeleteOnClose: spec.DeleteOnClose}
			if err != nil {
				result.Error = err.Error()
			}
			results = append(results, result)
		}
		s.writeResponse(conn, header, protocol.OpcodeRecoverHandlesResp, header.SessionID, protocol.RecoverHandlesResp{Handles: results})
	case protocol.OpcodeRevalidateReq:
		req, err := transport.DecodePayload[protocol.RevalidateReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid revalidate payload", false, nil)
			return
		}
		entries := make([]protocol.RevalidateEntry, 0, len(req.NodeIDs))
		for _, nodeID := range req.NodeIDs {
			entry, err := s.Backend.RevalidateNode(nodeID)
			if err != nil {
				entries = append(entries, protocol.RevalidateEntry{NodeID: nodeID, Exists: false, Error: err.Error()})
				continue
			}
			entries = append(entries, entry)
		}
		s.writeResponse(conn, header, protocol.OpcodeRevalidateResp, header.SessionID, protocol.RevalidateResp{Entries: entries})
	case protocol.OpcodeResubscribeReq:
		req, err := transport.DecodePayload[protocol.ResubscribeReq](payload)
		if err != nil {
			s.writeError(conn, header, protocol.ErrInvalidRequest, "invalid resubscribe payload", false, nil)
			return
		}
		results, err := s.Journal.Resubscribe(req.Watches)
		if err != nil {
			s.writeError(conn, header, protocol.ErrRecoveryFailed, err.Error(), false, nil)
			return
		}
		s.writeResponse(conn, header, protocol.OpcodeResubscribeResp, header.SessionID, protocol.ResubscribeResp{Watches: results})
	default:
		s.writeError(conn, header, protocol.ErrUnsupportedOp, fmt.Sprintf("unsupported recovery opcode %d", header.Opcode), false, nil)
	}
}

func (s *Server) requireActiveSession(conn net.Conn, header protocol.Header) bool {
	now := s.Now()
	_, found, active := s.SessionManager.ValidateActive(header.SessionID, now)
	if !found {
		s.writeError(conn, header, protocol.ErrSessionNotFound, "session not found", false, nil)
		return false
	}
	if !active {
		s.writeError(conn, header, protocol.ErrSessionExpired, "session expired", false, nil)
		return false
	}
	return true
}

func (s *Server) audit(category, action, outcome string, requestID, sessionID uint64, path string, details map[string]any) error {
	if s == nil || s.Audit == nil {
		return nil
	}
	return s.Audit.Log(AuditRecord{Category: category, Action: action, Outcome: outcome, RequestID: requestID, SessionID: sessionID, Path: path, Details: details})
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

func (s *Server) writeBackendError(conn net.Conn, reqHeader protocol.Header, err error) {
	switch {
	case errors.Is(err, os.ErrNotExist):
		s.writeError(conn, reqHeader, protocol.ErrNotFound, "not found", false, nil)
	case errors.Is(err, errAlreadyExists) || errors.Is(err, os.ErrExist):
		s.writeError(conn, reqHeader, protocol.ErrAlreadyExists, "already exists", false, nil)
	case errors.Is(err, errInvalidHandle):
		s.writeError(conn, reqHeader, protocol.ErrInvalidHandle, "invalid handle", false, nil)
	case errors.Is(err, errAccessDenied):
		s.writeError(conn, reqHeader, protocol.ErrAccessDenied, "access denied", false, nil)
	case isNotDir(err):
		s.writeError(conn, reqHeader, protocol.ErrNotDir, "not a directory", false, nil)
	case isDir(err):
		s.writeError(conn, reqHeader, protocol.ErrIsDir, "is a directory", false, nil)
	default:
		s.writeError(conn, reqHeader, protocol.ErrInternal, err.Error(), false, nil)
	}
}

func (s *Server) writeWatchError(conn net.Conn, reqHeader protocol.Header, err error) {
	if err != nil && err.Error() == "watch not found" {
		s.writeError(conn, reqHeader, protocol.ErrWatchNotFound, "watch not found", false, nil)
		return
	}
	s.writeBackendError(conn, reqHeader, err)
}

func stringsJoin(parentRel, name string) string {
	if parentRel == "" {
		return name
	}
	return filepath.Join(parentRel, name)
}
