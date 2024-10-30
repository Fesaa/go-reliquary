package reliquary

import (
	"errors"
	"fmt"
	"github.com/Fesaa/go-reliquary/pb"
	"github.com/google/gopacket"
	"google.golang.org/protobuf/proto"
)

func NewSniffer() *Sniffer {
	return &Sniffer{}
}

type Sniffer struct {
	sentKcp *KcpSniffer
	recvKcp *KcpSniffer
	key     []byte
}

func (s *Sniffer) ReadPacket(packet gopacket.Packet) (GamePacket, error) {
	connPacket, err := parseConnectionPacket(packet)
	if err != nil {
		return nil, err
	}
	trace("received connection packet", "type", connPacket.Type, "payloadLength", len(connPacket.Payload))

	switch connPacket.Type {
	case HandshakeRequested:
		s.sentKcp = nil
		s.recvKcp = nil
		s.key = nil
		return connPacket, nil
	case HandshakeEstablished:
		return connPacket, nil
	case Disconnected:
		return connPacket, nil
	case SegmentData:
		commands, err := s.handleKCP(connPacket.Direction, connPacket.Payload)
		if err != nil {
			return nil, err
		}
		return &CommandsPacket{Commands: commands}, nil
	}

	logger.Warn("unhandled packet", "type", connPacket.Type)
	return nil, errors.New("unhandled packet")
}

func (s *Sniffer) handleKCP(direction Direction, segment []byte) ([]GameCommand, error) {
	dKcp := func() *KcpSniffer {
		switch direction {
		case Received:
			return s.recvKcp
		case Send:
			return s.sentKcp
		default:
			panic("invalid direction in packet")
		}
	}()

	if dKcp == nil {
		nKcp, err := NewKcpSniffer(segment)
		if err != nil {
			return nil, err
		}
		dKcp = nKcp
		switch direction {
		case Received:
			s.recvKcp = nKcp
		case Send:
			s.sentKcp = nKcp
		default:
			panic("invalid direction in packet")
		}
	}

	if dKcp == nil {
		return nil, errors.New("kcp is still nil, cannot construct commands")
	}

	commands := make([]GameCommand, 0)
	for _, data := range dKcp.ReceiveSegments(segment) {
		command, err := s.receiveCommand(data)
		if err != nil {
			return nil, err
		}
		trace("command packet read from SegmentData", "commandId", command.Id, "dataLen", len(command.ProtoData))
		commands = append(commands, *command)
	}

	return commands, nil
}

func (s *Sniffer) receiveCommand(data []byte) (*GameCommand, error) {
	key := func(data []byte) []byte {
		if s.key != nil {
			return s.key
		}
		return getKnownKey(data)
	}(data)

	decryptedCommandData := DecryptCommand(key, data)
	command, err := GameCommandFromData(decryptedCommandData)
	if err != nil {
		return nil, err
	}

	if command.Id == PlayerGetTokenScRsp {
		var playerGetTokenScRsp pb.PlayerGetTokenScRsp
		if err = proto.Unmarshal(command.ProtoData, &playerGetTokenScRsp); err != nil {
			return nil, fmt.Errorf("unable to unmarshal PlayerGetTokenScRsp: %v", err)
		}
		seed := playerGetTokenScRsp.SecretKeySeed
		s.key = NewKeyFromSeed(seed)
		logger.Info("new session key was set", "seed", seed)
	}

	return command, nil
}
