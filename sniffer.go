package reliquary

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/Fesaa/go-reliquary/pb"
	"github.com/google/gopacket"
	"google.golang.org/protobuf/proto"
)

type Sniffer struct {
	sentKcp     *kcpSniffer
	recvKcp     *kcpSniffer
	key         *Key
	initialKeys map[uint32]*Key
}

// ReadPacket reads a packet, and returns the correct GamePacket
// You can handle pb conversion yourself by checking the PacketType against CommandsPacketType
func (s *Sniffer) ReadPacket(packet gopacket.Packet) (GamePacket, error) {
	connPacket, err := parseConnectionPacket(packet)
	if err != nil {
		return nil, err
	}
	l := logger.With().
		Any("packetType", string(connPacket.Type)).
		Any("direction", connPacket.Direction).
		Int("len", len(connPacket.Payload)).
		Logger()
	l.Debug().Msg("Start packet")
	defer l.Debug().Msg("End packet")

	if isTraceEnabled(l) {
		l.Trace().
			Str("bytes", bytesAsHex(connPacket.Payload)).
			Msg("received connection packet")
	}

	switch connPacket.Type {
	case HandshakeRequested:
		s.sentKcp = nil
		s.recvKcp = nil
		s.key = nil
		l.Info().Msg("state reset after HandshakeRequested packet")
		return connPacket, nil
	case HandshakeEstablished:
		return connPacket, nil
	case Disconnected:
		return connPacket, nil
	case SegmentData:
		var commands []GameCommand
		if commands, err = s.read(connPacket.Direction, connPacket.Payload); err != nil {
			return nil, err
		}
		if commands == nil {
			return &ContinuePacket{}, nil
		}

		if len(commands) == 0 {
			l.Warn().Msg("received empty commands list")
		}

		return &CommandsPacket{
			_conn:     connPacket,
			Commands:  commands,
			Direction: connPacket.Direction,
		}, nil
	}

	l.Warn().Int("len", len(connPacket.Payload)).Msg("unhandled packet")
	return nil, errors.New("unhandled packet")
}

func (s *Sniffer) read(direction Direction, segment []byte) ([]GameCommand, error) {
	dKcp := s.getKCP(direction)

	if dKcp == nil {
		nKcp, err := newKcpSniffer(segment)
		if err != nil {
			return nil, err
		}
		dKcp = nKcp
		s.setKCP(direction, nKcp)
	}

	if dKcp == nil {
		return nil, errors.New("kcp is still nil, cannot construct commands")
	}

	commands := make([]GameCommand, 0)
	splitData, err := dKcp.receive(segment)
	if err != nil {
		return nil, err
	}

	// The command was split up in smaller packages. And will be returned when the last part has arrived
	if splitData == nil {
		return nil, nil
	}

	for _, data := range splitData {
		command, err := s.readCommand(data)
		if err != nil {
			return nil, err
		}
		logger.Trace().
			Uint16("id", command.Id).
			Int("dataLen", len(command.ProtoData)).
			Msg("command packet read from SegmentData")
		commands = append(commands, *command)
	}

	return commands, nil
}

func (s *Sniffer) getKCP(direction Direction) *kcpSniffer {
	switch direction {
	case Received:
		return s.recvKcp
	case Send:
		return s.sentKcp
	default:
		panic(fmt.Sprintf("cannot get KCP for %d direction", direction))
	}
}

func (s *Sniffer) setKCP(direction Direction, kcp *kcpSniffer) {
	switch direction {
	case Received:
		s.recvKcp = kcp
	case Send:
		s.sentKcp = kcp
	default:
		panic(fmt.Sprintf("cannot set KCP for %d direction", direction))
	}
}

func (s *Sniffer) getKey(data []byte) *Key {
	if s.key != nil {
		return s.key
	}

	if key, ok := s.initialKeys[version(data)]; ok {
		return key
	}

	return getKnownKey(data)
}

func (s *Sniffer) readCommand(data []byte) (*GameCommand, error) {
	key := s.getKey(data)

	decryptedCommandData := key.decryptCommand(data)
	command, err := gameCommandFromData(decryptedCommandData)
	if err != nil {
		return nil, err
	}

	if isTraceEnabled() {
		logger.Trace().Str("data", base64.StdEncoding.EncodeToString(command.ProtoData)).Msg("received")
	}

	if command.Id == PlayerGetTokenScRsp {
		var playerGetTokenScRsp pb.PlayerGetTokenScRsp
		if err = proto.Unmarshal(command.ProtoData, &playerGetTokenScRsp); err != nil {
			return nil, fmt.Errorf("unable to unmarshal PlayerGetTokenScRsp: %v", err)
		}
		seed := playerGetTokenScRsp.SecretKeySeed

		keyBytes := newKeyBytesFromSeed(seed)
		s.key = &Key{_bytes: keyBytes}
		logger.Info().Uint64("seed", seed).Msg("new session Key was set")
	}

	return command, nil
}
