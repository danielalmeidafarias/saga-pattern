package main

import (
	"errors"
	"testing"
)

func TestCreateOrderUseCasev1_Run_FailsAndRollsBack(t *testing.T) {
	tests := []struct {
		name      string
		failAt    string
		expectErr bool
		check     func(t *testing.T, deps *testDepsV1)
	}{
		{
			name:      "inventory-update",
			failAt:    "inventory-update",
			expectErr: true,
			check: func(t *testing.T, deps *testDepsV1) {
				t.Helper()
				if got := deps.inventory.items["inventory-1"]["product-1"].VirtualStock; got != 10 {
					t.Fatalf("inventory should stay intact: got %d want %d", got, 10)
				}
				if len(deps.payment.created) != 0 || len(deps.payment.canceled) != 0 {
					t.Fatalf("payment should not run: created=%d canceled=%d", len(deps.payment.created), len(deps.payment.canceled))
				}
				if len(deps.shipping.created) != 0 || len(deps.order.orders) != 0 {
					t.Fatalf("later steps should not run: shipping=%d order=%d", len(deps.shipping.created), len(deps.order.orders))
				}
			},
		},
		{
			name:      "payment",
			failAt:    "payment-create",
			expectErr: true,
			check: func(t *testing.T, deps *testDepsV1) {
				t.Helper()
				if got := deps.inventory.items["inventory-1"]["product-1"].VirtualStock; got != 10 {
					t.Fatalf("inventory not rolled back: got %d want %d", got, 10)
				}
				if len(deps.payment.created) != 0 || len(deps.payment.canceled) != 0 {
					t.Fatalf("payment should not persist on create failure: created=%d canceled=%d", len(deps.payment.created), len(deps.payment.canceled))
				}
				if len(deps.shipping.created) != 0 || len(deps.order.orders) != 0 {
					t.Fatalf("later steps should not run: shipping=%d order=%d", len(deps.shipping.created), len(deps.order.orders))
				}
			},
		},
		{
			name:      "shipping",
			failAt:    "shipping-create",
			expectErr: true,
			check: func(t *testing.T, deps *testDepsV1) {
				t.Helper()
				if got := deps.inventory.items["inventory-1"]["product-1"].VirtualStock; got != 10 {
					t.Fatalf("inventory not rolled back: got %d want %d", got, 10)
				}
				if len(deps.payment.created) != 1 || len(deps.payment.canceled) != 1 {
					t.Fatalf("payment rollback not called: created=%d canceled=%d", len(deps.payment.created), len(deps.payment.canceled))
				}
				if len(deps.shipping.created) != 0 || len(deps.shipping.canceled) != 0 {
					t.Fatalf("shipping should not be persisted on failure: created=%d canceled=%d", len(deps.shipping.created), len(deps.shipping.canceled))
				}
				if len(deps.order.orders) != 0 {
					t.Fatalf("order should not be created: got %d want 0", len(deps.order.orders))
				}
			},
		},
		{
			name:      "order",
			failAt:    "order-create",
			expectErr: true,
			check: func(t *testing.T, deps *testDepsV1) {
				t.Helper()
				if got := deps.inventory.items["inventory-1"]["product-1"].VirtualStock; got != 10 {
					t.Fatalf("inventory not rolled back: got %d want %d", got, 10)
				}
				if len(deps.payment.created) != 1 || len(deps.payment.canceled) != 1 {
					t.Fatalf("payment rollback not called: created=%d canceled=%d", len(deps.payment.created), len(deps.payment.canceled))
				}
				if len(deps.shipping.created) != 1 || len(deps.shipping.canceled) != 1 {
					t.Fatalf("shipping rollback not called: created=%d canceled=%d", len(deps.shipping.created), len(deps.shipping.canceled))
				}
				if len(deps.order.orders) != 0 {
					t.Fatalf("order should not be created: got %d want 0", len(deps.order.orders))
				}
			},
		},
		{
			name:      "success",
			failAt:    "",
			expectErr: false,
			check: func(t *testing.T, deps *testDepsV1) {
				t.Helper()
				if got := deps.inventory.items["inventory-1"]["product-1"].VirtualStock; got != 8 {
					t.Fatalf("inventory should be decremented: got %d want %d", got, 8)
				}
				if len(deps.payment.created) != 1 || len(deps.payment.canceled) != 0 {
					t.Fatalf("payment should be created only: created=%d canceled=%d", len(deps.payment.created), len(deps.payment.canceled))
				}
				if len(deps.shipping.created) != 1 || len(deps.shipping.canceled) != 0 {
					t.Fatalf("shipping should be created only: created=%d canceled=%d", len(deps.shipping.created), len(deps.shipping.canceled))
				}
				if len(deps.order.orders) != 1 {
					t.Fatalf("order should be created: got %d want 1", len(deps.order.orders))
				}
				for _, order := range deps.order.orders {
					if order.Status != OrderPending {
						t.Fatalf("order status: got %v want %v", order.Status, OrderPending)
					}
					if order.Amount != 25 {
						t.Fatalf("order amount: got %v want %v", order.Amount, 25)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := newTestDepsV1(tt.failAt)
			err := deps.uc.Run(CreateOrderInputv1{
				Products:      []OrderProduct{{Product: deps.product, Quantity: 2}},
				InventoryUUID: "inventory-1",
			})
			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, deps)
		})
	}
}

func TestCreateOrderUseCasev1_Run_ValidatesInventorySpecificProductStock(t *testing.T) {
	product := Product{UUID: "product-1", Name: "Coffee", Price: 12.5}
	deps := &testDepsV1{
		product: product,
		inventory: &memoryInventoryService{
			inventories: map[string]Inventory{
				"inventory-1": {UUID: "inventory-1"},
				"inventory-2": {UUID: "inventory-2"},
			},
			items: map[string]map[string]InventoryProduct{
				"inventory-1": {
					"product-1": {InventoryUUID: "inventory-1", ProductUUID: "product-1", Stock: 1, VirtualStock: 1},
				},
				"inventory-2": {
					"product-1": {InventoryUUID: "inventory-2", ProductUUID: "product-1", Stock: 10, VirtualStock: 10},
				},
			},
		},
		payment:  &memoryPaymentServiceV1{},
		shipping: &memoryShippingServiceV1{created: map[string]string{}},
		order:    &memoryOrderServiceV1{orders: map[string]Order{}},
	}
	deps.uc = NewCreateOrderUseCasev1(
		deps.order,
		&memoryProductService{products: map[string]Product{"product-1": product}},
		deps.inventory,
		deps.payment,
		deps.shipping,
	)

	err := deps.uc.Run(CreateOrderInputv1{
		Products:      []OrderProduct{{Product: deps.product, Quantity: 2}},
		InventoryUUID: "inventory-1",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if got := deps.inventory.items["inventory-1"]["product-1"].VirtualStock; got != 1 {
		t.Fatalf("inventory-1 stock should stay intact: got %d want %d", got, 1)
	}
	if got := deps.inventory.items["inventory-2"]["product-1"].VirtualStock; got != 10 {
		t.Fatalf("inventory-2 stock should stay intact: got %d want %d", got, 10)
	}
	if len(deps.payment.created) != 0 || len(deps.shipping.created) != 0 || len(deps.order.orders) != 0 {
		t.Fatalf("later steps should not run: payment=%d shipping=%d order=%d", len(deps.payment.created), len(deps.shipping.created), len(deps.order.orders))
	}
}

type testDepsV1 struct {
	uc        IUseCase[CreateOrderInputv1]
	product   Product
	inventory *memoryInventoryService
	payment   *memoryPaymentServiceV1
	shipping  *memoryShippingServiceV1
	order     *memoryOrderServiceV1
}

func newTestDepsV1(failAt string) *testDepsV1 {
	product := Product{UUID: "product-1", Name: "Coffee", Price: 12.5}
	deps := &testDepsV1{
		product: product,
		inventory: &memoryInventoryService{
			failAt: failAt,
			inventories: map[string]Inventory{
				"inventory-1": {UUID: "inventory-1"},
			},
			items: map[string]map[string]InventoryProduct{
				"inventory-1": {
					"product-1": {InventoryUUID: "inventory-1", ProductUUID: "product-1", Stock: 10, VirtualStock: 10},
				},
			},
		},
		payment:  &memoryPaymentServiceV1{failAt: failAt},
		shipping: &memoryShippingServiceV1{failAt: failAt, created: map[string]string{}},
		order:    &memoryOrderServiceV1{failAt: failAt, orders: map[string]Order{}},
	}
	deps.uc = NewCreateOrderUseCasev1(
		deps.order,
		&memoryProductService{failAt: failAt, products: map[string]Product{"product-1": product}},
		deps.inventory,
		deps.payment,
		deps.shipping,
	)
	return deps
}

type memoryPaymentServiceV1 struct {
	failAt   string
	created  []float64
	canceled []string
}

func (m *memoryPaymentServiceV1) Create(amount float64) (string, error) {
	if m.failAt == "payment-create" {
		return "", errors.New("payment create failed")
	}
	paymentUUID := "payment-1"
	m.created = append(m.created, amount)
	return paymentUUID, nil
}

func (m *memoryPaymentServiceV1) Get(paymentUUID string) (Payment, error) {
	return Payment{}, errors.New("payment not found")
}
func (m *memoryPaymentServiceV1) Process(paymentUUID string) error { return nil }
func (m *memoryPaymentServiceV1) Cancel(paymentUUID string) error {
	m.canceled = append(m.canceled, paymentUUID)
	return nil
}
func (m *memoryPaymentServiceV1) Refund(paymentUUID string) error { return nil }

type memoryShippingServiceV1 struct {
	failAt   string
	created  map[string]string
	canceled []string
}

func (m *memoryShippingServiceV1) Create(orderUUID string) (string, error) {
	if m.failAt == "shipping-create" {
		return "", errors.New("shipping create failed")
	}
	shippingUUID := "shipping-1"
	m.created[shippingUUID] = orderUUID
	return shippingUUID, nil
}

func (m *memoryShippingServiceV1) Get(shippingUUID string) (Shipping, error) {
	return Shipping{}, errors.New("shipping not found")
}
func (m *memoryShippingServiceV1) Start(shippingUUID string) error   { return nil }
func (m *memoryShippingServiceV1) Deliver(shippingUUID string) error { return nil }
func (m *memoryShippingServiceV1) Cancel(shippingUUID string) error {
	m.canceled = append(m.canceled, shippingUUID)
	return nil
}

type memoryOrderServiceV1 struct {
	failAt string
	orders map[string]Order
}

func (m *memoryOrderServiceV1) Create(order Order) (string, error) {
	if m.failAt == "order-create" {
		return "", errors.New("order create failed")
	}
	orderUUID := "order-1"
	order.UUID = orderUUID
	m.orders[orderUUID] = order
	return orderUUID, nil
}

func (m *memoryOrderServiceV1) Get(orderUUID string) (Order, error) {
	order, ok := m.orders[orderUUID]
	if !ok {
		return Order{}, errors.New("order not found")
	}
	return order, nil
}

func (m *memoryOrderServiceV1) Confirm(orderUUID string) error { return nil }
func (m *memoryOrderServiceV1) Cancel(orderUUID string) error {
	delete(m.orders, orderUUID)
	return nil
}
