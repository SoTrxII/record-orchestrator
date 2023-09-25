package services

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	pb "record-orchestrator/proto"
	test_utils "record-orchestrator/test-utils"
	"testing"
)

func TestRecorder_StartOnlyPandora(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	mem := test_utils.MockStateStore{}
	recorder := NewRecorder(&pandora, &r20Rec, &mem)
	pandora.On("Start", "1").Return(nil)
	mem.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
	mem.EXPECT().Get(mock.Anything).Return(nil, nil)
	ret, err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "1"})
	assert.Equal(t, &pb.StartRecordReply{Discord: true, Roll20: false}, ret)
	pandora.AssertExpectations(t)
	r20Rec.AssertNotCalled(t, "Start", mock.Anything)
	if err != nil {
		t.Error(err)
	}
}

func TestRecorder_StartPandoraAndRoll20(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	mem := test_utils.MockStateStore{}
	recorder := NewRecorder(&pandora, &r20Rec, &mem)
	pandora.On("Start", "1").Return(nil)
	r20Rec.On("Start", "2").Return(nil)
	mem.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
	mem.EXPECT().Get(mock.Anything).Return(nil, nil)
	ret, err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "1", Roll20GameId: "2"})
	assert.Equal(t, &pb.StartRecordReply{Discord: true, Roll20: true}, ret)
	pandora.AssertExpectations(t)
	r20Rec.AssertExpectations(t)
	if err != nil {
		t.Error(err)
	}
}
