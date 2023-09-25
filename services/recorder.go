package services

import (
	"fmt"
	"github.com/google/uuid"
	"log/slog"
	"record-orchestrator/internal/utils"
	pb "record-orchestrator/proto"
)

type Recorder struct {
	pandora    utils.DiscordRecorder
	roll20Sync utils.R20Recorder
	memory     StateStore
	stateKey   string
}

func NewRecorder(pandora utils.DiscordRecorder, r20 utils.R20Recorder, memory StateStore) *Recorder {
	return &Recorder{
		pandora:    pandora,
		roll20Sync: r20,
		memory:     memory,
		stateKey:   "recorder-state",
	}
}

func (r *Recorder) Start(payload *pb.StartRecordRequest) (*pb.StartRecordReply, error) {
	// Input sanity check
	if payload.VoiceChannelId == "" {
		return nil, fmt.Errorf("[Recorder] :: voice channel id is required but got %+v", payload)
	}
	// Check if we're already recording
	state, err := r.memory.Get(r.stateKey)
	if err != nil {
		return nil, err
	}
	if state != nil {
		return nil, fmt.Errorf("[Recorder] :: already recording")
	}

	state = &State{
		VcId:  "",
		R20Id: "",
	}

	err = r.pandora.Start(payload.VoiceChannelId)
	if err != nil {
		return nil, err
	}
	reply := pb.StartRecordReply{
		Discord: true,
		Roll20:  false,
	}
	state.VcId = payload.VoiceChannelId
	//r.memory.Save(payload.VoiceChannelId, true)
	// Roll20 is optional so we don't return an error if it's not provided
	if payload.GetRoll20GameId() != "" {
		err = r.roll20Sync.Start(payload.GetRoll20GameId())
		if err != nil {
			slog.Warn(fmt.Sprintf("[Recorder] :: Failed to start roll20 sync, continuing without it. Reason : %s", err.Error()))
			return &reply, nil
		}
		reply.Roll20 = true
		state.R20Id = payload.GetRoll20GameId()
	}

	err = r.memory.Save(r.stateKey, *state)
	if err != nil {
		return nil, err
	}
	return &reply, nil
}

func (r *Recorder) Stop(payload *pb.StopRecordRequest) (*pb.StopRecordReply, error) {
	if payload.VoiceChannelId == "" {
		return nil, fmt.Errorf("[Recorder] :: voice channel id is required but got %+v", payload)
	}
	state, err := r.memory.Get(r.stateKey)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, fmt.Errorf("[Recorder] :: not recording")
	}
	if state.VcId != payload.VoiceChannelId || state.R20Id != payload.GetRoll20GameId() {
		return nil, fmt.Errorf("[Recorder] :: Wrong recordings parameters, expected %+v, got %+v", state, payload)
	}

	ids, err := r.pandora.Stop(payload.VoiceChannelId)
	if err != nil {
		return nil, err
	}

	r20Key := ""
	if payload.GetRoll20GameId() != "" {
		r20Key, err = r.roll20Sync.Stop(payload.GetRoll20GameId())
		if err != nil {
			slog.Warn(fmt.Sprintf("[Recorder] :: Failed to stop roll20 sync, continuing without it. Reason : %s", err.Error()))
		}
	}

	err = r.memory.Delete(r.stateKey)
	if err != nil {
		return nil, err
	}

	uuid := uuid.New().String()

	// TODO :: Enqueue record processing, return jobId to caller
	_ = uuid
	_ = r20Key
	_ = ids
	return &pb.StopRecordReply{JobId: uuid}, nil
}
