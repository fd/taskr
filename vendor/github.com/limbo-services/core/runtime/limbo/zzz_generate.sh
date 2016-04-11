set -e

PROTOC_INCL="$(which protoc | sed 's|bin/protoc|include|')"

rm -f *.pb.go

protoc \
  -I $PROTOC_INCL \
  -I $GOPATH/src \
  $PWD/*.proto \
  --gogo_out=Mgoogle/protobuf/descriptor.proto=github.com/limbo-services/protobuf/protoc-gen-gogo/descriptor:$GOPATH/src/
