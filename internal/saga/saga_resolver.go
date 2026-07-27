package saga

import (
	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

// Resolver creates the persisted step plan for a triggering message.
type Resolver interface {
	Resolve(message msg.Message) ([]SagaStep, *pkg.Error)
}

type ResolveFunc func(message msg.Message) ([]SagaStep, *pkg.Error)

type MessageResolver struct {
	definitions map[msg.MessageType]ResolveFunc
}

func NewResolver(definitions map[msg.MessageType]ResolveFunc) *MessageResolver {
	return &MessageResolver{definitions: definitions}
}

func (r *MessageResolver) Resolve(message msg.Message) ([]SagaStep, *pkg.Error) {
	resolve := r.definitions[message.Type]
	if resolve == nil {
		return nil, pkg.NewError("INVALID_SAGA", "unsupported saga trigger", nil)
	}
	return resolve(message)
}
