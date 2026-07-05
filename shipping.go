package main

// ShippingService implements IShippingService.
type ShippingService struct{}

func (*ShippingService) Create(orderUUID string) (Shipping, error) {
	return Shipping{}, nil
}

func (*ShippingService) Get(shippingUUID string) (Shipping, error) {
	return Shipping{}, nil
}

func (*ShippingService) Start(shippingUUID string) error {
	return nil
}

func (*ShippingService) Deliver(shippingUUID string) error {
	return nil
}

func (*ShippingService) Cancel(shippingUUID string) error {
	return nil
}
