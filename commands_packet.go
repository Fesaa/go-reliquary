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

// Game command header.
//
//	Contains the type of the command in `Id`
//	and the data encoded in protobuf in `ProtoData`
//
//	## Bit Layout
//	| Bit indices     |  Type |  Name |
//	| - | - | - |
//
// |    0:4       |  `uint32`  |  Header (magic constant) |
//
//	|   0:6      |  `uint16`  |  Id |
//	|   6:8      |  `uint16`  |  HeaderLen (unsure) |
//	|   8:12     |  `uint32`  |  DataLen |
//	|   12:12+data_len+header_len |  variable  |  ProtoData |
//	|  -4:  |  `uint32`  |  Tail (magic constant) |
//
// See https://github.com/IceDynamix/reliquary/pull/2
// https://github.com/IceDynamix/reliquary/blob/90cd0dda892751743966d5cac080a6541be5188a/src/network/mod.rs#L90-L103
func gameCommandFromData(data []byte) (*GameCommand, error) {
	if isTraceEnabled() {
		logger.Trace().
			Int("len", len(data)).
			Str("bytes", bytesAsHex(data)).
			Msg("reading command from bytes")
	}

	if len(data) < HEADER_OVERHEAD {
		logger.Warn().
			Int("wanted", HEADER_OVERHEAD).
			Int("got", len(data)).
			Msg("header not complete, missing bytes")
		return nil, errors.New("header not complete")
	}

	commandId := binary.BigEndian.Uint16(data[4:6])
	headerLen := binary.BigEndian.Uint16(data[6:8])
	dataLen := binary.BigEndian.Uint32(data[8:12])

	finalIdx := 12 + (uint)(dataLen) + (uint)(headerLen)
	commandData := data[12:finalIdx]

	commandName, ok := packetNames[commandId]
	if !ok {
		logger.Warn().Uint16("commandId", commandId).Msg("received command with unknown name")
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
