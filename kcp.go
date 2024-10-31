package reliquary

import (
	"encoding/binary"
	"fmt"
	"github.com/fatedier/kcp-go"
	"github.com/rs/zerolog"
	"time"
)

type kcpSniffer struct {
	ConvID    uint32
	Kcp       *kcp.KCP
	TimeStart time.Time
	logger    zerolog.Logger
}

// newKcpSniffer creates a new kcpSniffer instance from the provided segment.
func newKcpSniffer(segment []byte) (*kcpSniffer, error) {
	logger.Info().Int("segmentLen", len(segment)).Msg("creating new kcpSniffer")

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
		logger:    logger.With().Uint32("convID", convID).Logger(),
	}, nil
}

func (ks *kcpSniffer) receive(segments []byte) ([][]byte, error) {
	convID, err := validateKcpSegment(segments)
	if err != nil {
		return nil, err
	}

	if convID != ks.ConvID {
		logger.Warn().
			Uint32("expected", ks.ConvID).
			Uint32("got", convID).
			Msg("invalid KCP segment")
		return nil, PacketNotFromConversation
	}

	segments = ks.reformatKcpSegments(segments)

	if num := ks.Kcp.Input(segments, true); num < 0 {
		ks.logger.Error().
			Int("code", num).
			Msg("could not input to KCP")
	} else {
		ks.logger.Debug().
			Int("size", len(segments)).
			Msg("input successful")
	}

	var recv [][]byte
	hasReadAny := false

	for {
		size := ks.Kcp.PeekSize()
		ks.logger.Trace().Int("size", size).Msg("reading bytes")
		if size < 0 {
			break // No more messages
		}
		hasReadAny = true

		bytes := make([]byte, size)
		if num := ks.Kcp.Recv(bytes); num < 0 {
			ks.logger.Error().
				Int("code", num).
				Msg("could not receive from KCP")
			continue
		}
		recv = append(recv, bytes)
	}

	ks.Kcp.Update(ks.clock())
	// No commands have been read as the data is split up.
	if !hasReadAny {
		return nil, nil
	}

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

	if isTraceEnabled() {
		ks.logger.Trace().
			Int("len", len(data)).
			Str("bytes", bytesAsHex(data)).
			Msg("before split")
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

	if isTraceEnabled() {
		ks.logger.Trace().
			Int("len", len(reformattedBytes)).
			Str("bytes", bytesAsHex(reformattedBytes)).
			Msg("after split")
	}

	return reformattedBytes
}

// validateKcpSegment checks the validity of the KCP segment and extracts the conversation ID.
func validateKcpSegment(payload []byte) (uint32, error) {
	if len(payload) <= kcp.IKCP_OVERHEAD {
		logger.Warn().
			Int("len", len(payload)).
			Msg("kcp header was too short")
		return 0, fmt.Errorf("KCP header too short")
	}
	return binary.LittleEndian.Uint32(payload), nil
}
