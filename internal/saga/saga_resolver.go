package saga

import (
	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

// Resolver creates the persisted step plan for a triggering message.
type Resolver interface {
	Resolve(message msg.Message) ([]SagaStep, *pkg.Error)
}
