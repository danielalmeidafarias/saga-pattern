package main

// OrderService implements IOderService.
type OrderService struct{}

func (*OrderService) Create(order Order) (Order, error) {
	return Order{}, nil
}

func (*OrderService) Get(orderUUID string) (Order, error) {
	return Order{}, nil
}

func (*OrderService) Confirm(orderUUID string) error {
	return nil
}

func (*OrderService) Cancel(orderUUID string) error {
	return nil
}
