package main

// OrderService implements IOderService.
type OrderService struct{}

func (*OrderService) Create(productId string, amount float64) (Order, error) {
	return Order{}, nil
}

func (*OrderService) Get(orderId string) (Order, error) {
	return Order{}, nil
}

func (*OrderService) Confirm(orderId string) error {
	return nil
}

func (*OrderService) Cancel(orderId string) error {
	return nil
}

// OrderRepository implements IOrderRepository.
type OrderRepository struct{}

func (*OrderRepository) Save(order Order) error {
	return nil
}

func (*OrderRepository) FindById(orderId string) (Order, error) {
	return Order{}, nil
}
