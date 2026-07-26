package pkg

import "context"

// UseCase
type IUseCase[T any] interface {
	Run(ctx context.Context, in T) *Error
}
