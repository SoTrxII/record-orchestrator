//go:build integration
// +build integration

package services

import (
	"fmt"
	"github.com/dapr/go-sdk/client"
	daprd "github.com/dapr/go-sdk/service/grpc"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"os"
	"record-orchestrator/pkg/pandora"
	roll20_sync "record-orchestrator/pkg/roll20-sync"
	pb "record-orchestrator/proto"
	"testing"
	"time"
)

const (
	SERVER_PORT    = 55555
	DAPR_PORT      = 50011
	TEST_ROLL20_ID = "2"
	TEST_CHANNEL   = "416228669095411717"
)

var (
	recorder *Recorder
)

func beforeAll() net.Listener {
	// Dapr server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", SERVER_PORT))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	subServer := daprd.NewServiceWithListener(lis)

	// Dapr client
	var opts []grpc.CallOption
	opts = append(opts, grpc.MaxCallRecvMsgSize(4*1024*1024))
	conn, err := grpc.Dial(net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", DAPR_PORT)),
		grpc.WithDefaultCallOptions(opts...), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("error creating dapr client: %v", err)
	}
	daprClient := client.NewClientWithConnection(conn)

	// Recorders themselves
	pandora, err := pandora.NewPandora(daprClient, subServer, "pubsub", pandora.PandoraOpt{})
	if err != nil {
		log.Fatalf("error creating dapr client: %v", err)
	}
	r20 := roll20_sync.NewRoll20Sync(daprClient)
	recorder = NewRecorder(pandora, r20)

	// Start the server
	go func() {
		if err := subServer.Start(); err != nil {
			log.Fatalf("error listenning: %v", err)
		}
	}()
	return lis
}

func TestRecorder_PandoraOnly(t *testing.T) {
	err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: TEST_CHANNEL})
	assert.NoError(t, err)

	// TODO :: Pandora is kinda weak against a double start, The recording state should be stored in a state
	/*err = recorder.Start(&pb.StartRecordRequest{VoiceChannelId: TEST_CHANNEL})
	assert.Error(t, err)*/

	time.Sleep(5 * time.Second)
	err = recorder.Stop(&pb.StopRecordRequest{VoiceChannelId: TEST_CHANNEL})
	assert.NoError(t, err)

	// TODO :: The double stop is correctly denied but
	// doesn't return anything
	time.Sleep(5 * time.Second)
	err = recorder.Stop(&pb.StopRecordRequest{VoiceChannelId: TEST_CHANNEL})
	assert.Error(t, err)
}

func TestRecorder_PandoraAndSyncer(t *testing.T) {
	err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: TEST_CHANNEL, Roll20GameId: TEST_ROLL20_ID})
	assert.NoError(t, err)

	// TODO :: Pandora is kinda weak against a double start
	/*err = recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "416228669095411717"})
	assert.Error(t, err)*/

	time.Sleep(5 * time.Second)
	err = recorder.Stop(&pb.StopRecordRequest{VoiceChannelId: TEST_CHANNEL, Roll20GameId: TEST_ROLL20_ID})
	assert.NoError(t, err)

	// TODO :: The double stop is correctly denied but
	// doesn't return anything
	//time.Sleep(5 * time.Second)
	// err = recorder.Stop(&pb.StopRecordRequest{VoiceChannelId: TEST_CHANNEL, Roll20GameId: TEST_ROLL20_ID})
	//assert.Error(t, err)
}

func TestMain(m *testing.M) {
	lis := beforeAll()
	defer lis.Close()
	exitCode := m.Run()
	os.Exit(exitCode)
}
