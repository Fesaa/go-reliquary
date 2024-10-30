package reliquary

import (
	"github.com/google/gopacket/layers"
	"slices"
)

type Direction byte

const (
	Received Direction = iota
	Send     Direction = iota
	Unknown  Direction = iota
)

func (d Direction) String() string {
	switch d {
	case Received:
		return "Received"
	case Send:
		return "Send"
	case Unknown:
		return "Unknown"
	default:
		return "Unknown (unset)"
	}
}

func directionFromUdp(udp *layers.UDP) Direction {
	if slices.Contains(PORTS, udp.DstPort.String()) {
		return Received
	}

	if slices.Contains(PORTS, udp.SrcPort.String()) {
		return Send
	}

	logger.Warn("packet found with unknown direction", "dst", udp.DstPort.String(), "src", udp.SrcPort.String())
	return Unknown
}
