import json
import os

def main():
    version = os.getenv("VERSION")


    with open('field_mappings.json', 'r') as f:
        mappings = json.load(f)

    with open (f'proto/nameTranslation_{version}.txt', 'r') as f:
        lines = f.readlines()
        tLines = [l.split(' -> ') for l in lines if '->' in l]
        translations = {l[0]: l[1].strip("\n") for l in tLines}

        # Add custom translation back in
        for k, v in mappings.items():
            if k not in translations.keys():
                translations[k] = v

        with open('field_mappings.json', 'w') as mf:
            mf.write(json.dumps(translations, indent='\t'))

    with open(f'proto/StarRail_{version}.proto', 'r') as f:
        proto = f.read()

    for k, v in mappings.items():
        proto = proto.replace(k, v)

    with open(f'proto/StarRail_{version}.translated.proto', 'w') as f:
        f.write(proto)

if __name__ == '__main__':
    main()