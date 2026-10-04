// Pandora is a Discord Recording bot.
// It works by using the request/reply pattern
package pandora

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dapr/go-sdk/service/common"
	"github.com/google/uuid"
	"log/slog"
	"record-orchestrator/internal/utils"
	"sync"
	"time"
)

type PandoraOpt struct {
	WaitTimeout time.Duration
}
type Pandora struct {
	subServer utils.Subscriber
	pubClient utils.Publisher
	component string
	// One entry per in-flight request, keyed by the correlation id Pandora
	// echoes back. This is what allows several recording sessions to be
	// started and stopped concurrently without their replies crossing over.
	mu      sync.Mutex
	pending map[string]pendingRequest
	opt     *PandoraOpt
}

// A request waiting for its reply. The topic is kept so a reply can never be
// handed to a request of the other kind, whatever its correlation id
type pendingRequest struct {
	replyTopic string
	replies    chan PandoraReply
}

func NewPandora(pubClient utils.Publisher, subServer utils.Subscriber, component string, opt PandoraOpt) (*Pandora, error) {
	if opt.WaitTimeout == 0 {
		opt.WaitTimeout = time.Second * 30
	}
	p := &Pandora{
		pubClient: pubClient,
		subServer: subServer,
		component: component,
		pending:   map[string]pendingRequest{},
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

// register allocates a correlation id and the channel the reply on replyTopic
// will arrive on. The channel is buffered so delivering a reply never blocks,
// even if the caller has already given up waiting for it
func (p *Pandora) register(replyTopic string) (string, chan PandoraReply) {
	id := uuid.NewString()
	replies := make(chan PandoraReply, 1)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending[id] = pendingRequest{replyTopic: replyTopic, replies: replies}
	return id, replies
}

func (p *Pandora) unregister(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.pending, id)
}

// deliver hands a reply to the request that asked for it. A reply nobody is
// waiting for anymore (the request timed out, or the recording was resumed
// after a crash) is dropped rather than blocking the Dapr topic handler
func (p *Pandora) deliver(correlationId string, reply PandoraReply, topic string) {
	p.mu.Lock()
	req, isPending := p.pending[correlationId]
	// Compatibility with a Pandora that doesn't echo correlation ids yet : an
	// unlabelled reply is only unambiguous when a single request of that kind
	// is in flight. Drop this branch once every Pandora instance is up to date.
	if !isPending && correlationId == "" {
		if id, only, isSingle := p.onlyPendingOn(topic); isSingle {
			slog.Warn(fmt.Sprintf("[Pandora] :: Reply on topic %q carries no correlation id, assuming it answers the only request in flight (%s). Is Pandora up to date ?", topic, id))
			req, isPending = only, true
		}
	}
	p.mu.Unlock()

	if !isPending {
		slog.Warn(fmt.Sprintf("[Pandora] :: Dropping reply on topic %q, no request is waiting for correlation id %q", topic, correlationId))
		return
	}
	select {
	case req.replies <- reply:
	default:
		slog.Warn(fmt.Sprintf("[Pandora] :: Dropping duplicate reply on topic %q for correlation id %q", topic, correlationId))
	}
}

// onlyPendingOn returns the sole request awaiting a reply on this topic, if
// there is exactly one. Callers must hold the lock
func (p *Pandora) onlyPendingOn(topic string) (string, pendingRequest, bool) {
	var foundId string
	var found pendingRequest
	matches := 0
	for id, req := range p.pending {
		if req.replyTopic != topic {
			continue
		}
		foundId, found, matches = id, req, matches+1
	}
	return foundId, found, matches == 1
}

// RequestTopic addresses a request to one instance of the Pandora pool.
// Must stay in sync with PubSubBroker.requestTopic on Pandora's side.
// An empty instance id targets the plain topic, which is what a Pandora
// running on its own listens to
func RequestTopic(topic, instanceId string) string {
	if instanceId == "" {
		return topic
	}
	return fmt.Sprintf("%s-%s", topic, instanceId)
}

// Start a new recording session on one instance of the pool
func (p *Pandora) Start(instanceId, vcId string) error {
	correlationId, replies := p.register(S_Started)
	defer p.unregister(correlationId)

	err := p.pubClient.PublishEvent(context.Background(), p.component, RequestTopic(P_Start, instanceId), StartPandoraRequest{
		VoiceChannelId: vcId,
		CorrelationId:  correlationId,
	})
	if err != nil {
		return err
	}
	select {
	case <-time.After(p.opt.WaitTimeout):
		err = fmt.Errorf("[Pandora] :: Timeout during initialization, could not start recording")
	case reply := <-replies:
		if reply.Error != nil {
			err = fmt.Errorf("[Pandora] :: error during initialization, could not start recording : %w", reply.Error)
		}
	}
	return err
}

func (p *Pandora) Stop(instanceId, vcId string) (Recording, error) {
	correlationId, replies := p.register(S_Ended)
	defer p.unregister(correlationId)

	err := p.pubClient.PublishEvent(context.Background(), p.component, RequestTopic(P_End, instanceId), StopPandoraRequest{
		VoiceChannelId: vcId,
		CorrelationId:  correlationId,
	})
	if err != nil {
		return Recording{}, err
	}

	var recording Recording
	select {
	case <-time.After(p.opt.WaitTimeout):
		err = fmt.Errorf("[Pandora] :: Timeout, could not end recording")
	case reply := <-replies:
		if reply.Error != nil {
			err = fmt.Errorf("[Pandora] :: could not end recording : %w", reply.Error)
		} else {
			recording = Recording{Ids: reply.Stopped.Ids, Participants: reply.Stopped.Participants}
		}
	}
	return recording, err
}

func (p *Pandora) onStoppedReply(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	reply := StopPandoraReply{}
	err = json.Unmarshal(e.RawData, &reply)
	if err != nil {
		err = fmt.Errorf("[Pandora] :: Received wrong response type from pandora %+v, %w", reply, err)
		slog.Error(err.Error())
	}
	p.deliver(reply.CorrelationId, PandoraReply{
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
	p.deliver(reply.CorrelationId, PandoraReply{
		Started: &reply,
		Stopped: nil,
		Error:   err,
	}, S_Started)
	return false, err
}
