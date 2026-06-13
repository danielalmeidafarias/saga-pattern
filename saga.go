package main

type Saga interface {
	Run()
	RollBack()
	AddStep(step ISagaStep) error
}

type ISagaStep interface {
	Run() error
	RollBack() error
	// MaxRetries() int
	// RetryCount() int
}

type SagaOrchestrator struct {
	SagaList   []ISagaStep
	FailedStep int
}

func (sg *SagaOrchestrator) AddStep(saga ISagaStep) {
	sg.SagaList = append(sg.SagaList, saga)
}

type SagaRollbackError struct {
	Err        error
	FailedStep ISagaStep
}

func (sg *SagaOrchestrator) RollBack() *SagaRollbackError {
	if len(sg.SagaList) == 0 || sg.FailedStep == 0 {
		return nil
	}

	s := sg.SagaList[:sg.FailedStep]
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}

	for _, s := range s {
		err := s.RollBack()
		if err != nil {
			return &SagaRollbackError{
				Err:        err,
				FailedStep: s,
			}
		}
	}

	return nil
}
