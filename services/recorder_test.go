package services

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"record-orchestrator/pkg/memory"
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

// A failing roll20 sync must not prevent the recording state from being saved :
// Pandora is recording, so the orchestrator has to remember it
func TestRecorder_StartSavesStateWhenRoll20Fails(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	mem := test_utils.MockStateStore{}
	recorder := NewRecorder(&pandora, &r20Rec, &mem)
	pandora.On("Start", "1").Return(nil)
	r20Rec.On("Start", "2").Return(errors.New("roll20 is down"))
	mem.EXPECT().Get(mock.Anything).Return(nil, nil)

	var saved memory.State
	mem.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(key string, value memory.State) { saved = value }).Return(nil)

	ret, err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "1", Roll20GameId: "2"})
	assert.NoError(t, err)
	// Discord kept recording, roll20 did not
	assert.Equal(t, &pb.StartRecordReply{Discord: true, Roll20: false}, ret)
	assert.Equal(t, memory.State{VcId: "1", R20Id: ""}, saved)
	pandora.AssertExpectations(t)
	r20Rec.AssertExpectations(t)
	mem.AssertExpectations(t)
}

// Following the case above : the caller still knows about the roll20 id, and
// that must not stop it from ending the Discord recording
func TestRecorder_StopWhenRoll20NeverStarted(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	mem := test_utils.MockStateStore{}
	recorder := NewRecorder(&pandora, &r20Rec, &mem)
	pandora.On("Stop", "1").Return([]string{"rec-1"}, nil)
	mem.EXPECT().Get(mock.Anything).Return(&memory.State{VcId: "1", R20Id: ""}, nil)
	mem.EXPECT().Delete(mock.Anything).Return(nil)

	ret, err := recorder.Stop(&pb.StopRecordRequest{VoiceChannelId: "1", Roll20GameId: "2"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"rec-1"}, ret.DiscordKeys)
	assert.Empty(t, ret.Roll20Key)
	// We never started it, so we must not try to stop it
	r20Rec.AssertNotCalled(t, "Stop", mock.Anything)
	pandora.AssertExpectations(t)
}

// A stop aimed at another voice channel must still be rejected
func TestRecorder_StopWrongVoiceChannel(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	mem := test_utils.MockStateStore{}
	recorder := NewRecorder(&pandora, &r20Rec, &mem)
	mem.EXPECT().Get(mock.Anything).Return(&memory.State{VcId: "1"}, nil)

	_, err := recorder.Stop(&pb.StopRecordRequest{VoiceChannelId: "42"})
	assert.Error(t, err)
	pandora.AssertNotCalled(t, "Stop", mock.Anything)
}
