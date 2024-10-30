package reliquary

import (
	"encoding/binary"
	"errors"
)

type CommandsPacket struct {
	Commands []GameCommand
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

func GameCommandFromData(data []byte) (*GameCommand, error) {
	if len(data) < HEADER_OVERHEAD {
		logger.Warn("header not complete, missing bytes", "wanted", HEADER_OVERHEAD, "got", len(data))
		return nil, errors.New("header not complete")
	}

	commandId := binary.BigEndian.Uint16(data[4:6])
	headerLen := binary.BigEndian.Uint16(data[6:8])
	dataLen := binary.BigEndian.Uint32(data[8:12])

	finalIdx := 12 + (uint)(dataLen) + (uint)(headerLen)
	commandData := data[12:finalIdx]

	commandName, ok := PacketNames[(int)(commandId)]
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
