// Pandora is a Discord Recording bot.
// It works by using the request/reply pattern
package pandora

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dapr/go-sdk/service/common"
	"log/slog"
	"record-orchestrator/internal/utils"
	"time"
)

type PandoraOpt struct {
	WaitTimeout time.Duration
}
type Pandora struct {
	subServer utils.Subscriber
	pubClient utils.Publisher
	component string
	replies   chan PandoraReply
	opt       *PandoraOpt
}

func NewPandora(pubClient utils.Publisher, subServer utils.Subscriber, component string, opt PandoraOpt) (*Pandora, error) {
	if opt.WaitTimeout == 0 {
		opt.WaitTimeout = time.Second * 30
	}
	p := &Pandora{
		pubClient: pubClient,
		subServer: subServer,
		component: component,
		replies:   make(chan PandoraReply),
		opt:       &opt,
	}

	err := p.subscribeTo(subServer)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Subscribe to event emitted by Pandora
func (p *Pandora) subscribeTo(subServer utils.Subscriber) error {
	// Subscribe to the ACK after a recording request
	err := subServer.AddTopicEventHandler(&common.Subscription{
		PubsubName: p.component,
		Topic:      S_Started,
	}, p.onStartedReply)
	if err != nil {
		return err
	}

	// Subscribe to the reply after a stop record request
	err = subServer.AddTopicEventHandler(&common.Subscription{
		PubsubName: p.component,
		Topic:      S_Ended,
	}, p.onStoppedReply)

	if err != nil {
		return err
	}

	return nil
}

// Start a new recording session
func (p *Pandora) Start(vcId string) error {
	// Pandora can only record a single voice channel at a time.
	// In an effort to be completely stateless, we will let Pandora
	// check the recording state
	err := p.pubClient.PublishEvent(context.Background(), p.component, string(P_Start), StartPandoraRequest{
		VoiceChannelId: vcId,
	})
	if err != nil {
		return err
	}
	select {
	case <-time.After(p.opt.WaitTimeout):
		err = fmt.Errorf("[Pandora] :: Timeout during initialization, could not start recording")
	case reply := <-p.replies:
		if reply.Error != nil {
			err = fmt.Errorf("[Pandora] :: error during initialization, could not start recording : %w", reply.Error)
		}
	}
	return err
}

func (p *Pandora) Stop(vcId string) ([]string, error) {
	err := p.pubClient.PublishEvent(context.Background(), p.component, P_End, StartPandoraRequest{
		VoiceChannelId: vcId,
	})
	if err != nil {
		return []string{}, err
	}

	var ids []string
	select {
	case <-time.After(p.opt.WaitTimeout):
		err = fmt.Errorf("[Pandora] :: Timeout, could not end recording")
	case reply := <-p.replies:
		if reply.Error != nil {
			err = fmt.Errorf("[Pandora] :: could not end recording : %w", reply.Error)
		} else {
			ids = reply.Stopped.Ids
		}
	}
	return ids, err
}

func (p *Pandora) onStoppedReply(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	reply := StopPandoraReply{}
	err = json.Unmarshal(e.RawData, &reply)
	if err != nil {
		err = fmt.Errorf("[Pandora] :: Received wrong response type from pandora %+v, %w", reply, err)
		slog.Error(err.Error())
	}
	p.replies <- PandoraReply{
		Started: nil,
		Stopped: &reply,
		Error:   err,
	}
	return false, err
}

func (p *Pandora) onStartedReply(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	reply := StartPandoraReply{}
	err = json.Unmarshal(e.RawData, &reply)
	if err != nil {
		err = fmt.Errorf("[Pandora] :: Received wrong response type from pandora %+v, %w", reply, err)
		slog.Error(err.Error())
	}
	p.replies <- PandoraReply{
		Started: &reply,
		Stopped: nil,
		Error:   err,
	}
	return false, err
}
