package main

// PaymentService implements IPaymentService.
type PaymentService struct{}

func (*PaymentService) Create(amount float64) (Payment, error) {
	return Payment{}, nil
}

func (*PaymentService) Get(paymentUUID string) (Payment, error) {
	return Payment{}, nil
}

func (*PaymentService) Process(paymentUUID string) error {
	return nil
}

func (*PaymentService) Cancel(paymentUUID string) error {
	return nil
}

func (*PaymentService) Refund(paymentUUID string) error {
	return nil
}
