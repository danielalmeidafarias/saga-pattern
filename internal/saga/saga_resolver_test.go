package saga

import (
	"testing"

	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/contracts"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

func TestMessageResolverSelectsDefinitionByMessageType(t *testing.T) {
	resolver := NewResolver(map[msg.MessageType]ResolveFunc{
		contracts.OrderCreated: func(msg.Message) ([]SagaStep, *pkg.Error) {
			return []SagaStep{{ID: "step-1"}}, nil
		},
	})
	steps, err := resolver.Resolve(msg.Message{Type: contracts.OrderCreated})
	if err != nil || len(steps) != 1 || steps[0].ID != "step-1" {
		t.Fatalf("resolved steps: %+v, err: %+v", steps, err)
	}
	if _, err := resolver.Resolve(msg.Message{Type: "unknown"}); err == nil {
		t.Fatal("unsupported trigger must fail")
	}
}
