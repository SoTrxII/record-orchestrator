package services

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"record-orchestrator/pkg/memory"
	pb "record-orchestrator/proto"
	test_utils "record-orchestrator/test-utils"
	"sync"
	"testing"
)

// Two sessions recorded at once must land on two different bots : a single
// Discord token can only hold one voice connection per guild
func TestRecorderPool_SpreadsSessionsOverInstances(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	store := newFakeStore()
	recorder := NewRecorder(&pandora, &r20Rec, store, []string{"pandora-0", "pandora-1"})
	pandora.On("Start", mock.Anything, mock.Anything).Return(nil)

	_, err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "channel-A"})
	assert.NoError(t, err)
	_, err = recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "channel-B"})
	assert.NoError(t, err)

	sessions := store.sessions()
	assert.Len(t, sessions, 2)
	assert.NotEqual(t, sessions["channel-A"].InstanceId, sessions["channel-B"].InstanceId,
		"two sessions were handed to the same bot")
	assert.Subset(t, []string{"pandora-0", "pandora-1"},
		[]string{sessions["channel-A"].InstanceId, sessions["channel-B"].InstanceId})
}

// Once every bot of the pool is busy, a new session has to be refused rather
// than silently stealing one
func TestRecorderPool_RefusesWhenExhausted(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	store := newFakeStore()
	recorder := NewRecorder(&pandora, &r20Rec, store, []string{"pandora-0"})
	pandora.On("Start", mock.Anything, mock.Anything).Return(nil)

	_, err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "channel-A"})
	assert.NoError(t, err)

	_, err = recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "channel-B"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "busy")
	assert.Len(t, store.sessions(), 1)
}

// Recording the same voice channel twice makes no sense, whatever the pool size
func TestRecorderPool_RefusesTheSameChannelTwice(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	store := newFakeStore()
	recorder := NewRecorder(&pandora, &r20Rec, store, []string{"pandora-0", "pandora-1"})
	pandora.On("Start", mock.Anything, mock.Anything).Return(nil)

	_, err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "channel-A"})
	assert.NoError(t, err)
	_, err = recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "channel-A"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already recording")
	assert.Len(t, store.sessions(), 1)
}

// A bot whose recording failed to start must go back into the pool, otherwise
// capacity leaks away one failure at a time
func TestRecorderPool_ReleasesInstanceOnFailedStart(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	store := newFakeStore()
	recorder := NewRecorder(&pandora, &r20Rec, store, []string{"pandora-0"})
	pandora.On("Start", "pandora-0", "channel-A").Return(errors.New("pandora is down")).Once()

	_, err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "channel-A"})
	assert.Error(t, err)
	assert.Empty(t, store.sessions(), "the failed session was left holding a bot")

	// The only bot of the pool is free again
	pandora.On("Start", "pandora-0", "channel-B").Return(nil).Once()
	_, err = recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "channel-B"})
	assert.NoError(t, err)
}

// Stopping a session must go to the bot that is actually recording it
func TestRecorderPool_StopsOnTheRightInstance(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	store := newFakeStore()
	recorder := NewRecorder(&pandora, &r20Rec, store, []string{"pandora-0", "pandora-1"})
	pandora.On("Start", mock.Anything, mock.Anything).Return(nil)

	_, err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "channel-A"})
	assert.NoError(t, err)
	_, err = recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "channel-B"})
	assert.NoError(t, err)

	ownerOfB := store.sessions()["channel-B"].InstanceId
	pandora.On("Stop", ownerOfB, "channel-B").Return([]string{"rec-B"}, nil).Once()

	ret, err := recorder.Stop(&pb.StopRecordRequest{VoiceChannelId: "channel-B"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"rec-B"}, ret.DiscordKeys)
	pandora.AssertExpectations(t)

	// The other session is untouched, and the bot that recorded B is free again
	assert.Len(t, store.sessions(), 1)
	assert.Contains(t, store.sessions(), "channel-A")
}

// Stopping frees the bot for the next session
func TestRecorderPool_FreesInstanceOnStop(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	store := newFakeStore()
	recorder := NewRecorder(&pandora, &r20Rec, store, []string{"pandora-0"})
	pandora.On("Start", mock.Anything, mock.Anything).Return(nil)
	pandora.On("Stop", mock.Anything, mock.Anything).Return([]string{"rec-A"}, nil)

	_, err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "channel-A"})
	assert.NoError(t, err)
	_, err = recorder.Stop(&pb.StopRecordRequest{VoiceChannelId: "channel-A"})
	assert.NoError(t, err)

	_, err = recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "channel-B"})
	assert.NoError(t, err, "the bot was not handed back to the pool")
}

// Two sessions started at the same time must never be given the same bot
func TestRecorderPool_ConcurrentStartsNeverShareAnInstance(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	store := newFakeStore()
	pool := []string{"pandora-0", "pandora-1", "pandora-2", "pandora-3"}
	recorder := NewRecorder(&pandora, &r20Rec, store, pool)
	pandora.On("Start", mock.Anything, mock.Anything).Return(nil)

	channels := []string{"channel-A", "channel-B", "channel-C", "channel-D"}
	var wg sync.WaitGroup
	for _, channel := range channels {
		wg.Add(1)
		go func(vcId string) {
			defer wg.Done()
			_, err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: vcId})
			assert.NoError(t, err)
		}(channel)
	}
	wg.Wait()

	sessions := store.sessions()
	assert.Len(t, sessions, len(pool))
	taken := map[string]string{}
	for vcId, session := range sessions {
		if previous, isTaken := taken[session.InstanceId]; isTaken {
			t.Fatalf("bot %q was handed to both %q and %q", session.InstanceId, previous, vcId)
		}
		taken[session.InstanceId] = vcId
	}
}

// Without a pool configured, Pandora is addressed on the plain topics, exactly
// as it was before pools existed
func TestRecorderPool_UnnamedInstanceByDefault(t *testing.T) {
	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	store := newFakeStore()
	recorder := NewRecorder(&pandora, &r20Rec, store, nil)
	pandora.On("Start", "", "channel-A").Return(nil).Once()

	_, err := recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "channel-A"})
	assert.NoError(t, err)
	assert.Equal(t, "", store.sessions()["channel-A"].InstanceId)

	// And there is only one of it : one bot, one recording
	_, err = recorder.Start(&pb.StartRecordRequest{VoiceChannelId: "channel-B"})
	assert.Error(t, err)
	pandora.AssertExpectations(t)
}

// The recording state must survive a restart : a fresh orchestrator reading an
// existing state has to see which bots are still busy
func TestRecorderPool_RecoversBusyInstancesFromState(t *testing.T) {
	store := newFakeStore()
	_ = store.Save("recorder-state", memory.State{
		Sessions: map[string]memory.Session{
			"channel-A": {VcId: "channel-A", InstanceId: "pandora-0"},
		},
	})

	pandora := test_utils.MockDiscordRecorder{}
	r20Rec := test_utils.MockR20Recorder{}
	revived := NewRecorder(&pandora, &r20Rec, store, []string{"pandora-0", "pandora-1"})
	pandora.On("Start", "pandora-1", "channel-B").Return(nil).Once()

	_, err := revived.Start(&pb.StartRecordRequest{VoiceChannelId: "channel-B"})
	assert.NoError(t, err)
	pandora.AssertExpectations(t)
}
