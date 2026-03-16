package transport

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"testing"

	"developer-mount/internal/protocol"
)

func encodeTestFrame(t *testing.T) []byte {
	t.Helper()
	frame, err := EncodeFrame(protocol.Header{
		Channel:   protocol.ChannelControl,
		Opcode:    protocol.OpcodeHelloReq,
		Flags:     protocol.FlagRequest,
		RequestID: 1,
	}, protocol.HelloReq{
		ClientName:                "neg-client",
		ClientVersion:             "0.1.0",
		SupportedProtocolVersions: []uint8{protocol.Version},
		Capabilities:              protocol.DefaultCapabilities(),
	})
	if err != nil {
		t.Fatalf("EncodeFrame() error = %v", err)
	}
	return frame
}

func TestDecodeFrameNegativePaths(t *testing.T) {
	t.Run("short-length-prefix", func(t *testing.T) {
		_, _, err := DecodeFrame(bytes.NewReader([]byte{0x00, 0x01}))
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("DecodeFrame() error = %v, want %v", err, io.ErrUnexpectedEOF)
		}
	})

	t.Run("invalid-frame-length", func(t *testing.T) {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, protocol.HeaderLength-1)
		_, _, err := DecodeFrame(bytes.NewReader(buf))
		if err == nil || !strings.Contains(err.Error(), "invalid frame length") {
			t.Fatalf("DecodeFrame() error = %v, want invalid frame length", err)
		}
	})

	t.Run("truncated-frame-body", func(t *testing.T) {
		frame := encodeTestFrame(t)
		frame = frame[:len(frame)-3]
		_, _, err := DecodeFrame(bytes.NewReader(frame))
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("DecodeFrame() error = %v, want %v", err, io.ErrUnexpectedEOF)
		}
	})

	t.Run("invalid-magic", func(t *testing.T) {
		frame := encodeTestFrame(t)
		copy(frame[4:8], []byte("FAIL"))
		_, _, err := DecodeFrame(bytes.NewReader(frame))
		if err == nil || !strings.Contains(err.Error(), "invalid magic") {
			t.Fatalf("DecodeFrame() error = %v, want invalid magic", err)
		}
	})

	t.Run("unsupported-version", func(t *testing.T) {
		frame := encodeTestFrame(t)
		frame[8] = protocol.Version + 1
		_, _, err := DecodeFrame(bytes.NewReader(frame))
		if err == nil || !strings.Contains(err.Error(), "unsupported protocol version") {
			t.Fatalf("DecodeFrame() error = %v, want unsupported protocol version", err)
		}
	})

	t.Run("invalid-header-length", func(t *testing.T) {
		frame := encodeTestFrame(t)
		frame[9] = protocol.HeaderLength - 1
		_, _, err := DecodeFrame(bytes.NewReader(frame))
		if err == nil || !strings.Contains(err.Error(), "invalid header length") {
			t.Fatalf("DecodeFrame() error = %v, want invalid header length", err)
		}
	})

	t.Run("payload-length-mismatch", func(t *testing.T) {
		frame := encodeTestFrame(t)
		binary.BigEndian.PutUint32(frame[32:36], 1)
		_, _, err := DecodeFrame(bytes.NewReader(frame))
		if err == nil || !strings.Contains(err.Error(), "payload length mismatch") {
			t.Fatalf("DecodeFrame() error = %v, want payload length mismatch", err)
		}
	})
}
