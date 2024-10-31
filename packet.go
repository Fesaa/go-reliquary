package reliquary

import "encoding/binary"

type GamePacket interface {
	isGamePacket()
	PacketType() PacketType
}

type PacketType byte

const (
	ConnectionPacketType PacketType = iota
	CommandsPacketType   PacketType = iota
	// ContinuePacketType the underlying packet did not include a full message
	// The CommandsPacket will return from the Sniffer at a later point
	ContinuePacketType PacketType = iota
)

func packetCode(payload []byte) uint32 {
	codeBytes := payload[:4]
	return binary.BigEndian.Uint32(codeBytes)
}
