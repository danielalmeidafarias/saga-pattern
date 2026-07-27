package shipping

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/danielalmeidafarias/saga-pattern/pkg/db"
)

func TestServiceCRUD(t *testing.T) {
	database, err := db.OpenSQLite(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	service, err := NewService(database, "")
	if err != nil {
		t.Fatal(err)
	}
	shipping := Shipping{ID: "1", UUID: "shipping-1", OrderUUID: "order-1", Status: Pending}
	if err := service.Create(shipping); err != nil {
		t.Fatal(err)
	}
	if err := service.Deliver(shipping.UUID); err != nil {
		t.Fatal(err)
	}
	stored, err := service.Get(shipping.UUID)
	if err != nil || stored.Status != Delivered {
		t.Fatalf("stored shipping: %+v, err: %v", stored, err)
	}
	if err := service.Delete(shipping.UUID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(shipping.UUID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Get after delete: %v", err)
	}
}
