protoc \
  --proto_path=. \
  --go_out=. \
  --go-vtproto_out=. \
  --go_opt=paths=source_relative \
  messages.proto