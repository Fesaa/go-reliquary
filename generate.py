import json

def main():
    with open('proto/PacketIds.json', 'r', encoding='utf-8') as f:
        data = json.load(f)

    go_code_ids = "package reliquary\n\n// Generated file, do not edit\n\nconst (\n"
    for key, value in sorted(data.items(), key=lambda item: item[1]):
        go_code_ids += f"    {value} = {key}\n"
    go_code_ids += ")\n"

    with open("packet_ids.go", "w") as file:
        file.write(go_code_ids)

    go_code_names = """package reliquary
    
// Generated file, do not edit

// PacketName return the name of packet by id
// returns an empty string, if the passed id is invalid
func PacketName(id uint16) string {
    if name, ok := packetNames[id]; ok {
        return name
    }
    return ""
}

var packetNames = map[uint16]string{ 
"""
    for key, value in sorted(data.items(), key=lambda item: int(item[0])):
        go_code_names += f"    {key}: \"{value}\",\n"
    go_code_names += "}\n"

    with open("packet_names.go", "w") as file:
        file.write(go_code_names)

    go_code_registry = """package reliquary

// Generated file, do not edit

import (
    "github.com/Fesaa/go-reliquary/pb"
    "google.golang.org/protobuf/proto"
)

func PacketProto(id uint16) proto.Message {
    if f, ok := packetRegistry[id]; ok {
        return f()
    }
    return nil
}

// Some command ids are not mapping correctly. We manually filter them out
// these have not been correctly mapped in either the translation mappings
// Or the original proto file
var packetRegistry = map[uint16]func() proto.Message {
"""
    for key, value in data.items():
        # Translation for these ids are currently not correctly included in the protobuf
        if int(key) in [4795, 4796, 4739, 58, 24, 2828, 5695, 5691, 5618, 5611, 5605, 5617, 8038, 8081]:
            continue

        go_code_registry += f"    {key}: func() proto.Message {{ return &pb.{value}{{}} }},\n"
    go_code_registry += "}\n"

    with open("packet_registry.go", "w") as file:
        file.write(go_code_registry)

    print("Generated packet_ids.go, packet_names.go, and packet_registry.go with const definitions, ID-to-name mapping, and ID-to-struct registry.")


if __name__ == '__main__':
    main()
