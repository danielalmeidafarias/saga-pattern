package payment

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/danielalmeidafarias/saga-pattern/pkg/db"
)

func TestServiceCRUDAndFailure(t *testing.T) {
	database, err := db.OpenSQLite(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	service, err := NewService(database, "")
	if err != nil {
		t.Fatal(err)
	}
	payment := Payment{ID: "1", UUID: "payment-1", Amount: 20, Method: Pix, Status: Pending}
	if err := service.Create(payment); err != nil {
		t.Fatal(err)
	}
	if err := service.Process(payment.UUID); err != nil {
		t.Fatal(err)
	}
	stored, err := service.Get(payment.UUID)
	if err != nil || stored.Status != Succeeded {
		t.Fatalf("stored payment: %+v, err: %v", stored, err)
	}
	if err := service.Delete(payment.UUID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(payment.UUID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Get after delete: %v", err)
	}

	failing, err := NewService(database, CreateOperation)
	if err != nil {
		t.Fatal(err)
	}
	if err := failing.Create(payment); err == nil {
		t.Fatal("expected injected failure")
	}
}
