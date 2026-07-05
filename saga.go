package main

import "errors"

type RollbackStep struct {
	rollbackFunc func() error
}

func NewRollBackStep(rollbackFunc func() error) RollbackStep {
	return RollbackStep{
		rollbackFunc: rollbackFunc,
	}
}

func (s RollbackStep) RollBack() error {
	return s.rollbackFunc()
}

type SagaStep struct {
	runFunc func() error
	DidRun  bool
	RollbackStep
}

func NewSagaStep(runFunc, rollbackFunc func() error) SagaStep {
	return SagaStep{
		runFunc:      runFunc,
		RollbackStep: NewRollBackStep(rollbackFunc),
	}
}

func (s SagaStep) Run() error {
	return s.runFunc()
}

type RollBackOrchestrator struct {
	RollbackStepList []RollbackStep
}

func (sg *RollBackOrchestrator) RollBack() error {
	if len(sg.RollbackStepList) == 0 {
		return nil
	}

	for i := len(sg.RollbackStepList) - 1; i >= 0; i-- {
		step := sg.RollbackStepList[i]

		if err := step.RollBack(); err != nil {
			return err
		}
	}

	return nil
}

type ISagaOrchestrator interface {
	Run() error
	RollBack() error
}

type SagaOrchestrator struct {
	SagaStepList []SagaStep
	RollBackOrchestrator
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
			// Add the failed step to the rollback list
			// As Step implements IRollbackStep, we can add it directly to the RollbackStepList
			sg.RollbackStepList = append(sg.RollbackStepList, NewRollBackStep(sg.SagaStepList[i].rollbackFunc))
			return err
		}

		sg.SagaStepList[i].DidRun = true
	}
	return nil
}

type SagaOrchestratorV1 struct {
	SagaStepList []SagaStep
	RollBackOrchestrator
	RunFunc func() error
}

func NewSagaOrchestratorV1(runFunc func() error) SagaOrchestratorV1 {
	return SagaOrchestratorV1{
		RunFunc: runFunc,
	}
}

func (sg *SagaOrchestratorV1) Run() error {
	if sg.RunFunc != nil {
		return sg.RunFunc()
	}
	return nil
}

// UseCase
type IUseCase[T any] interface {
	Run(in T) error
}

type UseCaseWithSagaAbstract[T any] interface {
	runFunc(in T) (*SagaOrchestrator, error)
}

type UseCaseWithSaga[T any] struct {
	abstract UseCaseWithSagaAbstract[T]
}

func NewUseCaseWithSaga[T any](
	abstract UseCaseWithSagaAbstract[T],
) UseCaseWithSaga[T] {
	return UseCaseWithSaga[T]{
		abstract: abstract,
	}
}

func (uc UseCaseWithSaga[T]) Run(in T) error {
	sagaOrchestrator, err := uc.abstract.runFunc(in)
	if err == nil {
		return nil
	}

	if sagaOrchestrator != nil {
		if errRollback := sagaOrchestrator.RollBack(); errRollback != nil {
			return errors.Join(err, errRollback)
		}
	}

	return err
}
