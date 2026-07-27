package msg

import (
	"context"

	"github.com/google/uuid"

	"github.com/danielalmeidafarias/saga-pattern/pkg"
)

type Outbox interface {
	Enqueue(message Message) *pkg.Error
	Pending(limit int) ([]Message, *pkg.Error)
	MarkPublished(id string) *pkg.Error
}

type OutboxPublisher struct{ store Outbox }

func NewOutboxPublisher(store Outbox) *OutboxPublisher { return &OutboxPublisher{store: store} }

func (p *OutboxPublisher) Publish(_ context.Context, message Message) *pkg.Error {
	if message.ID == "" {
		message.ID = uuid.NewString()
	}
	return p.store.Enqueue(message)
}

type OutboxDispatcher struct {
	store     Outbox
	publisher Publisher
}

func NewOutboxDispatcher(store Outbox, publisher Publisher) *OutboxDispatcher {
	return &OutboxDispatcher{store: store, publisher: publisher}
}

func (d *OutboxDispatcher) Dispatch(ctx context.Context, limit int) *pkg.Error {
	messages, err := d.store.Pending(limit)
	if err != nil {
		return err
	}
	for _, message := range messages {
		if err := d.publisher.Publish(ctx, message); err != nil {
			return err
		}
		if err := d.store.MarkPublished(message.ID); err != nil {
			return err
		}
	}
	return nil
}
