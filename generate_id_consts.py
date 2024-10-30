import json

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
go_code_names = "package reliquary\n\n// Generated file, do not edit\n\nvar PacketNames = map[int]string{\n"
for key, value in data.items():
    go_code_names += f"    {key}: \"{value}\",\n"
go_code_names += "}\n"

# Write packet_names.go file
with open("packet_names.go", "w") as file:
    file.write(go_code_names)

print("Generated packet_ids.go and packet_names.go with const definitions and ID-to-name mapping.")
