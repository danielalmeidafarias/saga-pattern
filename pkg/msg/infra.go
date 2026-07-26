package msg

import (
	"context"

	"github.com/danielalmeidafarias/saga-pattern/pkg"
)

type Publisher interface {
	Publish(ctx context.Context, message Message) *pkg.Error
}

type Handler func(ctx context.Context, message Message) *pkg.Error

type Subscriber interface {
	Subscribe(ctx context.Context, handler Handler) *pkg.Error
}
