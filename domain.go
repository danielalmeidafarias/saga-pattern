package main

import "errors"

// UUID is service-to-service reference. Id stays local/internal.

// Order

type OrderProduct struct {
	Product  Product
	Quantity int
}

type Order struct {
	Id           string
	UUID         string
	Products     []OrderProduct
	PaymentUUID  string
	ShippingUUID string
	Amount       float64
	Status       OrderStatus
}

type OrderStatus int

const (
	OrderPending OrderStatus = iota
	OrderConfirmed
	OrderCanceled
	OrderFailed
)

type IOderService interface {
	Create(order Order) error
	Get(orderUUID string) (Order, error)
	Confirm(orderUUID string) error
	Cancel(orderUUID string) error
}

// Payment

type Payment struct {
	Id         string
	UUID       string
	Status     PaymentStatus
	RefundId   *string
	RefundUUID *string
	Amont      float64
	Method     PaymentMethod
}

type Refund struct {
	Id     string
	UUID   string
	Status RefundStatus
}

type PaymentStatus int

const (
	PaymentPending PaymentStatus = iota
	PaymentProcessing
	PaymentSuccess
	PaymentFailed
	PaymentRefunded
)

type RefundStatus int

const (
	RefundPending RefundStatus = iota
	RefundProcessing
	RefundSuccess
	RefundFailed
)

type PaymentMethod int

const (
	PaymentMethodCreditCard PaymentMethod = iota
	PaymentMethodPix
)

type IPaymentService interface {
	Create(paymentUUID string, amount float64) error
	Get(paymentUUID string) (Payment, error)
	Process(paymentUUID string) error
	Cancel(paymentUUID string) error
	Refund(paymentUUID string) error
}

// Inventory

type Product struct {
	Id    string
	UUID  string
	Name  string
	Price float64
}

type IProductService interface {
	Create(productId string) error
	Get(productUUID string) (Product, error)
}

type Inventory struct {
	Id           string
	UUID         string
	ProductUUID  string
	Stock        int
	VirtualStock int
}

type IInventoryService interface {
	Create(productUUID string, stock int) error
	Get(inventoryUUID string) (Inventory, error)
	Update(inventoryUUID string, stock int, virtualStock int) error
}

// Shipping

type Shipping struct {
	Id        string
	UUID      string
	OrderUUID string
	Status    ShippingStatus
}

type ShippingStatus int

const (
	ShippingPending ShippingStatus = iota
	ShippingCreated
	ShippingStarted
	ShippingDelivered
	ShippingCanceled
	ShippingFailed
)

type IShippingService interface {
	Create(shippingUUID, orderUUID string) error
	Get(shippingUUID string) (Shipping, error)
	Start(shippingUUID string) error
	Deliver(shippingUUID string) error
	Cancel(shippingUUID string) error
}

// UseCase
type IUseCase[T any] interface {
	Run(in T) error
}

type UseCaseWithSagaAbstract[T any] interface {
	runFunc(in T) (*SagaOrchestrator, error)
}

type UseCaseWithSaga[T any] struct {
	abstract UseCaseWithSagaAbstract[T]
}

func NewUseCaseWithSaga[T any](
	abstract UseCaseWithSagaAbstract[T],
) UseCaseWithSaga[T] {
	return UseCaseWithSaga[T]{
		abstract: abstract,
	}
}

func (uc UseCaseWithSaga[T]) Run(in T) error {
	sagaOrchestrator, err := uc.abstract.runFunc(in)
	if err == nil {
		return nil
	}

	if sagaOrchestrator != nil {
		if errRollback := sagaOrchestrator.RollBack(); errRollback != nil {
			return errors.Join(err, errRollback)
		}
	}

	return err
}
