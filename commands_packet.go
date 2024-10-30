package reliquary

import (
	"encoding/binary"
	"errors"
)

type CommandsPacket struct {
	_conn     *ConnectionPacket
	Direction Direction
	Commands  []GameCommand
}

type GameCommand struct {
	Id        uint16
	Name      string
	HeaderLen uint16
	DataLen   uint32
	ProtoData []byte
}

func (CommandsPacket) isGamePacket() {}

func (cp CommandsPacket) PacketType() PacketType {
	return CommandsPacketType
}

// ConnPacket returns the original SegmentData ConnectionPacket
func (cp CommandsPacket) ConnPacket() *ConnectionPacket {
	return cp._conn
}

func gameCommandFromData(data []byte) (*GameCommand, error) {
	if logger.IsTraceEnabled() {
		logger.Trace("reading command from bytes", "len", len(data), "bytes", bytesAsHex(data))
	}

	if len(data) < HEADER_OVERHEAD {
		logger.Warn("header not complete, missing bytes", "wanted", HEADER_OVERHEAD, "got", len(data))
		return nil, errors.New("header not complete")
	}

	commandId := binary.BigEndian.Uint16(data[4:6])
	headerLen := binary.BigEndian.Uint16(data[6:8])
	dataLen := binary.BigEndian.Uint32(data[8:12])

	finalIdx := 12 + (uint)(dataLen) + (uint)(headerLen)
	commandData := data[12:finalIdx]

	commandName, ok := PacketNames[commandId]
	if !ok {
		logger.Warn("received command with unknown name")
		commandName = ""
	}

	command := &GameCommand{
		Id:        commandId,
		Name:      commandName,
		HeaderLen: headerLen,
		DataLen:   dataLen,
		ProtoData: commandData,
	}
	return command, nil
}
