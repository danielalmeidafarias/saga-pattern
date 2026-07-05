// Saga v1
// Nesse modelo, o saga se torna mais especifico do usecase, ligando logica com a orchestracao da operacao
// Talvez seja o mais realista, pois funciona em sistemas distribuidos de todos os tipos
// Porem, mais complexo, passivel de erros e menos reutilizavel, visto que a logica do usecase e misturada com a logica de orchestracao do saga

package main

import (
	"fmt"
)

type CreateOrderUseCasev1 struct {
	orderService     IOderServiceV1
	productService   IProductService
	inventoryService IInventoryService
	paymentService   IPaymentServiceV1
	shippingService  IShippingServiceV1
}
type CreateOrderInputv1 struct {
	Products      []OrderProduct
	InventoryUUID string
}

func NewCreateOrderUseCasev1(
	orderService IOderServiceV1,
	productService IProductService,
	inventoryService IInventoryService,
	paymentService IPaymentServiceV1,
	shippingService IShippingServiceV1,
) IUseCase[CreateOrderInputv1] {
	return CreateOrderUseCasev1{
		orderService:     orderService,
		productService:   productService,
		inventoryService: inventoryService,
		paymentService:   paymentService,
		shippingService:  shippingService,
	}
}

func (uc CreateOrderUseCasev1) Run(in CreateOrderInputv1) error {
	for _, p := range in.Products {
		product, err := uc.productService.Get(p.Product.UUID)
		if err != nil {
			return err
		}

		inventory, err := uc.inventoryService.Get(in.InventoryUUID)
		if err != nil {
			return err
		}

		if inventory.VirtualStock < p.Quantity {
			return fmt.Errorf("Requested product quantity not available for product: %s", product.Name)
		}
	}

	var amount float64
	for _, p := range in.Products {
		amount += float64(p.Quantity) * p.Product.Price
	}

	sagaOrchestrator := NewCreateOrderSagaOrchestrator(
		uc.orderService,
		uc.inventoryService,
		uc.paymentService,
		uc.shippingService,
	)

	err := sagaOrchestrator.Run(CreateOrderSagaOrchestratorInput{
		Amount:    amount,
		Inventory: Inventory{UUID: in.InventoryUUID},
		Products:  in.Products,
	})
	if err != nil {
		sagaOrchestrator.RollBack()
		return err
	}

	return nil
}

type CreateOrderSagaOrchestrator struct {
	RollBackOrchestrator
	orderService     IOderServiceV1
	inventoryService IInventoryService
	paymentService   IPaymentServiceV1
	shippingService  IShippingServiceV1
}

func NewCreateOrderSagaOrchestrator(
	orderService IOderServiceV1,
	inventoryService IInventoryService,
	paymentService IPaymentServiceV1,
	shippingService IShippingServiceV1,
) *CreateOrderSagaOrchestrator {
	return &CreateOrderSagaOrchestrator{
		orderService:     orderService,
		inventoryService: inventoryService,
		paymentService:   paymentService,
		shippingService:  shippingService,
	}
}

type CreateOrderSagaOrchestratorInput struct {
	Amount    float64
	Inventory Inventory
	Products  []OrderProduct
}

func (sg *CreateOrderSagaOrchestrator) Run(in CreateOrderSagaOrchestratorInput) error {
	for _, p := range in.Products {
		err := sg.inventoryService.Update(in.Inventory.UUID, in.Inventory.Stock, in.Inventory.VirtualStock-p.Quantity)
		if err != nil {
			sg.RollbackStepList = append(sg.RollbackStepList, NewRollBackStep(func() error {
				return sg.inventoryService.Update(in.Inventory.UUID, in.Inventory.Stock, in.Inventory.VirtualStock)
			}))
			return err
		}
	}

	paymentUUID, err := sg.paymentService.Create(in.Amount)
	if err != nil {
		sg.RollbackStepList = append(sg.RollbackStepList, NewRollBackStep(func() error {
			return sg.paymentService.Cancel(paymentUUID)
		}))
		return err
	}

	shippingUUID, err := sg.shippingService.Create(paymentUUID)
	if err != nil {
		sg.RollbackStepList = append(sg.RollbackStepList, NewRollBackStep(func() error {
			return sg.shippingService.Cancel(shippingUUID)
		}))
		return err
	}

	orderUUID, err := sg.orderService.Create(Order{
		Products:     in.Products,
		Status:       OrderPending,
		PaymentUUID:  paymentUUID,
		ShippingUUID: shippingUUID,
		Amount:       in.Amount,
	})
	if err != nil {
		sg.RollbackStepList = append(sg.RollbackStepList, NewRollBackStep(func() error {
			return sg.orderService.Cancel(orderUUID)
		}))
		return err
	}

	return nil
}
