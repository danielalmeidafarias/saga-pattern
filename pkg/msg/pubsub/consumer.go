package pubsub

import (
	"context"
	"encoding/json"
	"errors"

	gpubsub "cloud.google.com/go/pubsub/v2"
	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

// Consumer adapts a Google Pub/Sub subscription to the application's message handler.
type Consumer struct {
	subscriber *gpubsub.Subscriber
}

func NewConsumer(subscriber *gpubsub.Subscriber) *Consumer {
	return &Consumer{subscriber: subscriber}
}

// Subscribe blocks until ctx is cancelled or Pub/Sub returns a non-retryable error.
// A handler error is technical: the message is nacked for retry. Business failures
// must be published as a response message and return nil here.
func (c *Consumer) Subscribe(ctx context.Context, handler msg.Handler) *pkg.Error {
	err := c.subscriber.Receive(ctx, func(messageCtx context.Context, received *gpubsub.Message) {
		c.handle(messageCtx, received.Data, received.Ack, received.Nack, handler)
	})
	if err == nil || errors.Is(err, context.Canceled) {
		return nil
	}
	return pkg.NewError("PUBSUB_RECEIVE_FAILED", "receive Pub/Sub messages", err)
}

func (c *Consumer) handle(ctx context.Context, data []byte, ack, nack func(), handler msg.Handler) {
	var message msg.Message
	if err := json.Unmarshal(data, &message); err != nil {
		nack()
		return
	}
	if err := handler(ctx, message); err != nil {
		nack()
		return
	}
	ack()
}
