package main

// PaymentService implements IPaymentService.
type PaymentService struct{}

func (*PaymentService) Create(amount float64) (Payment, error) {
	return Payment{}, nil
}

func (*PaymentService) Get(paymentId string) (Payment, error) {
	return Payment{}, nil
}

func (*PaymentService) Process(paymentId string) error {
	return nil
}

func (*PaymentService) Cancel(paymentId string) error {
	return nil
}

func (*PaymentService) Refund(paymentId string) error {
	return nil
}
