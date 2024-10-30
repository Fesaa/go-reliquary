package reliquary

import (
	"encoding/binary"
	"fmt"
	"github.com/fatedier/kcp-go"
	"github.com/goark/mt/mt19937"
	"log/slog"
	"time"
)

// KcpSniffer is a structure that manages KCP connections.
type KcpSniffer struct {
	ConvID    uint32
	Kcp       *kcp.KCP
	TimeStart time.Time
	logger    *slog.Logger
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
		logger:    logger.With("convID", convID),
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

	if num := ks.Kcp.Input(segments, true); num < 0 {
		ks.logger.Error("could not input to KCP", "code", num)
	} else {
		traceL(ks.logger, "input successful", "size", len(segments))
	}

	var recv [][]byte
	for {
		size := ks.Kcp.PeekSize()
		if size < 0 {
			break // No more messages
		}

		bytes := make([]byte, size)
		if num := ks.Kcp.Recv(bytes); num < 0 {
			ks.logger.Error("could not receive from KCP", "code", num)
			continue
		}
		recv = append(recv, bytes)
	}

	ks.Kcp.Update(ks.Clock())
	return recv
}

func (ks *KcpSniffer) Clock() uint32 {
	now := time.Now()
	if ks.TimeStart.After(now) {
		panic("time went backwards")
	}
	return uint32(now.Sub(ks.TimeStart).Milliseconds())
}

// newKcp initializes a new KCP instance.
func newKcp(convID uint32) *kcp.KCP {
	n := kcp.NewKCP(convID, func(buf []byte, size int) {
		// ignore
	})
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

	if isTraceEnabled() {
		trace("before split", "bytes", bytesAsHex(data), "len", len(data))
	}

	var i uint = 0
	for i < uint(len(data)) {
		convID := data[i : i+4]

		remainingHeader := data[i+8 : i+28]

		contentLen := uint(binary.LittleEndian.Uint32(data[i+24 : i+28]))
		trace("contentLen", "len", contentLen, "curPos", i)
		content := data[i+28 : i+28+contentLen]

		reformattedBytes = append(reformattedBytes, convID...)
		reformattedBytes = append(reformattedBytes, remainingHeader...)
		reformattedBytes = append(reformattedBytes, content...)

		i += 28 + contentLen
	}

	if isTraceEnabled() {
		trace("after split", "bytes", bytesAsHex(reformattedBytes), "len", len(reformattedBytes))
	}

	return reformattedBytes
}

func NewKeyFromSeed(seed uint64) []byte {
	gen := mt19937.New((int64)(seed))

	key := make([]byte, 0)

	// Fill the key slice with random bytes
	for i := 0; i < 512; i++ {
		n := gen.Uint64()
		key = binary.BigEndian.AppendUint64(key, n)
	}
	return key
}
