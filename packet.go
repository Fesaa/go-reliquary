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
)

func packetCode(payload []byte) uint32 {
	codeBytes := payload[:4]
	return binary.BigEndian.Uint32(codeBytes)
}
