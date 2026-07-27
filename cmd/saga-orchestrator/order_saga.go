package main

import (
	"github.com/google/uuid"

	"github.com/danielalmeidafarias/saga-pattern/internal/saga"
	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/contracts"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

func orderSagaResolver() saga.Resolver {
	return saga.NewResolver(map[msg.MessageType]saga.ResolveFunc{
		contracts.OrderCreated: resolveOrderCreated,
	})
}

func resolveOrderCreated(trigger msg.Message) ([]saga.SagaStep, *pkg.Error) {
	return []saga.SagaStep{
		step(0,
			msg.NewMessage(contracts.TopicInventoryCommands, contracts.ReserveInventoryRequested, trigger.Payload),
			msg.NewMessage(contracts.TopicInventoryCommands, contracts.ReleaseInventoryRequested, nil)),
		step(1,
			msg.NewMessage(contracts.TopicPaymentCommands, contracts.CreatePaymentRequested, trigger.Payload),
			msg.NewMessage(contracts.TopicPaymentCommands, contracts.CancelPaymentRequested, nil)),
		step(2,
			msg.NewMessage(contracts.TopicShippingCommands, contracts.CreateShippingRequested, nil),
			msg.NewMessage(contracts.TopicShippingCommands, contracts.CancelShippingRequested, nil)),
	}, nil
}

func step(phase int, command, compensation msg.Message) saga.SagaStep {
	return saga.SagaStep{ID: uuid.NewString(), Phase: phase, Command: command, Compensation: &compensation}
}
