package transport

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"developer-mount/internal/protocol"
)

func EncodeFrame(header protocol.Header, payload any) ([]byte, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	header.Magic = [4]byte{'D', 'M', 'N', 'T'}
	header.Version = protocol.Version
	header.HeaderLength = protocol.HeaderLength
	header.PayloadLength = uint32(len(payloadBytes))

	frameLen := int(protocol.HeaderLength) + len(payloadBytes)
	buf := make([]byte, 4+frameLen)
	binary.BigEndian.PutUint32(buf[:4], uint32(frameLen))
	writeHeader(buf[4:4+protocol.HeaderLength], header)
	copy(buf[4+protocol.HeaderLength:], payloadBytes)
	return buf, nil
}

func DecodeFrame(r io.Reader) (protocol.Header, []byte, error) {
	var header protocol.Header
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return header, nil, err
	}
	frameLen := binary.BigEndian.Uint32(lenBuf)
	if frameLen < protocol.HeaderLength {
		return header, nil, fmt.Errorf("invalid frame length %d", frameLen)
	}
	frame := make([]byte, frameLen)
	if _, err := io.ReadFull(r, frame); err != nil {
		return header, nil, err
	}
	readHeader(frame[:protocol.HeaderLength], &header)
	if string(header.Magic[:]) != protocol.Magic {
		return header, nil, fmt.Errorf("invalid magic %q", string(header.Magic[:]))
	}
	if header.Version != protocol.Version {
		return header, nil, fmt.Errorf("unsupported protocol version %d", header.Version)
	}
	if header.HeaderLength != protocol.HeaderLength {
		return header, nil, fmt.Errorf("invalid header length %d", header.HeaderLength)
	}
	payload := frame[protocol.HeaderLength:]
	if uint32(len(payload)) != header.PayloadLength {
		return header, nil, fmt.Errorf("payload length mismatch header=%d actual=%d", header.PayloadLength, len(payload))
	}
	return header, payload, nil
}

func DecodePayload[T any](payload []byte) (T, error) {
	var out T
	err := json.Unmarshal(payload, &out)
	return out, err
}

func writeHeader(dst []byte, header protocol.Header) {
	copy(dst[0:4], header.Magic[:])
	dst[4] = header.Version
	dst[5] = header.HeaderLength
	dst[6] = byte(header.Channel)
	dst[7] = byte(header.Opcode)
	binary.BigEndian.PutUint32(dst[8:12], header.Flags)
	binary.BigEndian.PutUint64(dst[12:20], header.RequestID)
	binary.BigEndian.PutUint64(dst[20:28], header.SessionID)
	binary.BigEndian.PutUint32(dst[28:32], header.PayloadLength)
}

func readHeader(src []byte, header *protocol.Header) {
	copy(header.Magic[:], src[0:4])
	header.Version = src[4]
	header.HeaderLength = src[5]
	header.Channel = protocol.Channel(src[6])
	header.Opcode = protocol.Opcode(src[7])
	header.Flags = binary.BigEndian.Uint32(src[8:12])
	header.RequestID = binary.BigEndian.Uint64(src[12:20])
	header.SessionID = binary.BigEndian.Uint64(src[20:28])
	header.PayloadLength = binary.BigEndian.Uint32(src[28:32])
}
