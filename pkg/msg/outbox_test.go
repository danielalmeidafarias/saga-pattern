package msg

import (
	"context"
	"testing"

	"github.com/danielalmeidafarias/saga-pattern/pkg"
)

func TestOutboxDispatcherLeavesFailedMessagePending(t *testing.T) {
	store := &memoryOutbox{pending: []Message{{ID: "1", Topic: "events"}}}
	publisher := &outboxTestPublisher{fail: true}
	dispatcher := NewOutboxDispatcher(store, publisher)
	if err := dispatcher.Dispatch(context.Background(), 10); err == nil {
		t.Fatal("expected publish failure")
	}
	if store.published != "" {
		t.Fatal("failed message was marked as published")
	}
	publisher.fail = false
	if err := dispatcher.Dispatch(context.Background(), 10); err != nil {
		t.Fatal(err.Message)
	}
	if store.published != "1" {
		t.Fatalf("published id: got %s", store.published)
	}
}

type memoryOutbox struct {
	pending   []Message
	published string
}

func (s *memoryOutbox) Enqueue(message Message) *pkg.Error {
	s.pending = append(s.pending, message)
	return nil
}
func (s *memoryOutbox) Pending(int) ([]Message, *pkg.Error) { return s.pending, nil }
func (s *memoryOutbox) MarkPublished(id string) *pkg.Error {
	s.published = id
	s.pending = nil
	return nil
}

type outboxTestPublisher struct{ fail bool }

func (p *outboxTestPublisher) Publish(context.Context, Message) *pkg.Error {
	if p.fail {
		return pkg.NewError("FAILED", "expected", nil)
	}
	return nil
}
