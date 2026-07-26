package saga

import (
	"context"

	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

type SagaOrchestrator struct {
	sagaRepository SagaRepository
	stepRepository SagaStepRepository
	publisher      msg.Publisher
}

func NewSagaOrchestrator(sagaRepository SagaRepository, stepRepository SagaStepRepository, publisher msg.Publisher) *SagaOrchestrator {
	return &SagaOrchestrator{
		sagaRepository: sagaRepository,
		stepRepository: stepRepository,
		publisher:      publisher,
	}
}

// Advance publishes all pending commands in the active phase.
func (o *SagaOrchestrator) Advance(ctx context.Context, sagaID string) *pkg.Error {
	saga, err := o.sagaRepository.FindByID(sagaID)
	if err != nil {
		return err
	}

	switch saga.Status {
	case StatusRunning:
		return o.advanceForward(ctx, saga)
	case StatusCompensating:
		return o.advanceCompensation(ctx, saga)
	default:
		return nil
	}
}

// HandleResult records a command result and advances the Saga exactly once.
func (o *SagaOrchestrator) HandleResult(ctx context.Context, result msg.Message) *pkg.Error {
	if result.SagaID == "" || result.StepID == "" {
		return invalidSagaError("result must contain sagaId and stepId")
	}

	saga, err := o.sagaRepository.FindByID(result.SagaID)
	if err != nil {
		return err
	}

	step, err := o.stepRepository.FindByID(result.StepID)
	if err != nil {
		return err
	}
	if step.SagaID != saga.ID {
		return invalidSagaError("step does not belong to saga")
	}

	switch step.Status {
	case StepDispatched:
		if result.Failure != nil {
			step.Status = StepFailed
			saga.Status = StatusCompensating
		} else {
			step.Status = StepSucceeded
		}
	case StepCompensating:
		if result.Failure != nil {
			step.Status = StepFailed
			saga.Status = StatusFailed
		} else {
			step.Status = StepCompensated
		}
	case StepSucceeded, StepCompensated:
		return nil // Pub/Sub can deliver a result more than once.
	default:
		return invalidSagaError("result received for a step that was not dispatched")
	}

	if err := o.stepRepository.Update(step); err != nil {
		return err
	}
	if err := o.sagaRepository.Update(saga); err != nil {
		return err
	}

	return o.Advance(ctx, saga.ID)
}

func (o *SagaOrchestrator) advanceForward(ctx context.Context, saga *Saga) *pkg.Error {
	phase, ok := activePhase(saga.StepList, StepPending, StepDispatched)
	if !ok {
		saga.Status = StatusSucceeded
		return o.sagaRepository.Update(saga)
	}

	for i := range saga.StepList {
		step := &saga.StepList[i]
		if step.Phase != phase {
			continue
		}
		if step.Status == StepPending {
			if err := o.dispatch(ctx, saga, step, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *SagaOrchestrator) advanceCompensation(ctx context.Context, saga *Saga) *pkg.Error {
	phase, ok := compensationPhase(saga.StepList)
	if !ok {
		saga.Status = StatusFailed
		return o.sagaRepository.Update(saga)
	}

	for i := range saga.StepList {
		step := &saga.StepList[i]
		if step.Phase != phase {
			continue
		}
		if step.Status != StepSucceeded {
			continue
		}
		if step.Compensation == nil {
			step.Status = StepCompensated
			if err := o.stepRepository.Update(step); err != nil {
				return err
			}
			return o.advanceCompensation(ctx, saga)
		}
		if err := o.dispatch(ctx, saga, step, true); err != nil {
			return err
		}
	}
	return nil
}

func (o *SagaOrchestrator) dispatch(ctx context.Context, saga *Saga, step *SagaStep, compensation bool) *pkg.Error {
	message := step.Command
	if compensation {
		message = *step.Compensation
		step.Status = StepCompensating
	} else {
		step.Status = StepDispatched
	}
	message.SagaID = saga.ID
	message.StepID = step.ID
	message.OrderID = saga.OrderID

	if err := o.stepRepository.Update(step); err != nil {
		return err
	}
	if err := o.publisher.Publish(ctx, message); err != nil {
		return err
	}
	return nil
}

func invalidSagaError(message string) *pkg.Error {
	return pkg.NewError("INVALID_SAGA", message, nil)
}

func activePhase(steps []SagaStep, statuses ...StepStatus) (int, bool) {
	phase := 0
	found := false
	for _, step := range steps {
		if !hasStatus(step.Status, statuses) {
			continue
		}
		if !found || step.Phase < phase {
			phase = step.Phase
			found = true
		}
	}
	return phase, found
}

func compensationPhase(steps []SagaStep) (int, bool) {
	phase := 0
	found := false
	for _, step := range steps {
		if !hasStatus(step.Status, []StepStatus{StepSucceeded, StepCompensating}) {
			continue
		}
		if !found || step.Phase > phase {
			phase = step.Phase
			found = true
		}
	}
	return phase, found
}

func hasStatus(status StepStatus, statuses []StepStatus) bool {
	for _, candidate := range statuses {
		if status == candidate {
			return true
		}
	}
	return false
}
