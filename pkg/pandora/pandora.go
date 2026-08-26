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
	// Replies are delivered on two separate channels so an ack for a start
	// request can never be picked up by a pending stop request, and vice versa.
	// Both are buffered so a reply landing before the caller parks on the
	// channel isn't lost.
	// Note this still assumes a single request of each kind in flight at a
	// time : routing replies to concurrent sessions needs a correlation id.
	startReplies chan PandoraReply
	stopReplies  chan PandoraReply
	opt          *PandoraOpt
}

func NewPandora(pubClient utils.Publisher, subServer utils.Subscriber, component string, opt PandoraOpt) (*Pandora, error) {
	if opt.WaitTimeout == 0 {
		opt.WaitTimeout = time.Second * 30
	}
	p := &Pandora{
		pubClient:    pubClient,
		subServer:    subServer,
		component:    component,
		startReplies: make(chan PandoraReply, 1),
		stopReplies:  make(chan PandoraReply, 1),
		opt:          &opt,
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
	drainReplies(p.startReplies)
	err := p.pubClient.PublishEvent(context.Background(), p.component, P_Start, StartPandoraRequest{
		VoiceChannelId: vcId,
	})
	if err != nil {
		return err
	}
	select {
	case <-time.After(p.opt.WaitTimeout):
		err = fmt.Errorf("[Pandora] :: Timeout during initialization, could not start recording")
	case reply := <-p.startReplies:
		if reply.Error != nil {
			err = fmt.Errorf("[Pandora] :: error during initialization, could not start recording : %w", reply.Error)
		}
	}
	return err
}

func (p *Pandora) Stop(vcId string) ([]string, error) {
	drainReplies(p.stopReplies)
	err := p.pubClient.PublishEvent(context.Background(), p.component, P_End, StopPandoraRequest{
		VoiceChannelId: vcId,
	})
	if err != nil {
		return []string{}, err
	}

	var ids []string
	select {
	case <-time.After(p.opt.WaitTimeout):
		err = fmt.Errorf("[Pandora] :: Timeout, could not end recording")
	case reply := <-p.stopReplies:
		if reply.Error != nil {
			err = fmt.Errorf("[Pandora] :: could not end recording : %w", reply.Error)
		} else {
			ids = reply.Stopped.Ids
		}
	}
	return ids, err
}

// drainReplies discards a reply left behind by a previous request that timed
// out, so it can't be mistaken for the reply to the request we're about to send
func drainReplies(replies chan PandoraReply) {
	select {
	case stale := <-replies:
		slog.Warn(fmt.Sprintf("[Pandora] :: Discarding a stale reply %+v", stale))
	default:
	}
}

// deliver hands a reply over to the waiting caller. It never blocks : if no
// request is waiting anymore (it timed out), the reply is dropped rather than
// pinning the Dapr topic handler goroutine forever
func deliver(replies chan PandoraReply, reply PandoraReply, topic string) {
	select {
	case replies <- reply:
	default:
		slog.Warn(fmt.Sprintf("[Pandora] :: Dropping reply on topic %q, no request is waiting for it", topic))
	}
}

func (p *Pandora) onStoppedReply(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	reply := StopPandoraReply{}
	err = json.Unmarshal(e.RawData, &reply)
	if err != nil {
		err = fmt.Errorf("[Pandora] :: Received wrong response type from pandora %+v, %w", reply, err)
		slog.Error(err.Error())
	}
	deliver(p.stopReplies, PandoraReply{
		Started: nil,
		Stopped: &reply,
		Error:   err,
	}, S_Ended)
	return false, err
}

func (p *Pandora) onStartedReply(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	reply := StartPandoraReply{}
	err = json.Unmarshal(e.RawData, &reply)
	if err != nil {
		err = fmt.Errorf("[Pandora] :: Received wrong response type from pandora %+v, %w", reply, err)
		slog.Error(err.Error())
	}
	deliver(p.startReplies, PandoraReply{
		Started: &reply,
		Stopped: nil,
		Error:   err,
	}, S_Started)
	return false, err
}
