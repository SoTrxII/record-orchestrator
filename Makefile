.PHONY: proto dapr_run dapr

proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/recorder.proto


dapr_run:
	dapr run --app-id=record-orchestrator --app-port 55555 --dapr-grpc-port 50011 --resources-path ./dapr/components -- go run cmd/server.go

dapr:
	dapr run --app-id=record-orchestrator --app-protocol=grpc --app-port 55555 --dapr-grpc-port 50011  --resources-path ./dapr/components
