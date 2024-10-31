import json

def main():
    # Load JSON data
    with open('proto/PacketIds.json', 'r', encoding='utf-8') as f:
        data = json.load(f)

    # Generate Go code for packet_ids.go
    go_code_ids = "package reliquary\n\n// Generated file, do not edit\n\nconst (\n"
    for key, value in data.items():
        go_code_ids += f"    {value} = {key}\n"
    go_code_ids += ")\n"

    # Write packet_ids.go file
    with open("packet_ids.go", "w") as file:
        file.write(go_code_ids)

    # Generate Go code for packet_names.go with map
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
    for key, value in data.items():
        go_code_names += f"    {key}: \"{value}\",\n"
    go_code_names += "}\n"

    # Write packet_names.go file
    with open("packet_names.go", "w") as file:
        file.write(go_code_names)

    # Generate Go code for packet_registry.go with map
    go_code_registry = """package reliquary
    
// Generated file, do not edit
    
import (
    "github.com/Fesaa/go-reliquary/pb"
    "google.golang.org/protobuf/proto"
)
    
// The commands with ids [5638, 4745, 4720, 4711, 42, 83, 2828] are not mapped
// these have not been correctly mapped in either the translation mappings
// Or the original proto file
var packetRegistry = map[uint16]func() proto.Message{
"""
    for key, value in data.items():
        # Translation for these ids are currently not correctly included in the protobuf
        if int(key) in [5638, 4745, 4720, 4711, 42, 83, 2828]:
            continue

        go_code_registry += f"    {key}: func() proto.Message {{ return &pb.{value}{{}} }},\n"
    go_code_registry += "}\n"

    # Write packet_registry.go file
    with open("packet_registry.go", "w") as file:
        file.write(go_code_registry)

    print("Generated packet_ids.go, packet_names.go, and packet_registry.go with const definitions, ID-to-name mapping, and ID-to-struct registry.")


if __name__ == '__main__':
    main()
