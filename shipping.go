package main

// ShippingService implements IShippingService.
type ShippingService struct{}

func (*ShippingService) Create(orderId string) (Shipping, error) {
	return Shipping{}, nil
}

func (*ShippingService) Get(shippingId string) (Shipping, error) {
	return Shipping{}, nil
}

func (*ShippingService) Start(shippingId string) error {
	return nil
}

func (*ShippingService) Deliver(shippingId string) error {
	return nil
}

func (*ShippingService) Cancel(shippingId string) error {
	return nil
}
