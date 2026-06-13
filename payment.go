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

// PaymentRepository implements IPaymentRepository.
type PaymentRepository struct{}

func (*PaymentRepository) Save(payment Payment) error {
	return nil
}

func (*PaymentRepository) FindById(paymentId string) (Payment, error) {
	return Payment{}, nil
}

func (*PaymentRepository) SaveRefund(refund Refund) error {
	return nil
}

func (*PaymentRepository) FindRefundById(refundId string) (Refund, error) {
	return Refund{}, nil
}
