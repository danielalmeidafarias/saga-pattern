package contracts

import "github.com/danielalmeidafarias/saga-pattern/pkg/msg"

const (
	TopicSagaEvents        = "saga-events"
	TopicInventoryCommands = "inventory-commands"
	TopicPaymentCommands   = "payment-commands"
	TopicShippingCommands  = "shipping-commands"
	TopicOrderEvents       = "order-events"
)

const (
	SubscriptionSaga      = "saga-orchestrator"
	SubscriptionInventory = "inventory-service"
	SubscriptionPayment   = "payment-service"
	SubscriptionShipping  = "shipping-service"
	SubscriptionOrder     = "order-service"
)

const (
	OrderCreated               msg.MessageType = "order.created"
	ReserveInventoryRequested  msg.MessageType = "inventory.reserve.requested"
	InventoryReserved          msg.MessageType = "inventory.reserved"
	InventoryReservationFailed msg.MessageType = "inventory.reservation.failed"
	ReleaseInventoryRequested  msg.MessageType = "inventory.release.requested"
	InventoryReleased          msg.MessageType = "inventory.released"
	InventoryReleaseFailed     msg.MessageType = "inventory.release.failed"
	CreatePaymentRequested     msg.MessageType = "payment.create.requested"
	PaymentCreated             msg.MessageType = "payment.created"
	PaymentCreationFailed      msg.MessageType = "payment.creation.failed"
	CancelPaymentRequested     msg.MessageType = "payment.cancel.requested"
	PaymentCanceled            msg.MessageType = "payment.canceled"
	PaymentCancelFailed        msg.MessageType = "payment.cancel.failed"
	CreateShippingRequested    msg.MessageType = "shipping.create.requested"
	ShippingCreated            msg.MessageType = "shipping.created"
	ShippingCreationFailed     msg.MessageType = "shipping.creation.failed"
	CancelShippingRequested    msg.MessageType = "shipping.cancel.requested"
	ShippingCanceled           msg.MessageType = "shipping.canceled"
	ShippingCancelFailed       msg.MessageType = "shipping.cancel.failed"
	OrderCreationSucceeded     msg.MessageType = "order.creation.succeeded"
	OrderCreationFailed        msg.MessageType = "order.creation.failed"
)

type OrderItem struct {
	ProductUUID string  `json:"productUuid"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
}

type OrderCreatedPayload struct {
	Amount float64     `json:"amount"`
	Items  []OrderItem `json:"items"`
}

type ResourceCreated struct {
	UUID string `json:"uuid"`
}

type OrderCreationSucceededPayload struct {
	PaymentUUID  string `json:"paymentUuid"`
	ShippingUUID string `json:"shippingUuid"`
}
