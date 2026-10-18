package roll20_sync

import (
	"context"
	"encoding/json"
	"fmt"
	"record-orchestrator/internal/utils"
)

type Roll20Sync struct {
	client    utils.Invoker
	component string
}

type payload struct {
	Id      string `json:"id"`
	AlignTo int64  `json:"alignTo,omitempty"`
}

func NewRoll20Sync(client utils.Invoker, component string) *Roll20Sync {
	return &Roll20Sync{
		client:    client,
		component: component,
	}
}

func (r *Roll20Sync) Start(r20Id string, alignTo int64) error {
	content, err := json.Marshal(payload{
		Id:      r20Id,
		AlignTo: alignTo,
	})
	if err != nil {
		return err
	}
	_, err = r.client.InvokeMethodWithContent(context.Background(), r.component, "v1/jukeboxsyncer/start", "POST", &utils.DataContent{
		Data:        content,
		ContentType: "application/json",
	})
	return err
}

func (r *Roll20Sync) Stop(r20Id string) (string, error) {

	content, err := json.Marshal(payload{
		Id: r20Id,
	})
	if err != nil {
		return "", err
	}
	res, err := r.client.InvokeMethodWithContent(context.Background(), r.component, "v1/jukeboxsyncer/stop", "POST", &utils.DataContent{
		Data:        content,
		ContentType: "application/json",
	})
	if err != nil {
		return "", err
	}
	// roll20-audio-sync names the file: "<id>.ogg" for the jukebox alone,
	// "<id>.mkv" when the GM recorded the tab's picture too
	var reply struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(res, &reply); err != nil || reply.Key == "" {
		return "", fmt.Errorf("roll20-audio-sync gave no key for %s : %q", r20Id, res)
	}
	return reply.Key, nil
}
