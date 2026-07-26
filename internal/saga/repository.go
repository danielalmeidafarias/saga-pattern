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
	GetAll(filter GetAllSagaFilter) ([]Saga, *pkg.Error)
}

type SagaStepRepository interface {
	FindByID(id string) (*SagaStep, *pkg.Error)
	Update(step *SagaStep) *pkg.Error
}
