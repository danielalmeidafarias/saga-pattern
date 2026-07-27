package pubsub

import (
	"context"
	"encoding/json"
	"sync"

	gpubsub "cloud.google.com/go/pubsub/v2"
	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

type Publisher struct {
	client     *gpubsub.Client
	mu         sync.Mutex
	publishers map[string]*gpubsub.Publisher
}

func NewPublisher(client *gpubsub.Client) *Publisher {
	return &Publisher{client: client, publishers: make(map[string]*gpubsub.Publisher)}
}

func (p *Publisher) Publish(ctx context.Context, message msg.Message) *pkg.Error {
	if message.Topic == "" {
		return pkg.NewError("INVALID_MESSAGE", "message topic is required", nil)
	}
	data, err := json.Marshal(message)
	if err != nil {
		return pkg.NewError("PUBSUB_SERIALIZE_FAILED", "serialize Pub/Sub message", err)
	}
	publisher := p.publisher(message.Topic)
	if _, err := publisher.Publish(ctx, &gpubsub.Message{Data: data}).Get(ctx); err != nil {
		return pkg.NewError("PUBSUB_PUBLISH_FAILED", "publish Pub/Sub message", err)
	}
	return nil
}

func (p *Publisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, publisher := range p.publishers {
		publisher.Stop()
	}
}

func (p *Publisher) publisher(topic string) *gpubsub.Publisher {
	p.mu.Lock()
	defer p.mu.Unlock()
	if publisher := p.publishers[topic]; publisher != nil {
		return publisher
	}
	publisher := p.client.Publisher(topic)
	p.publishers[topic] = publisher
	return publisher
}
