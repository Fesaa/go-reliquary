package reliquary

import (
	"encoding/binary"
	"fmt"
	"github.com/goark/mt/mt19937"
	"github.com/xtaci/kcp-go"
	"time"
)

// KcpSniffer is a structure that manages KCP connections.
type KcpSniffer struct {
	ConvID    uint32
	Kcp       *kcp.KCP
	TimeStart time.Time
}

// NewKcpSniffer creates a new KcpSniffer instance from the provided segment.
func NewKcpSniffer(segment []byte) (*KcpSniffer, error) {
	trace("creating new KcpSniffer", "segmentLen", len(segment))
	convID, err := validateKcpSegment(segment)
	if err != nil {
		return nil, fmt.Errorf("could not create new KCP instance: %w", err)
	}
	return &KcpSniffer{
		ConvID:    convID,
		Kcp:       newKcp(convID),
		TimeStart: time.Now(),
	}, nil
}

// ReceiveSegments processes incoming segments and returns received messages.
func (ks *KcpSniffer) ReceiveSegments(segments []byte) [][]byte {
	convID, err := validateKcpSegment(segments)
	if err != nil {
		return nil
	}

	if convID != ks.ConvID {
		logger.Warn("warning: packet did not belong to conversation", "expected", ks.ConvID)
		return nil
	}

	// Reformat segments to skip bytes 4..8
	segments = reformatKcpSegments(segments)

	if num := ks.Kcp.Input(segments, true, false); num < 0 {
		logger.Error("could not input to KCP", "code", num)
	}

	var recv [][]byte
	for {
		size := ks.Kcp.PeekSize()
		if size < 0 {
			break // No more messages
		}

		bytes := make([]byte, size)
		if num := ks.Kcp.Recv(bytes); num < 0 {
			logger.Error("could not receive from KCP", "code", num)
			continue
		}
		recv = append(recv, bytes)
	}

	//ks.Kcp.Update()
	return recv
}

// newKcp initializes a new KCP instance.
func newKcp(convID uint32) *kcp.KCP {
	n := kcp.NewKCP(convID, nil)
	n.WndSize(1024, 1024)
	return n
}

// validateKcpSegment checks the validity of the KCP segment and extracts the conversation ID.
func validateKcpSegment(payload []byte) (uint32, error) {
	if len(payload) <= kcp.IKCP_OVERHEAD {
		logger.Warn("kcp header was too short", "length", len(payload))
		return 0, fmt.Errorf("KCP header too short")
	}
	return binary.LittleEndian.Uint32(payload), nil
}

// reformatKcpSegments reformats the segments to skip bytes 4..8.
func reformatKcpSegments(data []byte) []byte {
	var reformattedBytes []byte
	i := 0
	for i < len(data) {
		convID := data[i : i+4]

		remainingHeader := data[i+8 : i+28]

		contentLen := binary.LittleEndian.Uint32(data[i+24 : i+28])
		content := data[i+28 : i+28+int(contentLen)]

		reformattedBytes = append(reformattedBytes, convID...)
		reformattedBytes = append(reformattedBytes, remainingHeader...)
		reformattedBytes = append(reformattedBytes, content...)

		i += 28 + int(contentLen)
	}
	return reformattedBytes
}

func NewKeyFromSeed(seed uint64) []byte {
	gen := mt19937.New((int64)(seed))

	key := make([]byte, 512)

	// Fill the key slice with random bytes
	for i := 0; i < 512; i += 8 {
		n := gen.Uint64()
		copy(key[i:i+8], []byte{
			byte(n >> 56), byte(n >> 48), byte(n >> 40), byte(n >> 32),
			byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n),
		})
	}
	return key
}
