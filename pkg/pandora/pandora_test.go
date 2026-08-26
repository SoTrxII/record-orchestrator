package pandora

import (
	"context"
	"encoding/json"
	"github.com/dapr/go-sdk/service/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"record-orchestrator/internal/utils"
	"testing"
	"time"
)

type mockPublisher struct {
	mock.Mock
	utils.Publisher
}

// Implement publisher interface
func (m *mockPublisher) PublishEvent(ctx context.Context, pubsubName string, topicName string, data interface{}, opts ...utils.PublishEventOption) error {
	args := m.Called(ctx, pubsubName, topicName, data, opts)
	return args.Error(0)
}

type mockSubscriber struct {
	mock.Mock
	utils.Subscriber
}

// Implement subscriber interface
func (m *mockSubscriber) AddTopicEventHandler(sub *common.Subscription, fn common.TopicEventHandler) error {
	args := m.Called(sub, fn)
	return args.Error(0)
}

// Reply received in time
func TestPandora_OnStartedReply_Ok(t *testing.T) {
	pub := mockPublisher{}
	sub := mockSubscriber{}
	sub.On("AddTopicEventHandler", mock.Anything, mock.Anything).Return(nil)
	pub.On("PublishEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	p, err := NewPandora(&pub, &sub, "", PandoraOpt{})
	assert.NoError(t, err)

	payload, err := json.Marshal(StartPandoraReply{VoiceChannelId: "1"})
	assert.NoError(t, err)
	done := make(chan bool)
	go func() {
		select {
		case <-time.After(1 * time.Second):
			ok, err := p.onStartedReply(context.Background(), &common.TopicEvent{RawData: payload})
			assert.False(t, ok)
			assert.NoError(t, err)
			done <- true
		}
	}()
	err = p.Start("", "1")
	pub.AssertExpectations(t)
	sub.AssertExpectations(t)
	assert.NoError(t, err)
	<-done
}

func TestPandora_OnStartedReply_WrongReply(t *testing.T) {
	pub := mockPublisher{}
	sub := mockSubscriber{}
	sub.On("AddTopicEventHandler", mock.Anything, mock.Anything).Return(nil)
	pub.On("PublishEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	p, err := NewPandora(&pub, &sub, "", PandoraOpt{})
	assert.NoError(t, err)

	done := make(chan bool)
	go func() {
		select {
		case <-time.After(1 * time.Second):
			ok, err := p.onStartedReply(context.Background(), &common.TopicEvent{RawData: []byte("wrong")})
			assert.False(t, ok)
			assert.Error(t, err)
			done <- true
		}
	}()
	err = p.Start("", "1")
	pub.AssertExpectations(t)
	sub.AssertExpectations(t)
	assert.Error(t, err)
	<-done
}

func TestPandora_OnStartedReply_Timeout(t *testing.T) {
	pub := mockPublisher{}
	sub := mockSubscriber{}
	sub.On("AddTopicEventHandler", mock.Anything, mock.Anything).Return(nil)
	pub.On("PublishEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	timeout := time.Second * 2
	p, err := NewPandora(&pub, &sub, "", PandoraOpt{WaitTimeout: timeout})
	assert.NoError(t, err)

	go func() {
		select {
		case <-time.After(timeout + 1*time.Second):
			_, err := p.onStartedReply(context.Background(), &common.TopicEvent{})
			assert.Error(t, err)
		}
	}()
	err = p.Start("", "1")
	pub.AssertExpectations(t)
	sub.AssertExpectations(t)
	assert.Error(t, err)
}

func TestPandora_OnStoppedReply_Timeout(t *testing.T) {
	pub := mockPublisher{}
	sub := mockSubscriber{}
	sub.On("AddTopicEventHandler", mock.Anything, mock.Anything).Return(nil)
	pub.On("PublishEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	timeout := time.Second * 2
	p, err := NewPandora(&pub, &sub, "", PandoraOpt{WaitTimeout: timeout})
	assert.NoError(t, err)

	go func() {
		select {
		case <-time.After(timeout + 1*time.Second):
			_, err := p.onStoppedReply(context.Background(), &common.TopicEvent{})
			assert.Error(t, err)
		}
	}()
	_, err = p.Stop("", "1")
	pub.AssertExpectations(t)
	sub.AssertExpectations(t)
	assert.Error(t, err)
}
func TestPandora_OnStoppedReply_WrongReply(t *testing.T) {
	pub := mockPublisher{}
	sub := mockSubscriber{}
	sub.On("AddTopicEventHandler", mock.Anything, mock.Anything).Return(nil)
	pub.On("PublishEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	p, err := NewPandora(&pub, &sub, "", PandoraOpt{})
	assert.NoError(t, err)

	done := make(chan bool)
	go func() {
		select {
		case <-time.After(1 * time.Second):
			ok, err := p.onStoppedReply(context.Background(), &common.TopicEvent{RawData: []byte("wrong")})
			assert.False(t, ok)
			assert.Error(t, err)
			done <- true
		}
	}()
	_, err = p.Stop("", "1")
	assert.Error(t, err)
	pub.AssertExpectations(t)
	sub.AssertExpectations(t)
	<-done
}

func TestPandora_OnStoppedReply_Ok(t *testing.T) {
	pub := mockPublisher{}
	sub := mockSubscriber{}
	sub.On("AddTopicEventHandler", mock.Anything, mock.Anything).Return(nil)
	pub.On("PublishEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	p, err := NewPandora(&pub, &sub, "", PandoraOpt{})
	assert.NoError(t, err)

	ids := []string{"1", "2", "3"}
	payload, err := json.Marshal(StopPandoraReply{Ids: ids})

	done := make(chan bool)
	go func() {
		select {
		case <-time.After(1 * time.Second):
			ok, err := p.onStoppedReply(context.Background(), &common.TopicEvent{RawData: payload})
			assert.False(t, ok)
			assert.NoError(t, err)
			done <- true
		}
	}()
	res, err := p.Stop("", "1")
	assert.NoError(t, err)
	assert.Equal(t, ids, res)
	pub.AssertExpectations(t)
	sub.AssertExpectations(t)
	<-done
}

// Pandora publishes its acks on dedicated topics : we must not subscribe to
// the topics we publish on, or we'd ack our own requests.
// The literals below are Pandora's own PubSubBroker.TOPICS : they're spelled
// out on purpose so this test pins the wire contract rather than our constants
func TestPandora_SubscribesToReplyTopicsOnly(t *testing.T) {
	pub := mockPublisher{}
	sub := mockSubscriber{}
	subscribed := []string{}
	sub.On("AddTopicEventHandler", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			subscribed = append(subscribed, args.Get(0).(*common.Subscription).Topic)
		}).Return(nil)

	_, err := NewPandora(&pub, &sub, "", PandoraOpt{})
	assert.NoError(t, err)

	assert.ElementsMatch(t,
		[]string{"startedRecordingDiscord", "stoppedRecordingDiscord"},
		subscribed,
		"we must subscribe to Pandora's reply topics, not to the ones we publish on")
}

// The topics we publish on must be the ones Pandora subscribes to
func TestPandora_PublishesOnRequestTopics(t *testing.T) {
	assert.Equal(t, "startRecordingDiscord", P_Start)
	assert.Equal(t, "stopRecordingDiscord", P_End)
}

// A reply arriving after its request timed out must neither block the topic
// handler nor be handed to the next request
func TestPandora_LateReplyIsNotReusedByNextRequest(t *testing.T) {
	pub := mockPublisher{}
	sub := mockSubscriber{}
	sub.On("AddTopicEventHandler", mock.Anything, mock.Anything).Return(nil)
	pub.On("PublishEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	timeout := 200 * time.Millisecond
	p, err := NewPandora(&pub, &sub, "", PandoraOpt{WaitTimeout: timeout})
	assert.NoError(t, err)

	// Nobody is waiting : the handler must return instead of blocking forever
	payload, err := json.Marshal(StartPandoraReply{VoiceChannelId: "1"})
	assert.NoError(t, err)
	handlerDone := make(chan bool)
	go func() {
		_, _ = p.onStartedReply(context.Background(), &common.TopicEvent{RawData: payload})
		handlerDone <- true
	}()
	select {
	case <-handlerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("the topic handler blocked with no request in flight")
	}

	// That stale reply must not satisfy the next request
	assert.Error(t, p.Start("", "2"), "a stale reply was mistaken for this request's ack")
}

// A start ack must never be consumed by a pending stop request
func TestPandora_StartAckDoesNotSatisfyStop(t *testing.T) {
	pub := mockPublisher{}
	sub := mockSubscriber{}
	sub.On("AddTopicEventHandler", mock.Anything, mock.Anything).Return(nil)
	pub.On("PublishEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	timeout := 500 * time.Millisecond
	p, err := NewPandora(&pub, &sub, "", PandoraOpt{WaitTimeout: timeout})
	assert.NoError(t, err)

	payload, err := json.Marshal(StartPandoraReply{VoiceChannelId: "1"})
	assert.NoError(t, err)
	go func() {
		time.Sleep(50 * time.Millisecond)
		_, _ = p.onStartedReply(context.Background(), &common.TopicEvent{RawData: payload})
	}()

	_, err = p.Stop("", "1")
	assert.Error(t, err, "a start ack was consumed by a pending stop request")
}

// The whole point of the correlation id : two sessions stopped at the same
// time must each get their own recording ids back, whatever the reply order
func TestPandora_ConcurrentStopsGetTheirOwnReplies(t *testing.T) {
	pub := mockPublisher{}
	sub := mockSubscriber{}
	sub.On("AddTopicEventHandler", mock.Anything, mock.Anything).Return(nil)

	published := make(chan StopPandoraRequest, 2)
	pub.On("PublishEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			published <- args.Get(3).(StopPandoraRequest)
		}).Return(nil)

	p, err := NewPandora(&pub, &sub, "", PandoraOpt{WaitTimeout: 5 * time.Second})
	assert.NoError(t, err)

	type result struct {
		ids []string
		err error
	}
	resA, resB := make(chan result, 1), make(chan result, 1)
	go func() { ids, err := p.Stop("", "channel-A"); resA <- result{ids, err} }()
	go func() { ids, err := p.Stop("", "channel-B"); resB <- result{ids, err} }()

	// Both requests are out : map each correlation id to its voice channel
	correlationOf := map[string]string{}
	for i := 0; i < 2; i++ {
		req := <-published
		correlationOf[req.VoiceChannelId] = req.CorrelationId
	}
	assert.Len(t, correlationOf, 2)
	assert.NotEqual(t, correlationOf["channel-A"], correlationOf["channel-B"],
		"each request must get its own correlation id")

	// Reply in the opposite order to make sure ordering plays no part
	for _, r := range []struct {
		channel string
		ids     []string
	}{{"channel-B", []string{"rec-B"}}, {"channel-A", []string{"rec-A"}}} {
		payload, err := json.Marshal(StopPandoraReply{
			Ids:           r.ids,
			CorrelationId: correlationOf[r.channel],
		})
		assert.NoError(t, err)
		_, _ = p.onStoppedReply(context.Background(), &common.TopicEvent{RawData: payload})
	}

	a, b := <-resA, <-resB
	assert.NoError(t, a.err)
	assert.NoError(t, b.err)
	assert.Equal(t, []string{"rec-A"}, a.ids, "channel-A got another session's recording ids")
	assert.Equal(t, []string{"rec-B"}, b.ids, "channel-B got another session's recording ids")
}

// A reply for a session we know nothing about must not satisfy a pending request
func TestPandora_UnknownCorrelationIdIsDropped(t *testing.T) {
	pub := mockPublisher{}
	sub := mockSubscriber{}
	sub.On("AddTopicEventHandler", mock.Anything, mock.Anything).Return(nil)
	pub.On("PublishEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	p, err := NewPandora(&pub, &sub, "", PandoraOpt{WaitTimeout: 300 * time.Millisecond})
	assert.NoError(t, err)

	payload, err := json.Marshal(StopPandoraReply{Ids: []string{"nope"}, CorrelationId: "some-other-session"})
	assert.NoError(t, err)
	go func() {
		time.Sleep(50 * time.Millisecond)
		_, _ = p.onStoppedReply(context.Background(), &common.TopicEvent{RawData: payload})
	}()

	ids, err := p.Stop("", "1")
	assert.Error(t, err, "a reply for an unrelated session was accepted")
	assert.Empty(t, ids)
}

// Compatibility shim : an unlabelled reply is only safe to assume when a
// single request of that kind is waiting. With two, it must be dropped
func TestPandora_UnlabelledReplyIsDroppedWhenAmbiguous(t *testing.T) {
	pub := mockPublisher{}
	sub := mockSubscriber{}
	sub.On("AddTopicEventHandler", mock.Anything, mock.Anything).Return(nil)
	published := make(chan StopPandoraRequest, 2)
	pub.On("PublishEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { published <- args.Get(3).(StopPandoraRequest) }).Return(nil)

	p, err := NewPandora(&pub, &sub, "", PandoraOpt{WaitTimeout: 500 * time.Millisecond})
	assert.NoError(t, err)

	errs := make(chan error, 2)
	go func() { _, err := p.Stop("", "A"); errs <- err }()
	go func() { _, err := p.Stop("", "B"); errs <- err }()
	<-published
	<-published

	// No correlation id, two candidates : we cannot guess
	payload, err := json.Marshal(StopPandoraReply{Ids: []string{"ambiguous"}})
	assert.NoError(t, err)
	_, _ = p.onStoppedReply(context.Background(), &common.TopicEvent{RawData: payload})

	assert.Error(t, <-errs)
	assert.Error(t, <-errs)
}

// Requests are addressed to one instance of the pool. The suffix must match
// PubSubBroker.requestTopic on Pandora's side, hence the literals
func TestPandora_RequestTopicAddressesAnInstance(t *testing.T) {
	assert.Equal(t, "startRecordingDiscord-pandora-0", RequestTopic(P_Start, "pandora-0"))
	assert.Equal(t, "stopRecordingDiscord-pandora-0", RequestTopic(P_End, "pandora-0"))
	// A lone Pandora listens on the plain topics
	assert.Equal(t, "startRecordingDiscord", RequestTopic(P_Start, ""))
	assert.Equal(t, "stopRecordingDiscord", RequestTopic(P_End, ""))
	// Two instances never share a request topic
	assert.NotEqual(t, RequestTopic(P_Start, "pandora-0"), RequestTopic(P_Start, "pandora-1"))
}

// Start and Stop must publish on the topic of the instance they target
func TestPandora_PublishesOnTheTargetedInstanceTopic(t *testing.T) {
	pub := mockPublisher{}
	sub := mockSubscriber{}
	sub.On("AddTopicEventHandler", mock.Anything, mock.Anything).Return(nil)
	topics := make(chan string, 2)
	pub.On("PublishEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { topics <- args.Get(2).(string) }).Return(nil)

	p, err := NewPandora(&pub, &sub, "", PandoraOpt{WaitTimeout: 100 * time.Millisecond})
	assert.NoError(t, err)

	_ = p.Start("pandora-1", "channel-A")
	assert.Equal(t, "startRecordingDiscord-pandora-1", <-topics)

	_, _ = p.Stop("pandora-2", "channel-B")
	assert.Equal(t, "stopRecordingDiscord-pandora-2", <-topics)
}
