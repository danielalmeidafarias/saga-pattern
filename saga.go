package main

type ISagaStep interface {
	RollBack() error
}

type SagaStepRollbackFunc func(args ...any) error

type SagaStep struct {
	rollbackFunc SagaStepRollbackFunc
}

func (s SagaStep) RollBack() error {
	return s.rollbackFunc()
}

func NewSagaStep(rollbackFunc SagaStepRollbackFunc) SagaStep {
	return SagaStep{
		rollbackFunc: rollbackFunc,
	}
}

type RollbackOrchestrator struct {
	SagaList []ISagaStep
}

func NewRollbackOrchestrator() RollbackOrchestrator {
	return RollbackOrchestrator{}
}

func (sg RollbackOrchestrator) AddStep(saga ISagaStep) {
	sg.SagaList = append(sg.SagaList, saga)
}

func (sg RollbackOrchestrator) RollBack() error {
	if len(sg.SagaList) == 0 {
		return nil
	}

	for i := len(sg.SagaList) - 1; i >= 0; i-- {
		step := sg.SagaList[i]

		if err := step.RollBack(); err != nil {
			return err
		}
	}

	return nil
}
