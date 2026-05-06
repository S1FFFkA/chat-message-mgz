.PHONY: proto

proto:
	protoc -I grpc \
		--go_out=. --go_opt=module=github.com/S1FFFkA/chat-message-mgz \
		--go-grpc_out=. --go-grpc_opt=module=github.com/S1FFFkA/chat-message-mgz \
		grpc/service.proto
