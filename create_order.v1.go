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
	if _, err := uc.inventoryService.Get(in.InventoryUUID); err != nil {
		return err
	}

	requestedQuantities := map[string]int{}
	for _, p := range in.Products {
		requestedQuantities[p.Product.UUID] += p.Quantity
	}

	inventoryProducts := map[string]InventoryProduct{}
	validatedProducts := map[string]struct{}{}
	for _, p := range in.Products {
		if _, ok := validatedProducts[p.Product.UUID]; ok {
			continue
		}
		validatedProducts[p.Product.UUID] = struct{}{}

		product, err := uc.productService.Get(p.Product.UUID)
		if err != nil {
			return err
		}

		inventoryProduct, err := uc.inventoryService.GetProduct(in.InventoryUUID, p.Product.UUID)
		if err != nil {
			return err
		}

		if inventoryProduct.VirtualStock < requestedQuantities[p.Product.UUID] {
			return fmt.Errorf("Requested product quantity not available for product: %s", product.Name)
		}

		inventoryProducts[p.Product.UUID] = inventoryProduct
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
		Amount:            amount,
		InventoryUUID:     in.InventoryUUID,
		InventoryProducts: inventoryProducts,
		Products:          in.Products,
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
	Amount            float64
	InventoryUUID     string
	InventoryProducts map[string]InventoryProduct
	Products          []OrderProduct
}

func (sg *CreateOrderSagaOrchestrator) Run(in CreateOrderSagaOrchestratorInput) error {
	requestedQuantities := map[string]int{}
	for _, p := range in.Products {
		requestedQuantities[p.Product.UUID] += p.Quantity
	}

	updatedProducts := map[string]struct{}{}
	for _, p := range in.Products {
		if _, ok := updatedProducts[p.Product.UUID]; ok {
			continue
		}
		updatedProducts[p.Product.UUID] = struct{}{}

		inventoryProduct := in.InventoryProducts[p.Product.UUID]
		requestedQuantity := requestedQuantities[p.Product.UUID]
		err := sg.inventoryService.UpdateProduct(
			in.InventoryUUID,
			inventoryProduct.ProductUUID,
			inventoryProduct.Stock,
			inventoryProduct.VirtualStock-requestedQuantity,
		)
		if err != nil {
			return err
		}

		currentInventoryProduct := inventoryProduct
		sg.RollbackStepList = append(sg.RollbackStepList, NewRollBackStep(func() error {
			return sg.inventoryService.UpdateProduct(
				in.InventoryUUID,
				currentInventoryProduct.ProductUUID,
				currentInventoryProduct.Stock,
				currentInventoryProduct.VirtualStock,
			)
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
