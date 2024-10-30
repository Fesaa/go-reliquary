package reliquary

import (
	"errors"
	"fmt"
	"github.com/Fesaa/go-reliquary/pb"
	"github.com/google/gopacket"
	"google.golang.org/protobuf/proto"
)

func NewSniffer() *Sniffer {
	return &Sniffer{
		handlerRegistry: make(map[uint16]func(cmd GameCommand, msg proto.Message) error),
		errorCh:         make(chan HandlerError),
	}
}

type Sniffer struct {
	sentKcp *KcpSniffer
	recvKcp *KcpSniffer
	key     []byte

	handlerRegistry map[uint16]func(cmd GameCommand, msg proto.Message) error
	errorCh         chan HandlerError
}

type HandlerError struct {
	CmdId uint16
	Err   error
}

func (h *HandlerError) Error() string {
	return fmt.Sprintf("handler %d error: %v", h.CmdId, h.Err)
}

// Errors returns the channel where handler errors are propagated to
func (s *Sniffer) Errors() <-chan HandlerError {
	return s.errorCh
}

func (s *Sniffer) propagate(cmd GameCommand, err error) {
	s.errorCh <- HandlerError{
		CmdId: cmd.Id,
		Err:   err,
	}
}

// Register a handler for the passed commandId, the msg in the function can be cast to the correct pb struct
// This assumes you passed the correct commandId.
func (s *Sniffer) Register(commandId uint16, handler func(cmd GameCommand, msg proto.Message) error) *Sniffer {
	_, ok := packetRegistry[commandId]
	if !ok {
		panic(fmt.Sprintf("cannot register handler for unknown command %d", commandId))
	}
	s.handlerRegistry[commandId] = handler
	logger.Debug("handler registered for command", "id", commandId, "name", PacketNames[commandId])
	return s
}

func (s *Sniffer) fireHandler(commands []GameCommand) {
	for _, cmd := range commands {
		l := logger.With("id", cmd.Id, "name", cmd.Name)
		handler, ok := s.handlerRegistry[cmd.Id]
		if !ok {
			traceL(l, "no handler for command")
			continue
		}

		var msg = packetRegistry[cmd.Id]()
		err := proto.Unmarshal(cmd.ProtoData, msg)
		if err != nil {
			l.Debug("failed to unmarshal packet")
			s.propagate(cmd, fmt.Errorf("cannot unmarshal protobuf packet: %w", err))
			continue
		}

		go func() {
			traceL(l, "firing handler")
			if err = handler(cmd, msg); err != nil {
				l.Debug("handler error")
				s.propagate(cmd, err)
			}
		}()
	}
}

// ReadPacket reads a packet, and returns the correct GamePacket
// You can handle pb conversion yourself by checking the PacketType against CommandsPacketType
// Consider using Sniffer.Register
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
		s.fireHandler(commands)
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
