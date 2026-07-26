package msg

const (
	OrderCreatedMessage               MessageType = "order.created"
	ReserveInventoryMessage           MessageType = "inventory.reserve.requested"
	InventoryReservedMessage          MessageType = "inventory.reserved"
	InventoryReservationFailedMessage MessageType = "inventory.reservation.failed"
	ReleaseInventoryMessage           MessageType = "inventory.release.requested"
	CreatePaymentMessage              MessageType = "payment.create.requested"
	PaymentCreatedMessage             MessageType = "payment.created"
	PaymentCreationFailedMessage      MessageType = "payment.creation.failed"
	CancelPaymentMessage              MessageType = "payment.cancel.requested"
	CreateShippingMessage             MessageType = "shipping.create.requested"
	ShippingCreatedMessage            MessageType = "shipping.created"
	ShippingCreationFailedMessage     MessageType = "shipping.creation.failed"
	CancelShippingMessage             MessageType = "shipping.cancel.requested"
	OrderSagaSucceededMessage         MessageType = "order.saga.succeeded"
	OrderSagaFailedMessage            MessageType = "order.saga.failed"
)
