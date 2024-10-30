package reliquary

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/Fesaa/go-reliquary/pb"
	"github.com/google/gopacket"
	"google.golang.org/protobuf/proto"
)

func NewSniffer() *Sniffer {
	return &Sniffer{
		handlerRegistry: make(map[uint16]Handler),
		errorCh:         make(chan HandlerError),
	}
}

type Sniffer struct {
	sentKcp     *kcpSniffer
	recvKcp     *kcpSniffer
	key         *Key
	initialKeys map[uint32]*Key

	handlerRegistry map[uint16]Handler
	errorCh         chan HandlerError
}

type Handler func(cmd GameCommand, msg proto.Message) error

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
// This assumes you passed the correct commandId. Will panic if not
func (s *Sniffer) Register(commandId uint16, handlers ...Handler) *Sniffer {
	for _, h := range handlers {
		s.register(commandId, h)
	}
	return s
}

func (s *Sniffer) register(commandId uint16, handler Handler) {
	if handler == nil {
		panic("handler must be non nil")
	}
	_, ok := packetRegistry[commandId]
	if !ok {
		panic(fmt.Sprintf("cannot register handler for unknown command %d", commandId))
	}
	s.handlerRegistry[commandId] = handler
	logger.Debug("handler registered for command", "id", commandId, "name", PacketNames[commandId])
}

func (s *Sniffer) fireHandler(commands []GameCommand) {
	for _, cmd := range commands {
		l := logger.WithArgs("id", cmd.Id, "name", cmd.Name)
		handler, ok := s.handlerRegistry[cmd.Id]
		if !ok {
			l.Trace("no handler for command")
			continue
		}

		var msg = packetRegistry[cmd.Id]()
		if err := proto.Unmarshal(cmd.ProtoData, msg); err != nil {
			l.Error("failed to unmarshal packet", "err", err)
			s.propagate(cmd, fmt.Errorf("cannot unmarshal protobuf packet: %w", err))
			continue
		}

		l.Trace("firing handler")
		if err := handler(cmd, msg); err != nil {
			l.Warn("handler error", "err", err)
			s.propagate(cmd, err)
		}
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
	l := logger.WithArgs("packetType", connPacket.Type, "direction", connPacket.Direction)

	if l.IsTraceEnabled() {
		l.Trace("received connection packet", "payloadLength", len(connPacket.Payload), "bytes", bytesAsHex(connPacket.Payload))
	}

	switch connPacket.Type {
	case HandshakeRequested:
		s.sentKcp = nil
		s.recvKcp = nil
		s.key = nil
		l.Info("state reset after HandshakeRequested packet")
		return connPacket, nil
	case HandshakeEstablished:
		return connPacket, nil
	case Disconnected:
		return connPacket, nil
	case SegmentData:
		var commands []GameCommand
		if commands, err = s.handleKCP(connPacket.Direction, connPacket.Payload); err != nil {
			return nil, err
		}
		s.fireHandler(commands)
		return &CommandsPacket{
			_conn:     connPacket,
			Commands:  commands,
			Direction: connPacket.Direction,
		}, nil
	}

	l.Warn("unhandled packet", "len", len(connPacket.Payload))
	return nil, errors.New("unhandled packet")
}

func (s *Sniffer) handleKCP(direction Direction, segment []byte) ([]GameCommand, error) {
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
	for _, data := range splitData {
		command, err := s.readCommand(data)
		if err != nil {
			return nil, err
		}
		logger.Trace("command packet read from SegmentData", "commandId", command.Id, "dataLen", len(command.ProtoData))
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

	if logger.IsTraceEnabled() {
		logger.Trace("received", "data", base64.StdEncoding.EncodeToString(command.ProtoData))
	}

	if command.Id == PlayerGetTokenScRsp {
		var playerGetTokenScRsp pb.PlayerGetTokenScRsp
		if err = proto.Unmarshal(command.ProtoData, &playerGetTokenScRsp); err != nil {
			return nil, fmt.Errorf("unable to unmarshal PlayerGetTokenScRsp: %v", err)
		}
		seed := playerGetTokenScRsp.SecretKeySeed

		keyBytes := newKeyBytesFromSeed(seed)
		s.key = &Key{_bytes: keyBytes}
		logger.Info("new session Key was set", "seed", seed)
	}

	return command, nil
}
