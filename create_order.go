package main

import (
	"fmt"

	"github.com/google/uuid"
)

type CreateOrderUseCase struct {
	orderService     IOderService
	productService   IProductService
	inventoryService IInventoryService
	paymentService   IPaymentService
	shippingService  IShippingService
}
type CreateOrderInput struct {
	Products      []OrderProduct
	InventoryUUID string
}

func NewCreateOrderUseCase(
	orderService IOderService,
	productService IProductService,
	inventoryService IInventoryService,
	paymentService IPaymentService,
	shippingService IShippingService,
) IUseCase[CreateOrderInput] {
	createOrderUseCase := CreateOrderUseCase{
		orderService:     orderService,
		productService:   productService,
		inventoryService: inventoryService,
		paymentService:   paymentService,
		shippingService:  shippingService,
	}

	return NewUseCaseWithSaga[CreateOrderInput](createOrderUseCase)
}

func (uc CreateOrderUseCase) runFunc(in CreateOrderInput) (*SagaOrchestrator, error) {
	sagaOrchestrator := NewSagaOrchestrator()

	for _, p := range in.Products {
		product, err := uc.productService.Get(p.Product.UUID)
		if err != nil {
			return &sagaOrchestrator, err
		}

		inventory, err := uc.inventoryService.Get(in.InventoryUUID)
		if err != nil {
			return &sagaOrchestrator, err
		}

		if inventory.VirtualStock < p.Quantity {
			return &sagaOrchestrator, fmt.Errorf("Requested product quantity not available for product: %s", product.Name)
		}

		sagaOrchestrator.AddStep(NewSagaStep(
			func() error {
				return uc.inventoryService.Update(in.InventoryUUID, inventory.Stock, inventory.VirtualStock-p.Quantity)
			},
			func() error {
				return uc.inventoryService.Update(in.InventoryUUID, inventory.Stock, inventory.VirtualStock)
			},
		))

	}

	var amount float64
	for _, p := range in.Products {
		amount += float64(p.Quantity) * p.Product.Price
	}

	paymentUUID := uuid.New().String()
	sagaOrchestrator.AddStep(
		NewSagaStep(
			func() error {
				return uc.paymentService.Create(paymentUUID, amount)
			},
			func() error {
				return uc.paymentService.Cancel(paymentUUID)
			},
		),
	)

	shippingUUID := uuid.New().String()
	orderUUID := uuid.New().String()

	sagaOrchestrator.AddStep(
		NewSagaStep(
			func() error {
				return uc.shippingService.Create(shippingUUID, orderUUID)
			},
			func() error {
				return uc.shippingService.Cancel(shippingUUID)
			},
		),
	)

	sagaOrchestrator.AddStep(NewSagaStep(
		func() error {
			return uc.orderService.Create(Order{
				UUID:         orderUUID,
				Products:     in.Products,
				Status:       OrderPending,
				PaymentUUID:  paymentUUID,
				ShippingUUID: shippingUUID,
				Amount:       amount,
			})
		},
		func() error {
			return uc.orderService.Cancel(orderUUID)
		},
	))

	return &sagaOrchestrator, sagaOrchestrator.Run()
}
