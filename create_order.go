package main

type CrateOrderUseCase struct {
	sg ISagaStep
}

// Separar estoque
// Criar pedido
// Criar pagamento
// Criar shipping
type CreateOrderInput struct {
}

func (uc *CrateOrderUseCase) Run() {

}

type CreateOrderSaga struct {
	SagaOrchestrator
}
