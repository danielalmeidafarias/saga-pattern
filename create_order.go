package main

import "fmt"

type CrateOrderUseCase struct {
	orderService     IOderService
	productService   IProductService
	inventoryService IInventoryService
	paymentService   IPaymentService
	shippingService  IShippingService
}

type CreateOrderInputProduct struct {
	ProductId string
	Quantity  int
}

type CreateOrderInput struct {
	Products    []CreateOrderInputProduct
	InventoryId string
}

func (uc *CrateOrderUseCase) Run(in CreateOrderInput) error {
	rollbackOrchestrator := NewRollbackOrchestrator()

	var orderProducts []Product
	for _, p := range in.Products {
		product, err := uc.productService.Get(p.ProductId)
		if err != nil {
			rollbackOrchestrator.RollBack()
			return err
		}
		orderProducts = append(orderProducts, product)

		inventory, err := uc.inventoryService.Get(in.InventoryId)
		if err != nil {
			rollbackOrchestrator.RollBack()
			return err
		}

		if inventory.VirtualStock < p.Quantity {
			rollbackOrchestrator.RollBack()
			return fmt.Errorf("Requested product quantity not available for product: %s", product.Name)
		}

		err = uc.inventoryService.Update(in.InventoryId, inventory.Stock, inventory.VirtualStock-p.Quantity)
		if err != nil {
			rollbackOrchestrator.RollBack()
			return nil
		}

		rollbackOrchestrator.AddStep(NewSagaStep(
			func(args ...any) error {
				return uc.inventoryService.Update(in.InventoryId, inventory.Stock, inventory.VirtualStock)
			},
		))
	}

	var productsIds []string
	var amount float64
	for _, p := range orderProducts {
		productsIds = append(productsIds, p.Id)
		amount += p.Price
	}

	order, err := uc.orderService.Create(productsIds, amount)
	if err != nil {
		rollbackOrchestrator.RollBack()
		return err
	}
	rollbackOrchestrator.AddStep(NewSagaStep(
		func(args ...any) error {
			return uc.orderService.Cancel(order.Id)
		},
	))

	payment, err := uc.paymentService.Create(amount)
	if err != nil {
		rollbackOrchestrator.RollBack()
		return err
	}
	rollbackOrchestrator.AddStep(NewSagaStep(
		func(args ...any) error {
			return uc.paymentService.Cancel(payment.Id)
		},
	))

	_, err = uc.shippingService.Create(order.Id)
	if err != nil {
		rollbackOrchestrator.RollBack()
		return err
	}

	return nil
}
