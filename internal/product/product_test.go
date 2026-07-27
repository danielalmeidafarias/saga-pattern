package product

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
	product := Product{ID: "1", UUID: "product-1", Name: "Coffee", Price: 10}
	if err := service.Create(product); err != nil {
		t.Fatal(err)
	}
	product.Price = 12
	if err := service.Update(product); err != nil {
		t.Fatal(err)
	}
	stored, err := service.Get(product.UUID)
	if err != nil || stored.Price != 12 {
		t.Fatalf("stored product: %+v, err: %v", stored, err)
	}
	if err := service.Delete(product.UUID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(product.UUID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Get after delete: %v", err)
	}
}
