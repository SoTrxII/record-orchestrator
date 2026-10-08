package services

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"record-orchestrator/pkg/memory"
	pando "record-orchestrator/pkg/pandora"
	pb "record-orchestrator/proto"
	test_utils "record-orchestrator/test-utils"
	"sync"
	"testing"
	"time"
)

// A state store backed by a map, so tests exercise the real allocation logic
// rather than a canned answer
type fakeStore struct {
	mu     sync.Mutex
	states map[string]memory.State
}

func newFakeStore() *fakeStore {
	return &fakeStore{states: map[string]memory.State{}}
}

func (f *fakeStore) Save(key string, value memory.State) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states[key] = value
	return nil
}

func (f *fakeStore) Get(key string) (*memory.State, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	state, ok := f.states[key]
	if !ok {
		return nil, nil
	}
	return &state, nil
}

func (f *fakeStore) Delete(key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.states, key)
	return nil
}

func (f *fakeStore) sessions() map[string]memory.Session {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.states["recorder-state"].Sessions
}

func TestRecorder_StartOnlyPandora(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	mem := test_utils.MockStateStore{}
	recorder := NewRecorder(&pandora, &r20Rec, &mem, nil)
	pandora.On("Start", "", "1").Return(nil)
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
	store := newFakeStore()
	recorder := NewRecorder(&pandora, &r20Rec, store, nil)
	var pandoraStarted int64
	pandora.On("Start", "", "1").Run(func(mock.Arguments) { pandoraStarted = time.Now().UnixMilli() }).Return(nil)
	// The jukebox is aligned on Discord's t=0, known once Pandora replied
	r20Rec.On("Start", "2", mock.MatchedBy(func(alignTo int64) bool {
		return alignTo >= pandoraStarted && alignTo <= time.Now().UnixMilli()
	})).Return(nil)
	ret, err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "1", Roll20GameId: "2"})
	assert.Equal(t, &pb.StartRecordReply{Discord: true, Roll20: true}, ret)
	pandora.AssertExpectations(t)
	r20Rec.AssertExpectations(t)
	assert.Equal(t, "2", store.sessions()["1"].R20Id)
	if err != nil {
		t.Error(err)
	}
}

// A failing roll20 sync must not prevent the recording state from being saved :
// Pandora is recording, so the orchestrator has to remember it
func TestRecorder_StartSavesStateWhenRoll20Fails(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	store := newFakeStore()
	recorder := NewRecorder(&pandora, &r20Rec, store, nil)
	pandora.On("Start", "", "1").Return(nil)
	r20Rec.On("Start", "2", mock.AnythingOfType("int64")).Return(errors.New("roll20 is down"))

	ret, err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "1", Roll20GameId: "2"})
	assert.NoError(t, err)
	// Discord kept recording, roll20 did not
	assert.Equal(t, &pb.StartRecordReply{Discord: true, Roll20: false}, ret)
	assert.Equal(t, memory.Session{VcId: "1", R20Id: ""}, store.sessions()["1"])
	pandora.AssertExpectations(t)
	r20Rec.AssertExpectations(t)
}

// Following the case above : the caller still knows about the roll20 id, and
// that must not stop it from ending the Discord recording
func TestRecorder_StopWhenRoll20NeverStarted(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	store := newFakeStore()
	_ = store.Save("recorder-state", memory.State{
		Sessions: map[string]memory.Session{"1": {VcId: "1"}},
	})
	recorder := NewRecorder(&pandora, &r20Rec, store, nil)
	pandora.On("Stop", "", "1").Return(pando.Recording{
		Ids:          []string{"rec-1"},
		Participants: []string{"gm", "player"},
	}, nil)

	ret, err := recorder.Stop(&pb.StopRecordRequest{VoiceChannelId: "1", Roll20GameId: "2"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"rec-1"}, ret.DiscordKeys)
	assert.Equal(t, []string{"gm", "player"}, ret.ParticipantIds)
	assert.Empty(t, ret.Roll20Key)
	// We never started it, so we must not try to stop it
	r20Rec.AssertNotCalled(t, "Stop", mock.Anything)
	pandora.AssertExpectations(t)
	assert.Empty(t, store.sessions())
}

// A stop aimed at another voice channel must still be rejected
func TestRecorder_StopWrongVoiceChannel(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	store := newFakeStore()
	_ = store.Save("recorder-state", memory.State{
		Sessions: map[string]memory.Session{"1": {VcId: "1"}},
	})
	recorder := NewRecorder(&pandora, &r20Rec, store, nil)

	_, err := recorder.Stop(&pb.StopRecordRequest{VoiceChannelId: "42"})
	assert.Error(t, err)
	pandora.AssertNotCalled(t, "Stop", mock.Anything, mock.Anything)
}
