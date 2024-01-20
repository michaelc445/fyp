rm -rf proto
git clone https://github.com/michaelc445/proto

protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/messages.proto

