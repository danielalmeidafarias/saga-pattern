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
	var inventoryProductList = make(map[OrderProduct]InventoryProduct)
	for _, p := range in.Products {
		inventoryProduct, err := uc.inventoryService.GetProduct(in.InventoryUUID, p.Product.UUID)
		if err != nil {
			return err
		}

		if inventoryProduct.VirtualStock < p.Quantity {
			return fmt.Errorf("Requested product quantity not available for product: %s", p.Product.Name)
		}

		inventoryProductList[p] = inventoryProduct
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
		Amount:              amount,
		InventoryProductMap: inventoryProductList,
		Products:            in.Products,
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
	Amount              float64
	InventoryProductMap map[OrderProduct]InventoryProduct
	Products            []OrderProduct
}

func (sg *CreateOrderSagaOrchestrator) Run(in CreateOrderSagaOrchestratorInput) error {
	for p, i := range in.InventoryProductMap {

		err := sg.inventoryService.UpdateProduct(i.InventoryUUID, i.ProductUUID, i.Stock, i.VirtualStock-p.Quantity)
		if err != nil {
			return err
		}

		sg.RollbackStepList = append(sg.RollbackStepList, NewRollBackStep(func() error {
			return sg.inventoryService.UpdateProduct(i.InventoryUUID, i.ProductUUID, i.Stock, i.VirtualStock)
		}))
	}

	paymentUUID, err := sg.paymentService.Create(in.Amount)
	if err != nil {
		return err
	}

	sg.RollbackStepList = append(sg.RollbackStepList, NewRollBackStep(func() error {
		return sg.paymentService.Cancel(paymentUUID)
	}))

	shippingUUID, err := sg.shippingService.Create(paymentUUID)
	if err != nil {
		return err
	}

	sg.RollbackStepList = append(sg.RollbackStepList, NewRollBackStep(func() error {
		return sg.shippingService.Cancel(shippingUUID)
	}))

	orderUUID, err := sg.orderService.Create(Order{
		Products:     in.Products,
		Status:       OrderPending,
		PaymentUUID:  paymentUUID,
		ShippingUUID: shippingUUID,
		Amount:       in.Amount,
	})
	if err != nil {
		return err
	}

	sg.RollbackStepList = append(sg.RollbackStepList, NewRollBackStep(func() error {
		return sg.orderService.Cancel(orderUUID)
	}))

	return nil
}
