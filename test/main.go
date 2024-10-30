package main

import (
	"fmt"
	"github.com/Fesaa/go-reliquary"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"log/slog"
)

func main() {
	handle, err := connect()
	if err != nil {
		panic(err)
	}

	defer handle.Close()

	reliquary.SetLevel(slog.LevelInfo)
	sniffer := reliquary.NewSniffer().
		Register(reliquary.ChessRogueRollDiceScRsp, LogProtoMessage).
		Register(reliquary.ChessRogueReRollDiceScRsp, LogProtoMessage).
		Register(reliquary.ChessRogueRollDiceCsReq, LogProtoMessage).
		Register(reliquary.ChessRogueReRollDiceCsReq, LogProtoMessage)

	go func() {
		for handlerErr := range sniffer.Errors() {
			slog.Error("error while handling command", "id", handlerErr.CmdId, "err", handlerErr.Err)
		}
	}()

	src := gopacket.NewPacketSource(handle, handle.LinkType())
	slog.Info("starting sniffer")
	for packet := range src.Packets() {
		if _, err = sniffer.ReadPacket(packet); err != nil {
			slog.Error("encountered an error while reading GamePacket", "error", err)
		}
	}
}

func LogProtoMessage(cmd reliquary.GameCommand, message proto.Message) error {
	marshaledJSON, err := protojson.Marshal(message)
	if err != nil {
		return err
	}
	fmt.Printf("%s(%d): %s\n\n", cmd.Name, cmd.Id, string(marshaledJSON))
	return nil
}

func connect() (*pcap.Handle, error) {
	live, err := pcap.OpenLive("en0", 65536, true, pcap.BlockForever)
	if err != nil {
		return nil, err
	}
	err = live.SetBPFFilter(reliquary.PCAP_FILTER)
	if err != nil {
		return nil, err
	}

	return live, nil
}
