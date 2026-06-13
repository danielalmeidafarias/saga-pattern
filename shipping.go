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

// ShippingRepository implements IShippingRepository.
type ShippingRepository struct{}

func (*ShippingRepository) Save(shipping Shipping) error {
	return nil
}

func (*ShippingRepository) FindById(shippingId string) (Shipping, error) {
	return Shipping{}, nil
}
