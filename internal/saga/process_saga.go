package saga

import (
	"context"

	"github.com/danielalmeidafarias/saga-pattern/pkg"
)

// ProcessSagaUseCase is a recovery job. It advances only pending work;
// it never publishes every step of a Saga in a single run.
type ProcessSagaUseCase struct {
	repository   SagaRepository
	orchestrator *SagaOrchestrator
}

type ProcessSagaUseCaseInput struct {
	Status    Status
	BatchSize int
}

func NewProcessSagaUseCase(repository SagaRepository, orchestrator *SagaOrchestrator) *ProcessSagaUseCase {
	return &ProcessSagaUseCase{repository: repository, orchestrator: orchestrator}
}

func (uc *ProcessSagaUseCase) Run(ctx context.Context, input ProcessSagaUseCaseInput) *pkg.Error {
	status := input.Status
	sagas, err := uc.repository.GetAll(GetAllSagaFilter{Status: &status, Limit: input.BatchSize})
	if err != nil {
		return err
	}

	for _, saga := range sagas {
		if err := uc.orchestrator.Advance(ctx, saga.ID); err != nil {
			return err
		}
	}
	return nil
}
