export VERSION="3.5.0"

PROTO_FILE="proto/StarRail_${VERSION}.translated.proto"
python3 remap.py

protoc --go_out=./pb --go_opt=Mproto/StarRail_${VERSION}.translated.proto=github.com/Fesaa/go-reliquary/pb ${PROTO_FILE}
mv pb/github.com/Fesaa/go-reliquary/pb/StarRail_${VERSION}.translated.pb.go pb/generated.translated.pb.go
rm -rf pb/github.com
python3 generate.py