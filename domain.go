package main

// ──────────────────────────────────────────────
// Order
// ──────────────────────────────────────────────

type Order struct {
	Id         string
	ProductId  string
	PaymentId  string
	ShippingId string
	Amount     float64
	Status     OrderStatus
}

type OrderStatus int

const (
	OrderPending OrderStatus = iota
	OrderConfirmed
	OrderCanceled
	OrderFailed
)

type IOderService interface {
	Create(productsId []string, amount float64) (Order, error)
	Get(orderId string) (Order, error)
	Confirm(orderId string) error
	Cancel(orderId string) error
}

// ──────────────────────────────────────────────
// Payment
// ──────────────────────────────────────────────

type Payment struct {
	Id       string
	Status   PaymentStatus
	RefundId *string
	Amont    float64
	Method   PaymentMethod
}

type Refund struct {
	Id     string
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
	Create(amount float64) (Payment, error)
	Get(paymentId string) (Payment, error)
	Process(paymentId string) error
	Cancel(paymentId string) error
	Refund(paymentId string) error
}

// ──────────────────────────────────────────────
// Inventory
// ──────────────────────────────────────────────

type Product struct {
	Id    string
	Name  string
	Price float64
}

type IProductService interface {
	Create(productId string) (Product, error)
	Get(productId string) (Product, error)
}

type Inventory struct {
	Id           string
	ProductId    string
	Stock        int
	VirtualStock int
}

type IInventoryService interface {
	Create(productId string, stock int) (Inventory, error)
	Get(productId string) (Inventory, error)
	Update(inventoryId string, stock int, virtualStock int) error
}

// ──────────────────────────────────────────────
// Shipping
// ──────────────────────────────────────────────

type Shipping struct {
	Id      string
	OrderId string
	Status  ShippingStatus
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
	Create(orderId string) (Shipping, error)
	Get(shippingId string) (Shipping, error)
	Start(shippingId string) error
	Deliver(shippingId string) error
	Cancel(shippingId string) error
}

type UseCase interface {
	Run(input interface{}) error
}
