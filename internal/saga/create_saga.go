package saga

import (
	"context"

	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

type CreateSagaUseCase struct {
	repository SagaRepository
	resolver   Resolver
}

type CreateSagaUseCaseInput struct {
	Message msg.Message
}

func NewCreateSagaUseCase(repository SagaRepository, resolver Resolver) *CreateSagaUseCase {
	return &CreateSagaUseCase{repository: repository, resolver: resolver}
}

func (uc *CreateSagaUseCase) Run(_ context.Context, in CreateSagaUseCaseInput) *pkg.Error {
	steps, err := uc.resolver.Resolve(in.Message)
	if err != nil {
		return err
	}

	for i := range steps {
		steps[i].SagaID = in.Message.SagaID
		steps[i].Status = StepPending
	}

	return uc.repository.Save(&Saga{
		ID:       in.Message.SagaID,
		OrderID:  in.Message.OrderID,
		Trigger:  in.Message.Type,
		Status:   StatusRunning,
		StepList: steps,
	})
}
