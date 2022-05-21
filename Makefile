proto:
	protoc \
		--proto_path=internal/pb \
		--go_out=internal/pb \
		internal/pb/*.proto
