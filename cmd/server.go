package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/dapr/go-sdk/client"
	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"log/slog"
	"net"
	"os"
	"record-orchestrator/pkg/memory"
	pando "record-orchestrator/pkg/pandora"
	roll20_sync "record-orchestrator/pkg/roll20-sync"
	pb "record-orchestrator/proto"
	"record-orchestrator/services"
	"strconv"
)

const (
	DEFAULT_DAPR_PORT = 50001
	// Dapr services app ids
	// TODO :: Move these to env vars
	DEFAULT_PANDORA_ID     = "pandora"
	DEFAULT_R20_ID         = "r20-audio-bouncer"
	DEFAULT_STATE_STORE_ID = "state-store"
)

var (
	port = flag.Int("port", 55555, "The server port")
)

type server struct {
	pb.UnimplementedRecordServiceServer
	service *services.Recorder
}

func (s *server) Start(ctx context.Context, req *pb.StartRecordRequest) (*pb.StartRecordReply, error) {
	if req.VoiceChannelId == "" {
		return nil, fmt.Errorf("voice channel id is required")
	}

	slog.Info(fmt.Sprintf("[Server] :: Starting a new record with params %+v", req))
	return s.service.Start(req)
}

func (s *server) Stop(ctx context.Context, req *pb.StopRecordRequest) (*pb.StopRecordReply, error) {
	if req.VoiceChannelId == "" {
		return nil, fmt.Errorf("voice channel id is required")
	}

	slog.Info(fmt.Sprintf("[Server] :: Starting a new record with params %+v", req))
	return s.service.Stop(req)
}

func main() {
	daprPort := DEFAULT_DAPR_PORT
	if envPort, err := strconv.ParseInt(os.Getenv("DAPR_GRPC_PORT"), 10, 32); err == nil && envPort != 0 {
		daprPort = int(envPort)
	}
	slog.Info("[Main] :: Dapr port is " + strconv.Itoa(daprPort))

	// Strat the gRPC Server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	daprServer := daprd.NewServiceWithGrpcServer(lis, s)
	recorder, err := DI(daprServer, daprPort)
	if err != nil {
		panic(fmt.Errorf("failed to initialize event controller: %w", err))
	}
	pb.RegisterRecordServiceServer(s, &server{service: recorder})

	slog.Info(fmt.Sprintf("[Main] :: Starting gRPC server at %v", lis.Addr()))
	if err := daprServer.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}

}

func DI(subServer common.Service, daprPort int) (*services.Recorder, error) {
	// Dapr client, at the heart of everything
	daprClient, err := makeDaprClient(daprPort, 16)
	if err != nil {
		return nil, err
	}

	// State store
	store := memory.NewMemory[services.State](daprClient, DEFAULT_STATE_STORE_ID)
	// Recorders themselves
	pandora, err := pando.NewPandora(daprClient, subServer, DEFAULT_PANDORA_ID, pando.PandoraOpt{})
	if err != nil {
		return nil, err
	}
	r20 := roll20_sync.NewRoll20Sync(daprClient, DEFAULT_R20_ID)
	return services.NewRecorder(pandora, r20, store), nil
}

func makeDaprClient(port, maxRequestSizeMB int) (client.Client, error) {
	var opts []grpc.CallOption
	opts = append(opts, grpc.MaxCallRecvMsgSize(maxRequestSizeMB*1024*1024))
	conn, err := grpc.Dial(net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port)),
		grpc.WithDefaultCallOptions(opts...), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return client.NewClientWithConnection(conn), nil
}
