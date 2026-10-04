package services

import (
	"fmt"
	"log/slog"
	"record-orchestrator/pkg/memory"
	"record-orchestrator/pkg/pandora"
	roll20_sync "record-orchestrator/pkg/roll20-sync"
	pb "record-orchestrator/proto"
	"sync"
)

type Recorder struct {
	pandora    pandora.DiscordRecorder
	roll20Sync roll20_sync.R20Recorder
	memory     memory.StateStore
	stateKey   string
	// Instances of the Pandora pool a recording can be handed to. A single
	// Discord bot holds one voice connection per guild, so recording several
	// sessions at once means spreading them over several bots
	instances []string
	// Guards the allocation of an instance to a session. The orchestrator runs
	// as a single replica, so an in-process lock is enough to stop two
	// sessions from landing on the same instance
	mu sync.Mutex
}

// NewRecorder builds the recorder over a pool of Pandora instances.
// An empty pool means a single unnamed Pandora, which is how it behaved before
// pools existed
func NewRecorder(pandora pandora.DiscordRecorder, r20 roll20_sync.R20Recorder, memory memory.StateStore, instances []string) *Recorder {
	if len(instances) == 0 {
		instances = []string{""}
	}
	return &Recorder{
		pandora:    pandora,
		roll20Sync: r20,
		memory:     memory,
		stateKey:   "recorder-state",
		instances:  instances,
	}
}

func (r *Recorder) Start(payload *pb.StartRecordRequest) (*pb.StartRecordReply, error) {
	// Input sanity check
	if payload.VoiceChannelId == "" {
		return nil, fmt.Errorf("[Recorder] :: voice channel id is required but got %+v", payload)
	}

	// Claim an instance for this session before doing anything slow, so a
	// second session starting at the same time can't be handed the same one
	instanceId, err := r.reserve(payload.VoiceChannelId)
	if err != nil {
		return nil, err
	}

	if err := r.pandora.Start(instanceId, payload.VoiceChannelId); err != nil {
		// Nothing is recording, hand the instance back to the pool
		if releaseErr := r.release(payload.VoiceChannelId); releaseErr != nil {
			slog.Error(fmt.Sprintf("[Recorder] :: Failed to release instance %q after a failed start : %s", instanceId, releaseErr.Error()))
		}
		return nil, err
	}
	reply := pb.StartRecordReply{
		Discord: true,
		Roll20:  false,
	}

	// Roll20 is optional so we don't return an error if it's not provided.
	// If it fails we keep the Discord recording going : the session is already
	// committed to the state, so it can still be stopped later on
	if payload.GetRoll20GameId() != "" {
		if err := r.roll20Sync.Start(payload.GetRoll20GameId()); err != nil {
			slog.Warn(fmt.Sprintf("[Recorder] :: Failed to start roll20 sync, continuing without it. Reason : %s", err.Error()))
		} else {
			reply.Roll20 = true
			if err := r.attachRoll20(payload.VoiceChannelId, payload.GetRoll20GameId()); err != nil {
				return nil, err
			}
		}
	}

	slog.Info(fmt.Sprintf("[Recorder] :: Voice channel %s is being recorded by instance %q", payload.VoiceChannelId, instanceId))
	return &reply, nil
}

func (r *Recorder) Stop(payload *pb.StopRecordRequest) (*pb.StopRecordReply, error) {
	if payload.VoiceChannelId == "" {
		return nil, fmt.Errorf("[Recorder] :: voice channel id is required but got %+v", payload)
	}

	session, err := r.session(payload.VoiceChannelId)
	if err != nil {
		return nil, err
	}
	// The voice channel identifies the recording. The roll20 id is only checked
	// when we actually managed to start a roll20 sync : if it failed at start
	// time the caller still sends its id, and that must not block the stop
	if session.R20Id != "" && session.R20Id != payload.GetRoll20GameId() {
		return nil, fmt.Errorf("[Recorder] :: Wrong recordings parameters, expected %+v, got %+v", session, payload)
	}

	recording, err := r.pandora.Stop(session.InstanceId, payload.VoiceChannelId)
	if err != nil {
		return nil, err
	}

	r20Key := ""
	if session.R20Id != "" {
		r20Key, err = r.roll20Sync.Stop(session.R20Id)
		if err != nil {
			slog.Warn(fmt.Sprintf("[Recorder] :: Failed to stop roll20 sync, continuing without it. Reason : %s", err.Error()))
		}
	}

	if err := r.release(payload.VoiceChannelId); err != nil {
		return nil, err
	}

	// TODO :: Calculate offset for synchronisation
	return &pb.StopRecordReply{
		DiscordKeys:    recording.Ids,
		Roll20Key:      r20Key,
		ParticipantIds: recording.Participants,
	}, nil
}

// reserve claims a free instance of the pool for this voice channel and
// commits the session to the state, so a concurrent start can see it
func (r *Recorder) reserve(vcId string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return "", err
	}
	if session, isRecording := state.Sessions[vcId]; isRecording {
		return "", fmt.Errorf("[Recorder] :: already recording voice channel %s on instance %q", vcId, session.InstanceId)
	}
	instanceId, err := r.freeInstance(state)
	if err != nil {
		return "", err
	}
	state.Sessions[vcId] = memory.Session{VcId: vcId, InstanceId: instanceId}
	if err := r.memory.Save(r.stateKey, *state); err != nil {
		return "", err
	}
	return instanceId, nil
}

// release hands the instance recording this voice channel back to the pool
func (r *Recorder) release(vcId string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return err
	}
	delete(state.Sessions, vcId)
	// Nothing is being recorded anymore, leave the store clean
	if len(state.Sessions) == 0 {
		return r.memory.Delete(r.stateKey)
	}
	return r.memory.Save(r.stateKey, *state)
}

// attachRoll20 records that this session also drives a roll20 sync
func (r *Recorder) attachRoll20(vcId, r20Id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return err
	}
	session, isRecording := state.Sessions[vcId]
	if !isRecording {
		return fmt.Errorf("[Recorder] :: not recording voice channel %s anymore", vcId)
	}
	session.R20Id = r20Id
	state.Sessions[vcId] = session
	return r.memory.Save(r.stateKey, *state)
}

// session returns the recording in progress on this voice channel
func (r *Recorder) session(vcId string) (memory.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return memory.Session{}, err
	}
	session, isRecording := state.Sessions[vcId]
	if !isRecording {
		return memory.Session{}, fmt.Errorf("[Recorder] :: not recording voice channel %s", vcId)
	}
	return session, nil
}

// freeInstance picks an instance of the pool that isn't recording anything.
// Callers must hold the lock
func (r *Recorder) freeInstance(state *memory.State) (string, error) {
	busy := make(map[string]bool, len(state.Sessions))
	for _, session := range state.Sessions {
		busy[session.InstanceId] = true
	}
	for _, instanceId := range r.instances {
		if !busy[instanceId] {
			return instanceId, nil
		}
	}
	return "", fmt.Errorf("[Recorder] :: every recorder is busy, %d recording(s) in progress", len(state.Sessions))
}

// loadState reads the state, treating a blank store as no recording at all.
// Callers must hold the lock
func (r *Recorder) loadState() (*memory.State, error) {
	state, err := r.memory.Get(r.stateKey)
	if err != nil {
		return nil, err
	}
	if state == nil {
		state = &memory.State{}
	}
	if state.Sessions == nil {
		state.Sessions = map[string]memory.Session{}
	}
	return state, nil
}
