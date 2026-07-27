package saga

import (
	"context"
	"encoding/json"

	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/contracts"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

type SagaOrchestrator struct {
	sagaRepository SagaRepository
	stepRepository SagaStepRepository
	publisher      msg.Publisher
}

func NewSagaOrchestrator(sagaRepository SagaRepository, stepRepository SagaStepRepository, publisher msg.Publisher) *SagaOrchestrator {
	return &SagaOrchestrator{sagaRepository: sagaRepository, stepRepository: stepRepository, publisher: publisher}
}

// Advance queues every pending command in the active phase.
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

// HandleResult persists the result before advancing the workflow.
func (o *SagaOrchestrator) HandleResult(ctx context.Context, result msg.Message) *pkg.Error {
	if result.SagaID == "" || result.StepID == "" {
		return invalidSagaError("result must contain sagaId and stepId")
	}
	saga, err := o.sagaRepository.FindByID(result.SagaID)
	if err != nil {
		return err
	}
	step, err := o.stepRepository.FindStepByID(result.StepID)
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
			step.Result = append(step.Result[:0], result.Payload...)
		}
	case StepCompensating:
		if result.Failure != nil {
			step.Status = StepFailed
			saga.Status = StatusFailed
		} else {
			step.Status = StepCompensated
		}
	case StepSucceeded, StepCompensated, StepFailed:
		return o.Advance(ctx, saga.ID)
	default:
		return invalidSagaError("result received for a step that was not dispatched")
	}

	if err := o.sagaRepository.UpdateResult(saga, step); err != nil {
		return err
	}
	return o.Advance(ctx, saga.ID)
}

func (o *SagaOrchestrator) advanceForward(ctx context.Context, saga *Saga) *pkg.Error {
	phase, ok := activePhase(saga.StepList, StepPending, StepDispatched)
	if !ok {
		if saga.Trigger == contracts.OrderCreated {
			message, err := orderSucceededMessage(saga)
			if err != nil {
				return err
			}
			if err := o.publisher.Publish(ctx, message); err != nil {
				return err
			}
		}
		saga.Status = StatusSucceeded
		return o.sagaRepository.Update(saga)
	}

	for i := range saga.StepList {
		step := &saga.StepList[i]
		if step.Phase == phase && step.Status == StepPending {
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
		if saga.Trigger == contracts.OrderCreated {
			message := msg.NewMessage(contracts.TopicOrderEvents, contracts.OrderCreationFailed, nil)
			message.ID = saga.ID + ":completion"
			message.OrderID = saga.OrderID
			if err := o.publisher.Publish(ctx, message); err != nil {
				return err
			}
		}
		saga.Status = StatusFailed
		return o.sagaRepository.Update(saga)
	}

	for i := range saga.StepList {
		step := &saga.StepList[i]
		if step.Phase != phase || step.Status != StepSucceeded {
			continue
		}
		if step.Compensation == nil {
			step.Status = StepCompensated
			if err := o.stepRepository.UpdateStep(step); err != nil {
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
	suffix := "forward"
	if compensation {
		message = *step.Compensation
		suffix = "compensation"
	}
	message.ID = saga.ID + ":" + step.ID + ":" + suffix
	message.SagaID = saga.ID
	message.StepID = step.ID
	message.OrderID = saga.OrderID
	if err := o.publisher.Publish(ctx, message); err != nil {
		return err
	}
	if compensation {
		step.Status = StepCompensating
	} else {
		step.Status = StepDispatched
	}
	return o.stepRepository.UpdateStep(step)
}

func orderSucceededMessage(saga *Saga) (msg.Message, *pkg.Error) {
	payment, err := resultForCommand[contracts.ResourceCreated](saga.StepList, contracts.CreatePaymentRequested)
	if err != nil {
		return msg.Message{}, err
	}
	shipping, err := resultForCommand[contracts.ResourceCreated](saga.StepList, contracts.CreateShippingRequested)
	if err != nil {
		return msg.Message{}, err
	}
	payload, marshalErr := json.Marshal(contracts.OrderCreationSucceededPayload{PaymentUUID: payment.UUID, ShippingUUID: shipping.UUID})
	if marshalErr != nil {
		return msg.Message{}, pkg.NewError("SERIALIZATION_ERROR", "serialize order creation result", marshalErr)
	}
	message := msg.NewMessage(contracts.TopicOrderEvents, contracts.OrderCreationSucceeded, payload)
	message.ID = saga.ID + ":completion"
	message.OrderID = saga.OrderID
	return message, nil
}

func resultForCommand[T any](steps []SagaStep, command msg.MessageType) (T, *pkg.Error) {
	var result T
	for _, step := range steps {
		if step.Command.Type != command {
			continue
		}
		if len(step.Result) == 0 {
			return result, invalidSagaError("successful step is missing a result")
		}
		if err := json.Unmarshal(step.Result, &result); err != nil {
			return result, pkg.NewError("INVALID_SAGA", "invalid successful step result", err)
		}
		return result, nil
	}
	return result, invalidSagaError("saga is missing a required step")
}

func invalidSagaError(message string) *pkg.Error {
	return pkg.NewError("INVALID_SAGA", message, nil)
}

func activePhase(steps []SagaStep, statuses ...StepStatus) (int, bool) {
	phase, found := 0, false
	for _, step := range steps {
		if hasStatus(step.Status, statuses) && (!found || step.Phase < phase) {
			phase, found = step.Phase, true
		}
	}
	return phase, found
}

func compensationPhase(steps []SagaStep) (int, bool) {
	phase, found := 0, false
	for _, step := range steps {
		if hasStatus(step.Status, []StepStatus{StepSucceeded, StepCompensating}) && (!found || step.Phase > phase) {
			phase, found = step.Phase, true
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
