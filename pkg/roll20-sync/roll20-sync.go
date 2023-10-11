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
	Id string `json:"id"`
}

func NewRoll20Sync(client utils.Invoker, component string) *Roll20Sync {
	return &Roll20Sync{
		client:    client,
		component: component,
	}
}

func (r *Roll20Sync) Start(r20Id string) error {

	content, err := json.Marshal(payload{
		Id: r20Id,
	})
	if err != nil {
		return err
	}
	res, err := r.client.InvokeMethodWithContent(context.Background(), r.component, "v1/jukeboxsyncer/start", "POST", &utils.DataContent{
		Data:        content,
		ContentType: "application/json",
	})
	fmt.Printf("res: %s\n", res)
	return err
}

func (r *Roll20Sync) Stop(r20Id string) (string, error) {

	content, err := json.Marshal(payload{
		Id: r20Id,
	})
	if err != nil {
		return "", err
	}
	_, err = r.client.InvokeMethodWithContent(context.Background(), r.component, "v1/jukeboxsyncer/stop", "POST", &utils.DataContent{
		Data:        content,
		ContentType: "application/json",
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.ogg", r20Id), nil
}
