package saga

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/db"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

func TestHandlerAndRecoveryUsePersistedWorkflow(t *testing.T) {
	database, err := db.OpenSQLite(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store, err := NewSQLiteStore(database)
	if err != nil {
		t.Fatal(err)
	}
	triggerType := msg.MessageType("test.started")
	resolver := NewResolver(map[msg.MessageType]ResolveFunc{
		triggerType: func(msg.Message) ([]SagaStep, *pkg.Error) {
			compensation := msg.NewMessage("commands", "test.undo", nil)
			return []SagaStep{{
				ID:           uuid.NewString(),
				Phase:        0,
				Command:      msg.NewMessage("commands", "test.do", nil),
				Compensation: &compensation,
			}}, nil
		},
	})
	create := NewCreateSagaUseCase(store, resolver)
	orchestrator := NewSagaOrchestrator(store, store, msg.NewOutboxPublisher(store))
	handler := NewHandler(create, orchestrator)
	trigger := msg.Message{ID: "trigger-1", OrderID: "order-1", Type: triggerType}
	if handleErr := handler.HandleTrigger(context.Background(), trigger); handleErr != nil {
		t.Fatal(handleErr.Message)
	}
	stored, findErr := store.FindByID(trigger.ID)
	if findErr != nil || stored.StepList[0].Status != StepDispatched {
		t.Fatalf("dispatched saga: %+v, err: %+v", stored, findErr)
	}
	if handleErr := handler.HandleResult(context.Background(), msg.Message{SagaID: stored.ID, StepID: stored.StepList[0].ID}); handleErr != nil {
		t.Fatal(handleErr.Message)
	}
	stored, findErr = store.FindByID(trigger.ID)
	if findErr != nil || stored.Status != StatusSucceeded {
		t.Fatalf("completed saga: %+v, err: %+v", stored, findErr)
	}

	second := msg.Message{ID: "trigger-2", OrderID: "order-2", Type: triggerType}
	if createErr := create.Run(context.Background(), CreateSagaUseCaseInput{Message: second}); createErr != nil {
		t.Fatal(createErr.Message)
	}
	recovery := NewProcessSagaUseCase(store, orchestrator)
	if recoveryErr := recovery.Run(context.Background(), ProcessSagaUseCaseInput{Status: StatusRunning, BatchSize: 10}); recoveryErr != nil {
		t.Fatal(recoveryErr.Message)
	}
	secondSaga, findErr := store.FindByID(second.ID)
	if findErr != nil || secondSaga.StepList[0].Status != StepDispatched {
		t.Fatalf("recovered saga: %+v, err: %+v", secondSaga, findErr)
	}
}
