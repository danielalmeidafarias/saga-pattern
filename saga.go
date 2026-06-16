package main

type SagaStep struct {
	rollbackFunc func() error
	runcFunc     func() error
	DidRun       bool
}

func (s SagaStep) RollBack() error {
	return s.rollbackFunc()
}

func (s SagaStep) Run() error {
	return s.rollbackFunc()
}

func NewSagaStep(rollbackFunc, runFunc func() error) SagaStep {
	return SagaStep{
		rollbackFunc: rollbackFunc,
		runcFunc:     runFunc,
	}
}

type SagaOrchestrator struct {
	SagaStepList []SagaStep
}

func NewSagaOrchestrator() SagaOrchestrator {
	return SagaOrchestrator{}
}

func (sg SagaOrchestrator) AddStep(saga SagaStep) {
	sg.SagaStepList = append(sg.SagaStepList, saga)
}

func (sg SagaOrchestrator) Run() error {
	for _, step := range sg.SagaStepList {
		err := step.Run()
		if err != nil {
			return err
		}

		step.DidRun = true
	}
	return nil
}

func (sg SagaOrchestrator) RollBack() error {
	if len(sg.SagaStepList) == 0 {
		return nil
	}

	for i := len(sg.SagaStepList) - 1; i >= 0; i-- {
		step := sg.SagaStepList[i]
		if !step.DidRun {
			continue
		}

		if err := step.RollBack(); err != nil {
			return err
		}
	}

	return nil
}
