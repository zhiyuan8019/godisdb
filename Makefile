build-proto:
	@export INCLUDE_DIR=/usr/local/include:.
	@rm -rf ./proto/*.pb.go
	@protoc ./proto/*.proto -I=${INCLUDE_DIR}  --go_out=.
