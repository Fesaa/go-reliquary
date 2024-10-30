package reliquary

import (
	"fmt"
	"github.com/Fesaa/go-reliquary/pb"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestOffline(t *testing.T) {
	handle, err := pcap.OpenOffline("./login_dump.pcapng")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	snf := NewSniffer()
	src := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range src.Packets() {
		p, err := snf.ReadPacket(packet)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		if p == nil {
			logger.Warn("got a nil packet?", "err", err)
			continue
		}

		switch p.PacketType() {
		case ConnectionPacketType:
			logger.Info("found connection packet", "type", p.(*ConnectionPacket).Type)
		case CommandsPacketType:
			logger.Info("found commands packet", "len", len(p.(*CommandsPacket).Commands))
		}
	}
}

func TestLive(t *testing.T) {
	live, err := pcap.OpenLive("en0", 65536, true, pcap.BlockForever)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	err = live.SetBPFFilter(PCAP_FILTER)
	if err != nil {
		t.Error(err)
		return
	}

	snf := NewSniffer()
	src := gopacket.NewPacketSource(live, live.LinkType())

	logger.Info("starting reading...")
	for packet := range src.Packets() {
		p, err := snf.ReadPacket(packet)
		if err != nil {
			t.Error(err)
			t.Fail()
			return
		}

		switch p.PacketType() {
		case ConnectionPacketType:
			logger.Info("found connection packet", "type", p.(*ConnectionPacket).Type)
		case CommandsPacketType:
			logger.Debug("found commands packet", "len", len(p.(*CommandsPacket).Commands))
			for _, cmd := range p.(*CommandsPacket).Commands {
				switch cmd.Id {
				case ChessRogueRollDiceScRsp:
					var chessRogueRollDiceScRsp pb.ChessRogueRollDiceScRsp
					if err = proto.Unmarshal(cmd.ProtoData, &chessRogueRollDiceScRsp); err != nil {
						t.Error(err)
					} else {
						logger.Info("unmarshaled ChessRogueRollDiceScRsp successfully", "res", fmt.Sprintf("%+v", chessRogueRollDiceScRsp))
					}
				}

				//logger.Info("found command", "cmd", cmd.Id, "dataLen", len(cmd.ProtoData))
			}
		}
	}
}
