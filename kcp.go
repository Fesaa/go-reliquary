package reliquary

import (
	"encoding/binary"
	"fmt"
	"github.com/fatedier/kcp-go"
	"time"
)

type kcpSniffer struct {
	ConvID    uint32
	Kcp       *kcp.KCP
	TimeStart time.Time
	logger    *traceLogger
}

// newKcpSniffer creates a new kcpSniffer instance from the provided segment.
func newKcpSniffer(segment []byte) (*kcpSniffer, error) {
	logger.Info("creating new kcpSniffer", "segmentLen", len(segment))

	convID, err := validateKcpSegment(segment)
	if err != nil {
		return nil, fmt.Errorf("could not create new KCP instance: %w", err)
	}

	_kcp := kcp.NewKCP(convID, func(buf []byte, size int) {}) // ignore output
	_kcp.WndSize(1024, 1024)

	return &kcpSniffer{
		ConvID:    convID,
		Kcp:       _kcp,
		TimeStart: time.Now(),
		logger:    logger.WithArgs("convID", convID),
	}, nil
}

func (ks *kcpSniffer) receive(segments []byte) ([][]byte, error) {
	convID, err := validateKcpSegment(segments)
	if err != nil {
		return nil, err
	}

	if convID != ks.ConvID {
		logger.Warn("warning: packet did not belong to conversation", "expected", ks.ConvID)
		return nil, PacketNotFromConversation
	}

	segments = ks.reformatKcpSegments(segments)

	if num := ks.Kcp.Input(segments, true); num < 0 {
		ks.logger.Error("could not input to KCP", "code", num)
	} else {
		ks.logger.Trace("input successful", "size", len(segments))
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

	ks.Kcp.Update(ks.clock())
	return recv, nil
}

func (ks *kcpSniffer) clock() uint32 {
	now := time.Now()
	if ks.TimeStart.After(now) {
		panic("time went backwards")
	}
	return uint32(now.Sub(ks.TimeStart).Milliseconds())
}

// reformatKcpSegments reformats the segments to skip bytes 4..8.
func (ks *kcpSniffer) reformatKcpSegments(data []byte) []byte {
	var reformattedBytes []byte

	if logger.IsTraceEnabled() {
		ks.logger.Trace("before split", "bytes", bytesAsHex(data), "len", len(data))
	}

	var i uint = 0
	for i < uint(len(data)) {
		convID := data[i : i+4]

		remainingHeader := data[i+8 : i+28]

		contentLen := uint(binary.LittleEndian.Uint32(data[i+24 : i+28]))
		content := data[i+28 : i+28+contentLen]

		reformattedBytes = append(reformattedBytes, convID...)
		reformattedBytes = append(reformattedBytes, remainingHeader...)
		reformattedBytes = append(reformattedBytes, content...)

		i += 28 + contentLen
	}

	if ks.logger.IsTraceEnabled() {
		ks.logger.Trace("after split", "bytes", bytesAsHex(reformattedBytes), "len", len(reformattedBytes))
	}

	return reformattedBytes
}

// validateKcpSegment checks the validity of the KCP segment and extracts the conversation ID.
func validateKcpSegment(payload []byte) (uint32, error) {
	if len(payload) <= kcp.IKCP_OVERHEAD {
		logger.Warn("kcp header was too short", "length", len(payload))
		return 0, fmt.Errorf("KCP header too short")
	}
	return binary.LittleEndian.Uint32(payload), nil
}
