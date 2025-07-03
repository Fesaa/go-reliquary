package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Fesaa/go-reliquary"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"log"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"sync"
)

var (
	ignoreIds = []uint16{
		reliquary.PlayerHeartBeatScRsp,
		reliquary.PlayerHeartBeatCsReq,
		reliquary.SceneEntityMoveCsReq,
		reliquary.SceneEntityMoveScRsp,
		reliquary.SceneCastSkillCsReq,
		reliquary.SceneCastSkillScRsp,
		reliquary.GateServerScNotify,
		reliquary.GetBasicInfoCsReq,
		reliquary.GetBasicInfoScRsp,
	}
	mu sync.Mutex

	logIds = []uint16{
		reliquary.ChessRogueRollDiceScRsp,
		reliquary.ChessRogueReRollDiceScRsp,
		reliquary.ChessRogueRollDiceCsReq,
		reliquary.ChessRogueReRollDiceCsReq,
		reliquary.ChessRogueUpdateDiceInfoScNotify,
		reliquary.ChessRogueCheatRollCsReq,
		reliquary.ChessRogueCheatRollScRsp,
		reliquary.ChessRogueConfirmRollCsReq,
		reliquary.RogueModifierSelectCellCsReq,
		reliquary.RogueModifierUpdateNotify,
		reliquary.ChessRogueCellUpdateNotify,
		reliquary.RogueModifierSelectCellScRsp,
		reliquary.ChessRogueUpdateMoneyInfoScNotify,
		reliquary.ChessRogueStartScRsp,
		reliquary.ChessRogueStartCsReq,
		reliquary.GetPlayerBoardDataScRsp,
		reliquary.PlayerLoginCsReq,
	}
)

var sniffer *reliquary.Sniffer

func main() {
	go startHttpServer()

	//handle, err := connect()
	handle, err := read("./hsr.pcapng")
	if err != nil {
		panic(err)
	}

	defer handle.Close()

	reliquary.SetLogLevel(zerolog.InfoLevel)
	sniffer = &reliquary.Sniffer{}

	src := gopacket.NewPacketSource(handle, handle.LinkType())
	slog.Info("starting sniffer")
	for packet := range src.Packets() {
		var p reliquary.GamePacket
		if p, err = sniffer.ReadPacket(packet); err != nil {
			slog.Error("encountered an error while reading GamePacket", "error", err)
			continue
		}

		if p.PacketType() != reliquary.CommandsPacketType {
			continue
		}

		commandsPacket := p.(*reliquary.CommandsPacket)
		for i, cmd := range commandsPacket.Commands {
			handleCmd(i, cmd)
		}
	}
}

func handleCmd(i int, cmd reliquary.GameCommand) {
	mu.Lock()
	defer mu.Unlock()

	if slices.Contains(logIds, cmd.Id) {
		msg := reliquary.PacketProto(cmd.Id)
		if err := proto.Unmarshal(cmd.ProtoData, msg); err != nil {
			slog.Error("encountered an error while unmarshalling proto data", "error", err)
			return
		}

		if err := LogProtoMessage(cmd, msg); err != nil {
			slog.Error("encountered an error while logging proto message", "error", err)
			return
		}

		return
	}

	if slices.Contains(ignoreIds, cmd.Id) {
		return
	}

	fmt.Printf("[%d] %s(%d)\n", i, cmd.Name, cmd.Id)
}

func LogProtoMessage(cmd reliquary.GameCommand, message proto.Message) error {
	marshaledJSON, err := protojson.Marshal(message)
	if err != nil {
		return err
	}

	var indentedJson bytes.Buffer
	err = json.Indent(&indentedJson, marshaledJSON, "", "  ")
	if err != nil {
		return err
	}

	//fmt.Printf("%s(%d):\n %s\n\n", cmd.Name, cmd.Id, indentedJson.String())
	return nil
}

func read(path string) (*pcap.Handle, error) {
	handle, err := pcap.OpenOffline(path)
	if err != nil {
		return nil, err
	}

	return handle, nil
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

func startHttpServer() {
	http.HandleFunc("/", addIDHandler)
	http.HandleFunc("/add/", addFullPrintHandler)

	fmt.Println("Starting server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func addIDHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/"):]

	// Convert id to uint16
	id, err := strconv.ParseUint(idStr, 10, 16)
	if err != nil {
		http.Error(w, "Invalid ID: must be an integer between 0 and 65535", http.StatusBadRequest)
		return
	}

	// Lock before adding to list to ensure thread safety
	mu.Lock()
	ignoreIds = append(ignoreIds, uint16(id))
	mu.Unlock()

	_, _ = fmt.Fprintf(w, "ID %d added to list\n", id)
}

func addFullPrintHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/add/"):]

	id, err := strconv.ParseUint(idStr, 10, 16)
	if err != nil {
		http.Error(w, "Invalid ID: must be an integer between 0 and 65535", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	logIds = append(logIds, uint16(id))

	_, _ = fmt.Fprintf(w, "ID %d added to list\n", id)
}
