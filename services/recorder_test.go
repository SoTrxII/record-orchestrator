package services

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"record-orchestrator/internal/utils"
	pb "record-orchestrator/proto"
	"testing"
)

func TestRecorder_StartOnlyPandora(t *testing.T) {
	pandora := MockPandora{}
	r20Rec := MockR20Recorder{}
	recorder := NewRecorder(&pandora, &r20Rec)
	pandora.On("Start", "1").Return(nil)
	ret, err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "1"})
	assert.Equal(t, &pb.StartRecordReply{Discord: true, Roll20: false}, ret)
	pandora.AssertExpectations(t)
	r20Rec.AssertNotCalled(t, "Start", mock.Anything)
	if err != nil {
		t.Error(err)
	}
}

func TestRecorder_StartPandoraAndRoll20(t *testing.T) {
	pandora := MockPandora{}
	r20Rec := MockR20Recorder{}
	recorder := NewRecorder(&pandora, &r20Rec)
	pandora.On("Start", "1").Return(nil)
	r20Rec.On("Start", "2").Return(nil)
	ret, err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "1", Roll20GameId: "2"})
	assert.Equal(t, &pb.StartRecordReply{Discord: true, Roll20: true}, ret)
	pandora.AssertExpectations(t)
	r20Rec.AssertExpectations(t)
	if err != nil {
		t.Error(err)
	}
}

type MockPandora struct {
	mock.Mock
	utils.DiscordRecorder
}

func (m *MockPandora) Start(voiceChannelId string) error {
	args := m.Called(voiceChannelId)
	return args.Error(0)
}

func (m *MockPandora) Stop(vcId string) ([]string, error) {
	args := m.Called(vcId)
	return args.Get(0).([]string), args.Error(1)
}

type MockR20Recorder struct {
	mock.Mock
	utils.R20Recorder
}

func (m *MockR20Recorder) Start(r20Id string) error {
	args := m.Called(r20Id)
	return args.Error(0)
}

func (m *MockR20Recorder) Stop(r20Id string) (string, error) {
	args := m.Called(r20Id)
	return args.String(0), args.Error(1)
}
