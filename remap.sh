
python3.13 remap.py


PROTO_FILE="proto/StarRail_2.7.0.translated.proto"
protoc --go_out=./pb --go_opt=Mproto/StarRail_2.7.0.translated.proto=github.com/Fesaa/go-reliquary/pb ${PROTO_FILE}
mv pb/github.com/Fesaa/go-reliquary/pb/StarRail_2.7.0.translated.pb.go pb/generated.translated.pb.go
rm -rf pb/github.com