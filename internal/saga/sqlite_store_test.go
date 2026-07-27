package saga

import (
	"testing"

	"github.com/danielalmeidafarias/saga-pattern/pkg/contracts"
	"github.com/danielalmeidafarias/saga-pattern/pkg/db"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

func TestSQLiteStorePersistsSagaResultsAndOutbox(t *testing.T) {
	database, err := db.OpenSQLite(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store, err := NewSQLiteStore(database)
	if err != nil {
		t.Fatal(err)
	}
	saga := &Saga{ID: "saga-1", OrderID: "order-1", Trigger: contracts.OrderCreated, Status: StatusRunning, StepList: []SagaStep{{
		ID: "step-1", SagaID: "saga-1", Phase: 0, Status: StepPending,
		Command: msg.NewMessage(contracts.TopicInventoryCommands, contracts.ReserveInventoryRequested, nil),
	}}}
	if saveErr := store.Save(saga); saveErr != nil {
		t.Fatal(saveErr.Message)
	}
	stored, findErr := store.FindByID("saga-1")
	if findErr != nil || len(stored.StepList) != 1 {
		t.Fatalf("stored saga: %+v, err: %+v", stored, findErr)
	}
	step := &stored.StepList[0]
	step.Status, step.Result = StepSucceeded, []byte(`{"uuid":"reservation-1"}`)
	if updateErr := store.UpdateResult(stored, step); updateErr != nil {
		t.Fatal(updateErr.Message)
	}
	message := msg.NewMessage(contracts.TopicPaymentCommands, contracts.CreatePaymentRequested, nil)
	message.ID = "outbox-1"
	if enqueueErr := store.Enqueue(message); enqueueErr != nil {
		t.Fatal(enqueueErr.Message)
	}
	if enqueueErr := store.Enqueue(message); enqueueErr != nil {
		t.Fatal(enqueueErr.Message)
	}
	pending, pendingErr := store.Pending(10)
	if pendingErr != nil || len(pending) != 1 {
		t.Fatalf("pending: %+v, err: %+v", pending, pendingErr)
	}
	if markErr := store.MarkPublished(message.ID); markErr != nil {
		t.Fatal(markErr.Message)
	}
	pending, pendingErr = store.Pending(10)
	if pendingErr != nil || len(pending) != 0 {
		t.Fatalf("published pending: %+v, err: %+v", pending, pendingErr)
	}
}
