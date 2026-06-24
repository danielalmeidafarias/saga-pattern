package main

type SagaStep struct {
	runFunc      func() error
	rollbackFunc func() error
	DidRun       bool
}

func (s SagaStep) RollBack() error {
	return s.rollbackFunc()
}

func (s SagaStep) Run() error {
	return s.runFunc()
}

func NewSagaStep(runFunc, rollbackFunc func() error) SagaStep {
	return SagaStep{
		runFunc:      runFunc,
		rollbackFunc: rollbackFunc,
	}
}

type SagaOrchestrator struct {
	SagaStepList []SagaStep
}

func NewSagaOrchestrator() SagaOrchestrator {
	return SagaOrchestrator{}
}

func (sg *SagaOrchestrator) AddStep(saga SagaStep) {
	sg.SagaStepList = append(sg.SagaStepList, saga)
}

func (sg *SagaOrchestrator) Run() error {
	for i := range sg.SagaStepList {
		err := sg.SagaStepList[i].Run()
		if err != nil {
			return err
		}

		sg.SagaStepList[i].DidRun = true
	}
	return nil
}

func (sg *SagaOrchestrator) RollBack() error {
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
