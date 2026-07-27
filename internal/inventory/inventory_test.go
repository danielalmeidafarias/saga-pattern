package inventory

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
	if err := service.Create(Inventory{ID: "1", UUID: "inventory-1"}); err != nil {
		t.Fatal(err)
	}
	item := Product{InventoryUUID: "inventory-1", ProductUUID: "product-1", Stock: 10, VirtualStock: 10}
	if err := service.CreateProduct(item); err != nil {
		t.Fatal(err)
	}
	item.VirtualStock = 8
	if err := service.UpdateProduct(item); err != nil {
		t.Fatal(err)
	}
	stored, err := service.GetProduct(item.InventoryUUID, item.ProductUUID)
	if err != nil || stored.VirtualStock != 8 {
		t.Fatalf("stored inventory product: %+v, err: %v", stored, err)
	}
	if err := service.DeleteProduct(item.InventoryUUID, item.ProductUUID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetProduct(item.InventoryUUID, item.ProductUUID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetProduct after delete: %v", err)
	}
}
