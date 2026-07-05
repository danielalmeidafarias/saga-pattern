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

/*
Saga v2 - Nesse modelo, cada passo do processo e adicionado com a funcao compensadora e sao executados automaticamente, sequencialmente.
O usecase implementa a funcao runFunc que retorna o orchestrador do saga.
O tipo UseCaseWithSaga e responsavel por executar o orchestrador de saga, e caso tenha erro, ele executa os rollbacks automaticamente.

Nesse formato de saga, cada passo do processo e adicionado com a funcao compensadora e sao executados automaticamente, sequencialmente.
Para funcionar, os ids devem ser gerados antes da execucao para permitir que funcoes rollback sejam definidas
- Pontos positivos: Mais simplicidade de implementacao e automacao do processo de rollback
- Pontos negativos: Menos flexibilidade, menor controle do fluxo de execucao, e nao permite que execucoes de rollback dependam das funcoes principais.

Para integracoes externas, talvez seja impossivel de implementar, visto que os rollbacks normalmente dependem de ids gerados nas APIs/Bancos de dados externos.
Mas, para sistemas de API's internas, provavelmente esse modelo e possivel

Outra solucao talvez seja a adicao de outro servico de orquestracao que iria de fato lidar com essa orchestracao de rollbacks
Enquanto nosso servico iria simplesmente gravar ids das tarefas que poderiam ser compensadas futuramente. Por exemplo, um servico de filas
*/
func (uc CreateOrderUseCase) runFunc(in CreateOrderInput) (*SagaOrchestrator, error) {
	sagaOrchestrator := NewSagaOrchestrator()

	var inventoryProductList = make(map[OrderProduct]InventoryProduct)
	for _, p := range in.Products {
		inventoryProduct, err := uc.inventoryService.GetProduct(in.InventoryUUID, p.Product.UUID)
		if err != nil {
			return &sagaOrchestrator, err
		}

		if inventoryProduct.VirtualStock < p.Quantity {
			return &sagaOrchestrator, fmt.Errorf("Requested product quantity not available for product: %s", p.Product.Name)
		}

		inventoryProductList[p] = inventoryProduct
	}

	for p, i := range inventoryProductList {
		sagaOrchestrator.AddStep(NewSagaStep(
			func() error {
				return uc.inventoryService.UpdateProduct(
					i.InventoryUUID,
					i.ProductUUID,
					i.Stock,
					i.VirtualStock-p.Quantity,
				)
			},
			func() error {
				return uc.inventoryService.UpdateProduct(
					i.InventoryUUID,
					i.ProductUUID,
					i.Stock,
					i.VirtualStock,
				)
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
