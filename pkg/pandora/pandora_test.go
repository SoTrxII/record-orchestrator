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
	err = p.Start("1")
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
	err = p.Start("1")
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
	err = p.Start("1")
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
	_, err = p.Stop("1")
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
	_, err = p.Stop("1")
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
	res, err := p.Stop("1")
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
	assert.Error(t, p.Start("2"), "a stale reply was mistaken for this request's ack")
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

	_, err = p.Stop("1")
	assert.Error(t, err, "a start ack was consumed by a pending stop request")
}
