import json

def main():
    with open('field_mappings.json', 'r') as f:
        mappings = json.load(f)

    with open('proto/StarRail_2.6.0_detailed.proto', 'r') as f:
        proto = f.read()

    for k, v in mappings.items():
        proto = proto.replace(k, v)

    with open('proto/StarRail_2.6.0_detailed.translated.proto', 'w') as f:
        f.write(proto)

if __name__ == '__main__':
    main()