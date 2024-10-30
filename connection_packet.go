package reliquary

import (
	"errors"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type ConnectionPacket struct {
	Type      ConnectionType
	Direction Direction
	Payload   []byte
}

func (ConnectionPacket) isGamePacket() {}

func (cp ConnectionPacket) PacketType() PacketType {
	return ConnectionPacketType
}

type ConnectionType string

const (
	HandshakeRequested   ConnectionType = "HandshakeRequested"
	Disconnected         ConnectionType = "Disconnected"
	HandshakeEstablished ConnectionType = "HandshakeEstablished"
	SegmentData          ConnectionType = "SegmentData"
)

func parseConnectionPacket(packet gopacket.Packet) (*ConnectionPacket, error) {
	updLayer := packet.Layer(layers.LayerTypeUDP)
	if updLayer == nil {
		return nil, errors.New("no udp packet found")
	}
	udp := updLayer.(*layers.UDP)
	direction := DirectionFromUdp(udp)

	if len(udp.Payload) <= 20 {
		switch packetCode(udp.Payload) {
		case 0xFF:
			logger.Debug("handshake requested")
			return &ConnectionPacket{Type: HandshakeRequested, Direction: direction}, nil
		case 404:
			logger.Debug("disconnected")
			return &ConnectionPacket{Type: Disconnected, Direction: direction}, nil
		default:
			logger.Debug("handshake established")
			return &ConnectionPacket{Type: HandshakeEstablished, Direction: direction}, nil
		}
	}

	return &ConnectionPacket{Type: SegmentData, Direction: direction, Payload: udp.Payload}, nil
}
