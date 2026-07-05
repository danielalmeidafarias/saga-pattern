package main

import (
	"errors"
	"testing"
)

func TestCreateOrderUseCase_Run_FailsAndRollsBack(t *testing.T) {
	tests := []struct {
		name      string
		failAt    string
		expectErr bool
		check     func(t *testing.T, deps *testDeps)
	}{
		{
			name:      "inventory-update",
			failAt:    "inventory-update",
			expectErr: true,
			check: func(t *testing.T, deps *testDeps) {
				t.Helper()
				if got := deps.inventory.items["inventory-1"].VirtualStock; got != 10 {
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
			check: func(t *testing.T, deps *testDeps) {
				t.Helper()
				if got := deps.inventory.items["inventory-1"].VirtualStock; got != 10 {
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
			check: func(t *testing.T, deps *testDeps) {
				t.Helper()
				if got := deps.inventory.items["inventory-1"].VirtualStock; got != 10 {
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
			check: func(t *testing.T, deps *testDeps) {
				t.Helper()
				if got := deps.inventory.items["inventory-1"].VirtualStock; got != 10 {
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
			check: func(t *testing.T, deps *testDeps) {
				t.Helper()
				if got := deps.inventory.items["inventory-1"].VirtualStock; got != 8 {
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
			deps := newTestDeps(tt.failAt)
			err := deps.uc.Run(CreateOrderInput{
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

type testDeps struct {
	uc        IUseCase[CreateOrderInput]
	product   Product
	inventory *memoryInventoryService
	payment   *memoryPaymentService
	shipping  *memoryShippingService
	order     *memoryOrderService
}

func newTestDeps(failAt string) *testDeps {
	product := Product{UUID: "product-1", Name: "Coffee", Price: 12.5}
	deps := &testDeps{
		product: product,
		inventory: &memoryInventoryService{failAt: failAt, items: map[string]Inventory{
			"inventory-1": {UUID: "inventory-1", Stock: 10, VirtualStock: 10},
		}},
		payment:  &memoryPaymentService{failAt: failAt, created: map[string]float64{}},
		shipping: &memoryShippingService{failAt: failAt, created: map[string]string{}},
		order:    &memoryOrderService{failAt: failAt, orders: map[string]Order{}},
	}
	deps.uc = NewCreateOrderUseCase(
		deps.order,
		&memoryProductService{failAt: failAt, products: map[string]Product{"product-1": product}},
		deps.inventory,
		deps.payment,
		deps.shipping,
	)
	return deps
}

type memoryProductService struct {
	failAt   string
	products map[string]Product
}

func (m *memoryProductService) Create(productId string) error { return nil }

func (m *memoryProductService) Get(productUUID string) (Product, error) {
	if m.failAt == "product-get" {
		return Product{}, errors.New("product not found")
	}
	product, ok := m.products[productUUID]
	if !ok {
		return Product{}, errors.New("product not found")
	}
	return product, nil
}

type memoryInventoryService struct {
	failAt string
	items  map[string]Inventory
}

func (m *memoryInventoryService) Create(productUUID string, stock int) error { return nil }

func (m *memoryInventoryService) Get(inventoryUUID string) (Inventory, error) {
	if m.failAt == "inventory-get" {
		return Inventory{}, errors.New("inventory not found")
	}
	inventory, ok := m.items[inventoryUUID]
	if !ok {
		return Inventory{}, errors.New("inventory not found")
	}
	return inventory, nil
}

func (m *memoryInventoryService) Update(inventoryUUID string, stock int, virtualStock int) error {
	if m.failAt == "inventory-update" {
		return errors.New("inventory update failed")
	}
	inventory := m.items[inventoryUUID]
	inventory.Stock = stock
	inventory.VirtualStock = virtualStock
	m.items[inventoryUUID] = inventory
	return nil
}

type memoryPaymentService struct {
	failAt   string
	created  map[string]float64
	canceled []string
}

func (m *memoryPaymentService) Create(paymentUUID string, amount float64) error {
	if m.failAt == "payment-create" {
		return errors.New("payment create failed")
	}
	m.created[paymentUUID] = amount
	return nil
}

func (m *memoryPaymentService) Get(paymentUUID string) (Payment, error) {
	return Payment{}, errors.New("payment not found")
}
func (m *memoryPaymentService) Process(paymentUUID string) error { return nil }
func (m *memoryPaymentService) Cancel(paymentUUID string) error {
	m.canceled = append(m.canceled, paymentUUID)
	return nil
}
func (m *memoryPaymentService) Refund(paymentUUID string) error { return nil }

type memoryShippingService struct {
	failAt   string
	created  map[string]string
	canceled []string
}

func (m *memoryShippingService) Create(shippingUUID, orderUUID string) error {
	if m.failAt == "shipping-create" {
		return errors.New("shipping create failed")
	}
	m.created[shippingUUID] = orderUUID
	return nil
}

func (m *memoryShippingService) Get(shippingUUID string) (Shipping, error) {
	return Shipping{}, errors.New("shipping not found")
}
func (m *memoryShippingService) Start(shippingUUID string) error   { return nil }
func (m *memoryShippingService) Deliver(shippingUUID string) error { return nil }
func (m *memoryShippingService) Cancel(shippingUUID string) error {
	m.canceled = append(m.canceled, shippingUUID)
	return nil
}

type memoryOrderService struct {
	failAt string
	orders map[string]Order
}

func (m *memoryOrderService) Create(order Order) error {
	if m.failAt == "order-create" {
		return errors.New("order create failed")
	}
	m.orders[order.UUID] = order
	return nil
}

func (m *memoryOrderService) Get(orderUUID string) (Order, error) {
	order, ok := m.orders[orderUUID]
	if !ok {
		return Order{}, errors.New("order not found")
	}
	return order, nil
}

func (m *memoryOrderService) Confirm(orderUUID string) error { return nil }
func (m *memoryOrderService) Cancel(orderUUID string) error  { delete(m.orders, orderUUID); return nil }
