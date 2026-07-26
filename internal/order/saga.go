package order

type IUseCase[T any] interface {
	Run(input T) error
}

type SagaStep struct {
	run        func() error
	compensate func() error
}

func NewSagaStep(run, compensate func() error) SagaStep {
	return SagaStep{run: run, compensate: compensate}
}

type SagaOrchestrator struct {
	steps []SagaStep
}

func NewSagaOrchestrator() *SagaOrchestrator {
	return &SagaOrchestrator{}
}

func (o *SagaOrchestrator) AddStep(step SagaStep) {
	o.steps = append(o.steps, step)
}

func (o *SagaOrchestrator) Run() error {
	completed := make([]SagaStep, 0, len(o.steps))
	for _, step := range o.steps {
		if err := step.run(); err != nil {
			for i := len(completed) - 1; i >= 0; i-- {
				_ = completed[i].compensate()
			}
			return err
		}
		completed = append(completed, step)
	}
	return nil
}

type sagaUseCase[T any] interface {
	runFunc(T) (*SagaOrchestrator, error)
}

type useCaseWithSaga[T any] struct {
	useCase sagaUseCase[T]
}

func NewUseCaseWithSaga[T any](useCase sagaUseCase[T]) IUseCase[T] {
	return useCaseWithSaga[T]{useCase: useCase}
}

func (u useCaseWithSaga[T]) Run(input T) error {
	_, err := u.useCase.runFunc(input)
	return err
}
