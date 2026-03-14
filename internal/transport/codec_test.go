package transport

import (
	"bytes"
	"testing"

	"developer-mount/internal/protocol"
)

func TestEncodeDecodeFrameRoundTrip(t *testing.T) {
	header := protocol.Header{
		Channel:   protocol.ChannelControl,
		Opcode:    protocol.OpcodeHelloReq,
		Flags:     protocol.FlagRequest,
		RequestID: 7,
		SessionID: 99,
	}
	payload := protocol.HelloReq{
		ClientName:                "client",
		ClientVersion:             "0.1.0",
		SupportedProtocolVersions: []uint8{1},
		Capabilities:              protocol.DefaultCapabilities(),
	}
	frame, err := EncodeFrame(header, payload)
	if err != nil {
		t.Fatalf("EncodeFrame() error = %v", err)
	}
	gotHeader, gotPayload, err := DecodeFrame(bytes.NewReader(frame))
	if err != nil {
		t.Fatalf("DecodeFrame() error = %v", err)
	}
	if gotHeader.RequestID != header.RequestID {
		t.Fatalf("request id mismatch: got %d want %d", gotHeader.RequestID, header.RequestID)
	}
	decoded, err := DecodePayload[protocol.HelloReq](gotPayload)
	if err != nil {
		t.Fatalf("DecodePayload() error = %v", err)
	}
	if decoded.ClientName != payload.ClientName {
		t.Fatalf("client name mismatch: got %q want %q", decoded.ClientName, payload.ClientName)
	}
}
