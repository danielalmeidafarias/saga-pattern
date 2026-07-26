package saga

import (
	"context"
	"testing"

	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

func TestSagaOrchestrator_AdvancesOnePhaseAtATime(t *testing.T) {
	store := newMemoryStore(&Saga{
		ID:      "saga-1",
		OrderID: "order-1",
		Status:  StatusRunning,
		StepList: []SagaStep{
			step("step-1", 0, msg.ReserveInventoryMessage, msg.ReleaseInventoryMessage),
			step("step-2", 1, msg.CreatePaymentMessage, msg.CancelPaymentMessage),
		},
	})
	publisher := &recordingPublisher{}
	orchestrator := NewSagaOrchestrator(&memorySagaRepository{store}, &memoryStepRepository{store}, publisher)

	if err := orchestrator.Advance(context.Background(), "saga-1"); err != nil {
		t.Fatalf("advance: %v", err)
	}
	assertPublished(t, publisher, 1, "step-1", msg.ReserveInventoryMessage)
	assertStepStatus(t, store, "step-1", StepDispatched)
	assertStepStatus(t, store, "step-2", StepPending)

	if err := orchestrator.HandleResult(context.Background(), success("saga-1", "step-1")); err != nil {
		t.Fatalf("complete first step: %v", err)
	}
	assertPublished(t, publisher, 2, "step-2", msg.CreatePaymentMessage)
	assertStepStatus(t, store, "step-1", StepSucceeded)

	if err := orchestrator.HandleResult(context.Background(), success("saga-1", "step-2")); err != nil {
		t.Fatalf("complete second step: %v", err)
	}
	if got := store.sagas["saga-1"].Status; got != StatusSucceeded {
		t.Fatalf("saga status: got %s want %s", got, StatusSucceeded)
	}
}

func TestSagaOrchestrator_CompensatesCompletedStepsInReverseOrder(t *testing.T) {
	store := newMemoryStore(&Saga{
		ID:      "saga-1",
		OrderID: "order-1",
		Status:  StatusRunning,
		StepList: []SagaStep{
			step("step-1", 0, msg.ReserveInventoryMessage, msg.ReleaseInventoryMessage),
			step("step-2", 1, msg.CreatePaymentMessage, msg.CancelPaymentMessage),
		},
	})
	publisher := &recordingPublisher{}
	orchestrator := NewSagaOrchestrator(&memorySagaRepository{store}, &memoryStepRepository{store}, publisher)

	if err := orchestrator.Advance(context.Background(), "saga-1"); err != nil {
		t.Fatalf("dispatch inventory: %v", err)
	}
	if err := orchestrator.HandleResult(context.Background(), success("saga-1", "step-1")); err != nil {
		t.Fatalf("complete inventory: %v", err)
	}
	if err := orchestrator.HandleResult(context.Background(), failure("saga-1", "step-2")); err != nil {
		t.Fatalf("fail payment: %v", err)
	}

	assertPublished(t, publisher, 3, "step-1", msg.ReleaseInventoryMessage)
	assertStepStatus(t, store, "step-1", StepCompensating)
	assertStepStatus(t, store, "step-2", StepFailed)

	if err := orchestrator.HandleResult(context.Background(), success("saga-1", "step-1")); err != nil {
		t.Fatalf("complete compensation: %v", err)
	}
	assertStepStatus(t, store, "step-1", StepCompensated)
	if got := store.sagas["saga-1"].Status; got != StatusFailed {
		t.Fatalf("saga status: got %s want %s", got, StatusFailed)
	}
}

func TestSagaOrchestrator_DispatchesEveryStepInTheActivePhase(t *testing.T) {
	store := newMemoryStore(&Saga{
		ID:      "saga-1",
		OrderID: "order-1",
		Status:  StatusRunning,
		StepList: []SagaStep{
			step("step-1", 0, msg.ReserveInventoryMessage, msg.ReleaseInventoryMessage),
			step("step-2", 0, msg.CreatePaymentMessage, msg.CancelPaymentMessage),
			step("step-3", 1, msg.CreateShippingMessage, msg.CancelShippingMessage),
		},
	})
	publisher := &recordingPublisher{}
	orchestrator := NewSagaOrchestrator(&memorySagaRepository{store}, &memoryStepRepository{store}, publisher)

	if err := orchestrator.Advance(context.Background(), "saga-1"); err != nil {
		t.Fatalf("advance first phase: %v", err)
	}
	assertPublished(t, publisher, 1, "step-1", msg.ReserveInventoryMessage)
	assertPublished(t, publisher, 2, "step-2", msg.CreatePaymentMessage)
	assertStepStatus(t, store, "step-3", StepPending)

	if err := orchestrator.HandleResult(context.Background(), success("saga-1", "step-1")); err != nil {
		t.Fatalf("complete first concurrent step: %v", err)
	}
	if len(publisher.messages) != 2 {
		t.Fatalf("phase must wait for every result: got %d messages", len(publisher.messages))
	}

	if err := orchestrator.HandleResult(context.Background(), success("saga-1", "step-2")); err != nil {
		t.Fatalf("complete second concurrent step: %v", err)
	}
	assertPublished(t, publisher, 3, "step-3", msg.CreateShippingMessage)
}

func step(id string, phase int, command, compensation msg.MessageType) SagaStep {
	compensationMessage := msg.NewMessage("compensation", compensation, nil)
	return SagaStep{
		ID:           id,
		Phase:        phase,
		Status:       StepPending,
		Command:      msg.NewMessage("commands", command, nil),
		Compensation: &compensationMessage,
	}
}

func success(sagaID, stepID string) msg.Message {
	return msg.Message{SagaID: sagaID, StepID: stepID}
}

func failure(sagaID, stepID string) msg.Message {
	return msg.Message{SagaID: sagaID, StepID: stepID, Failure: &msg.Failure{Code: "FAILED", Message: "expected"}}
}

type recordingPublisher struct {
	messages []msg.Message
}

func (p *recordingPublisher) Publish(_ context.Context, message msg.Message) *pkg.Error {
	p.messages = append(p.messages, message)
	return nil
}

type memoryStore struct {
	sagas map[string]*Saga
	steps map[string]*SagaStep
}

func newMemoryStore(saga *Saga) *memoryStore {
	store := &memoryStore{
		sagas: map[string]*Saga{saga.ID: saga},
		steps: make(map[string]*SagaStep, len(saga.StepList)),
	}
	for i := range saga.StepList {
		saga.StepList[i].SagaID = saga.ID
		store.steps[saga.StepList[i].ID] = &saga.StepList[i]
	}
	return store
}

type memorySagaRepository struct{ store *memoryStore }

func (r *memorySagaRepository) Save(saga *Saga) *pkg.Error {
	r.store.sagas[saga.ID] = saga
	for i := range saga.StepList {
		r.store.steps[saga.StepList[i].ID] = &saga.StepList[i]
	}
	return nil
}

func (r *memorySagaRepository) FindByID(id string) (*Saga, *pkg.Error) {
	saga, ok := r.store.sagas[id]
	if !ok {
		return nil, pkg.NewError("NOT_FOUND", "saga not found", nil)
	}
	return saga, nil
}

func (r *memorySagaRepository) Update(saga *Saga) *pkg.Error {
	r.store.sagas[saga.ID] = saga
	return nil
}

func (r *memorySagaRepository) GetAll(filter GetAllSagaFilter) ([]Saga, *pkg.Error) {
	var sagas []Saga
	for _, saga := range r.store.sagas {
		if filter.Status == nil || saga.Status == *filter.Status {
			sagas = append(sagas, *saga)
		}
	}
	return sagas, nil
}

type memoryStepRepository struct{ store *memoryStore }

func (r *memoryStepRepository) FindByID(id string) (*SagaStep, *pkg.Error) {
	step, ok := r.store.steps[id]
	if !ok {
		return nil, pkg.NewError("NOT_FOUND", "step not found", nil)
	}
	return step, nil
}

func (r *memoryStepRepository) Update(step *SagaStep) *pkg.Error {
	r.store.steps[step.ID] = step
	return nil
}

func assertPublished(t *testing.T, publisher *recordingPublisher, count int, stepID string, messageType msg.MessageType) {
	t.Helper()
	if len(publisher.messages) < count {
		t.Fatalf("published messages: got %d want at least %d", len(publisher.messages), count)
	}
	message := publisher.messages[count-1]
	if message.StepID != stepID || message.Type != messageType {
		t.Fatalf("message: got step=%s type=%s want step=%s type=%s", message.StepID, message.Type, stepID, messageType)
	}
}

func assertStepStatus(t *testing.T, store *memoryStore, stepID string, want StepStatus) {
	t.Helper()
	if got := store.steps[stepID].Status; got != want {
		t.Fatalf("step %s status: got %s want %s", stepID, got, want)
	}
}
