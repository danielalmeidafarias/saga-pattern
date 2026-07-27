package pubsub

import (
	"context"
	"testing"

	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

func TestConsumerHandle_AcknowledgesSuccessfulHandler(t *testing.T) {
	consumer := &Consumer{}
	acked := false
	nacked := false

	consumer.handle(context.Background(), []byte(`{"type":"order.created","version":1}`), func() { acked = true }, func() { nacked = true }, func(_ context.Context, message msg.Message) *pkg.Error {
		if message.Type != msg.OrderCreatedMessage {
			t.Fatalf("message type: got %s", message.Type)
		}
		return nil
	})

	if !acked || nacked {
		t.Fatalf("ack=%t nack=%t, want ack only", acked, nacked)
	}
}

func TestConsumerHandle_NacksMalformedMessage(t *testing.T) {
	consumer := &Consumer{}
	acked := false
	nacked := false

	consumer.handle(context.Background(), []byte(`{`), func() { acked = true }, func() { nacked = true }, func(context.Context, msg.Message) *pkg.Error {
		t.Fatal("handler must not run")
		return nil
	})

	if acked || !nacked {
		t.Fatalf("ack=%t nack=%t, want nack only", acked, nacked)
	}
}

func TestConsumerHandle_NacksTechnicalFailure(t *testing.T) {
	consumer := &Consumer{}
	acked := false
	nacked := false

	consumer.handle(context.Background(), []byte(`{"type":"order.created","version":1}`), func() { acked = true }, func() { nacked = true }, func(context.Context, msg.Message) *pkg.Error {
		return pkg.NewError("DATABASE_UNAVAILABLE", "save order", nil)
	})

	if acked || !nacked {
		t.Fatalf("ack=%t nack=%t, want nack only", acked, nacked)
	}
}
