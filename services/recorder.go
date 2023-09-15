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
}

func NewRecorder(pandora utils.DiscordRecorder, r20 utils.R20Recorder) *Recorder {
	return &Recorder{
		pandora:    pandora,
		roll20Sync: r20,
	}
}

func (r *Recorder) Start(payload *pb.StartRecordRequest) (*pb.StartRecordReply, error) {
	if payload.VoiceChannelId == "" {
		return nil, fmt.Errorf("[Recorder] :: voice channel id is required but got %+v", payload)
	}
	err := r.pandora.Start(payload.VoiceChannelId)
	if err != nil {
		return nil, err
	}
	reply := pb.StartRecordReply{
		Discord: true,
		Roll20:  false,
	}
	// Roll20 is optional so we don't return an error if it's not provided
	if payload.GetRoll20GameId() != "" {
		err = r.roll20Sync.Start(payload.GetRoll20GameId())
		if err != nil {
			slog.Warn("[Recorder] :: Failed to start roll20 sync, continuing without it. Reason : %s", err.Error())
			return &reply, nil
		}
		reply.Roll20 = true
	}

	return &reply, nil
}

func (r *Recorder) Stop(payload *pb.StopRecordRequest) (*pb.StopRecordReply, error) {
	if payload.VoiceChannelId == "" {
		return nil, fmt.Errorf("[Recorder] :: voice channel id is required but got %+v", payload)
	}

	ids, err := r.pandora.Stop(payload.VoiceChannelId)
	if err != nil {
		return nil, err
	}

	r20Key := ""
	if payload.GetRoll20GameId() != "" {
		r20Key, err = r.roll20Sync.Stop(payload.GetRoll20GameId())
		if err != nil {
			slog.Warn("[Recorder] :: Failed to stop roll20 sync, continuing without it. Reason : %s", err.Error())
		}
	}

	uuid := uuid.New().String()

	// TODO :: Enqueue record processing, return jobId to caller
	_ = uuid
	_ = r20Key
	_ = ids
	return &pb.StopRecordReply{JobId: uuid}, nil
}
