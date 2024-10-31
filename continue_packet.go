package reliquary

// ContinuePacket is returned when the data did not construct a message yet, as it's part of a larger packet
type ContinuePacket struct{}

func (ContinuePacket) isGamePacket() {}

func (cp ContinuePacket) PacketType() PacketType {
	return ContinuePacketType
}
