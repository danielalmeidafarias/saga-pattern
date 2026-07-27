package saga

import "github.com/danielalmeidafarias/saga-pattern/pkg"

type GetAllSagaFilter struct {
	Status *Status
	Limit  int
}

type SagaRepository interface {
	Save(saga *Saga) *pkg.Error
	FindByID(id string) (*Saga, *pkg.Error)
	Update(saga *Saga) *pkg.Error
	UpdateResult(saga *Saga, step *SagaStep) *pkg.Error
	GetAll(filter GetAllSagaFilter) ([]Saga, *pkg.Error)
}

type SagaStepRepository interface {
	FindStepByID(id string) (*SagaStep, *pkg.Error)
	UpdateStep(step *SagaStep) *pkg.Error
}
