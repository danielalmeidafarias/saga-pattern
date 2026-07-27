package order

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
	order := Order{ID: "1", UUID: "order-1", PaymentUUID: "payment-1", ShippingUUID: "shipping-1", Amount: 20, Status: Pending, Items: []Item{{ProductUUID: "product-1", Name: "Coffee", Price: 10, Quantity: 2}}}
	if err := service.Create(order); err != nil {
		t.Fatal(err)
	}
	if err := service.Confirm(order.UUID); err != nil {
		t.Fatal(err)
	}
	stored, err := service.Get(order.UUID)
	if err != nil || stored.Status != Confirmed || len(stored.Items) != 1 {
		t.Fatalf("stored order: %+v, err: %v", stored, err)
	}
	if err := service.Delete(order.UUID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(order.UUID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Get after delete: %v", err)
	}
}
