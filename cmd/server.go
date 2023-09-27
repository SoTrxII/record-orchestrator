package main

import (
	"context"
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
	DEFAULT_PORT      = 55555
	DEFAULT_DAPR_PORT = 50001
	// Dapr services app ids
	// TODO :: Move these to env vars
	DEFAULT_PUBSUB_ID      = "pubsub"
	DEFAULT_R20_ID         = "r20-audio-bouncer"
	DEFAULT_STATE_STORE_ID = "statestore"
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
	reply, err := s.service.Start(req)
	if err != nil {
		slog.Error(fmt.Sprintf("[Server] :: Error starting a new record with params %+v, %s", req, err.Error()))
	}
	return reply, err
}

func (s *server) Stop(ctx context.Context, req *pb.StopRecordRequest) (*pb.StopRecordReply, error) {
	if req.VoiceChannelId == "" {
		return nil, fmt.Errorf("voice channel id is required")
	}

	slog.Info(fmt.Sprintf("[Server] :: Starting a new record with params %+v", req))
	reply, err := s.service.Stop(req)
	if err != nil {
		slog.Error(fmt.Sprintf("[Server] :: Error starting a new record with params %+v, %s", req, err.Error()))
	}
	return reply, err
}

func main() {
	pEnv := parseEnv()
	slog.Info("[Main] :: Dapr port is " + strconv.Itoa(pEnv.daprGrpcPort))
	slog.Info(fmt.Sprintf("[Main] :: Parsed env %+v", pEnv))

	// Strat the gRPC Server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", pEnv.serverPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	daprServer := daprd.NewServiceWithGrpcServer(lis, s)
	recorder, err := DI(daprServer, pEnv.daprGrpcPort)
	if err != nil {
		panic(fmt.Errorf("failed to initialize event controller: %w", err))
	}
	pb.RegisterRecordServiceServer(s, &server{service: recorder})

	slog.Info(fmt.Sprintf("[Main] :: Starting gRPC server at %v", lis.Addr()))
	if err := daprServer.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}

}

type env struct {
	// Port to connect to Dapr sidecar
	daprGrpcPort int
	// Port the app is listening on
	serverPort int
	// Dapr components ids
	daprCpnPandora string
	daprCpnR20     string
	daprCpnState   string
}

func parseEnv() *env {
	pEnv := env{
		serverPort:     DEFAULT_PORT,
		daprGrpcPort:   DEFAULT_DAPR_PORT,
		daprCpnPandora: DEFAULT_PUBSUB_ID,
		daprCpnR20:     DEFAULT_R20_ID,
		daprCpnState:   DEFAULT_STATE_STORE_ID,
	}
	if envPort, err := strconv.ParseInt(os.Getenv("DAPR_GRPC_PORT"), 10, 32); err == nil && envPort != 0 {
		pEnv.daprGrpcPort = int(envPort)
	}
	if envPort, err := strconv.ParseInt(os.Getenv("SERVER_PORT"), 10, 32); err == nil && envPort != 0 {
		pEnv.serverPort = int(envPort)
	}
	if id, isDefined := os.LookupEnv("PUBSUB_NAME"); isDefined && id != "" {
		pEnv.daprCpnPandora = id
	}
	if id, isDefined := os.LookupEnv("ROLL20_NAME"); isDefined && id != "" {
		pEnv.daprCpnR20 = id
	}
	if id, isDefined := os.LookupEnv("STORE_NAME"); isDefined && id != "" {
		pEnv.daprCpnState = id
	}

	return &pEnv
}

func DI(subServer common.Service, daprPort int) (*services.Recorder, error) {
	// Dapr client, at the heart of everything
	daprClient, err := makeDaprClient(daprPort, 16)
	if err != nil {
		return nil, err
	}

	// State store
	store := memory.NewMemory[memory.State](daprClient, DEFAULT_STATE_STORE_ID)
	// Recorders themselves
	pandora, err := pando.NewPandora(daprClient, subServer, DEFAULT_PUBSUB_ID, pando.PandoraOpt{})
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
