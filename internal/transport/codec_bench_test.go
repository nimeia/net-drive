package transport

import (
	"bytes"
	"testing"

	"developer-mount/internal/protocol"
)

func BenchmarkEncodeDecodeFrame(b *testing.B) {
	header := protocol.Header{
		Channel:   protocol.ChannelData,
		Opcode:    protocol.OpcodeWriteReq,
		Flags:     protocol.FlagRequest,
		RequestID: 42,
		SessionID: 99,
	}
	payload := protocol.WriteReq{
		HandleID: 77,
		Offset:   5,
		Data:     []byte("benchmark-frame-payload"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frame, err := EncodeFrame(header, payload)
		if err != nil {
			b.Fatalf("EncodeFrame() error = %v", err)
		}
		if _, _, err := DecodeFrame(bytes.NewReader(frame)); err != nil {
			b.Fatalf("DecodeFrame() error = %v", err)
		}
	}
}
