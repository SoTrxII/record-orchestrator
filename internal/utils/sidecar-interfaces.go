// This package is mainly interfaces used elsewhere in the codebase.
// All of these are used for easier mocking and testing
package utils

import (
	"context"
	dapr "github.com/dapr/go-sdk/client"
	"github.com/dapr/go-sdk/service/common"
)

type PublishEventOption = dapr.PublishEventOption
type Publisher interface {
	PublishEvent(ctx context.Context, pubsubName string, topicName string, data interface{}, opts ...PublishEventOption) error
}
type Subscriber interface {
	AddTopicEventHandler(sub *common.Subscription, fn common.TopicEventHandler) error
}

type DataContent = dapr.DataContent
type Invoker interface {
	InvokeMethodWithContent(ctx context.Context, appID, method, verb string, content *DataContent) ([]byte, error)
}
