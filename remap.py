import json

def main():
    with open ('proto/nameTranslation_2.7.0.txt', 'r') as f:
        lines = f.readlines()
        tLines = [l.split(' -> ') for l in lines if '->' in l]
        translations = {l[0]: l[1].strip("\n") for l in tLines}

        with open('field_mappings.json', 'w') as mf:
            mf.write(json.dumps(translations, indent='\t'))

    with open('field_mappings.json', 'r') as f:
        mappings = json.load(f)

    with open('proto/StarRail_2.7.0.proto', 'r') as f:
        proto = f.read()

    for k, v in mappings.items():
        proto = proto.replace(k, v)

    with open('proto/StarRail_2.7.0.translated.proto', 'w') as f:
        f.write(proto)

if __name__ == '__main__':
    main()