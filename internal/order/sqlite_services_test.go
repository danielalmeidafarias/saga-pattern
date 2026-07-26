package order

import (
	"database/sql"
	"errors"
	"testing"
)

func TestSQLiteServicesPersistAndUpdateEntities(t *testing.T) {
	db, err := OpenSQLite(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	productService := NewSQLiteProductService(db, nil)
	inventoryService := NewSQLiteInventoryService(db, nil)
	paymentService := NewSQLitePaymentService(db, nil)
	shippingService := NewSQLiteShippingService(db, nil)
	orderService := NewSQLiteOrderService(db, nil)

	if err := productService.Create("product-1"); err != nil {
		t.Fatal(err)
	}
	product := Product{Id: "product-1", UUID: "product-1", Name: "Coffee", Price: 12.5}
	if err := productService.Update(product); err != nil {
		t.Fatal(err)
	}

	if err := inventoryService.Create("inventory-1"); err != nil {
		t.Fatal(err)
	}
	if err := inventoryService.AddProduct("inventory-1", "product-1", 10, 10); err != nil {
		t.Fatal(err)
	}
	if err := inventoryService.UpdateProduct("inventory-1", "product-1", 10, 8); err != nil {
		t.Fatal(err)
	}

	if err := paymentService.Create("payment-1", 25); err != nil {
		t.Fatal(err)
	}
	if err := paymentService.Process("payment-1"); err != nil {
		t.Fatal(err)
	}
	if err := shippingService.Create("shipping-1", "order-1"); err != nil {
		t.Fatal(err)
	}
	if err := shippingService.Start("shipping-1"); err != nil {
		t.Fatal(err)
	}

	if err := orderService.Create(Order{
		UUID:         "order-1",
		PaymentUUID:  "payment-1",
		ShippingUUID: "shipping-1",
		Amount:       25,
		Products:     []OrderProduct{{Product: product, Quantity: 2}},
	}); err != nil {
		t.Fatal(err)
	}

	storedOrder, err := orderService.Get("order-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(storedOrder.Products) != 1 || storedOrder.Products[0].Quantity != 2 {
		t.Fatalf("stored order products: %+v", storedOrder.Products)
	}
	storedInventoryProduct, err := inventoryService.GetProduct("inventory-1", "product-1")
	if err != nil {
		t.Fatal(err)
	}
	if storedInventoryProduct.VirtualStock != 8 {
		t.Fatalf("virtual stock: got %d want 8", storedInventoryProduct.VirtualStock)
	}
	storedPayment, err := paymentService.Get("payment-1")
	if err != nil {
		t.Fatal(err)
	}
	if storedPayment.Status != PaymentSuccess {
		t.Fatalf("payment status: got %v want %v", storedPayment.Status, PaymentSuccess)
	}
	storedShipping, err := shippingService.Get("shipping-1")
	if err != nil {
		t.Fatal(err)
	}
	if storedShipping.Status != ShippingStarted {
		t.Fatalf("shipping status: got %v want %v", storedShipping.Status, ShippingStarted)
	}

	if err := orderService.Confirm("order-1"); err != nil {
		t.Fatal(err)
	}
	storedOrder, err = orderService.Get("order-1")
	if err != nil {
		t.Fatal(err)
	}
	if storedOrder.Status != OrderConfirmed {
		t.Fatalf("order status: got %v want %v", storedOrder.Status, OrderConfirmed)
	}
}

func TestSQLiteServiceCanInjectFailure(t *testing.T) {
	db, err := OpenSQLite(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	failure := errors.New("payment unavailable")
	service := NewSQLitePaymentService(db, &FailureInjector{
		Operation: PaymentCreateOperation,
		Err:       failure,
	})

	if err := service.Create("payment-1", 10); !errors.Is(err, failure) {
		t.Fatalf("Create error: got %v want %v", err, failure)
	}
	if _, err := service.Get("payment-1"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("failed create should not persist payment: %v", err)
	}
}
