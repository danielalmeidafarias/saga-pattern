package payment

import (
	"database/sql"
	"fmt"

	"github.com/danielalmeidafarias/saga-pattern/pkg/db"
)

type Status int

const (
	Pending Status = iota
	Processing
	Succeeded
	Failed
	Refunded
)

type Method int

const (
	CreditCard Method = iota
	Pix
)

const (
	CreateOperation  = "payment.create"
	GetOperation     = "payment.get"
	UpdateOperation  = "payment.update"
	DeleteOperation  = "payment.delete"
	ProcessOperation = "payment.process"
	CancelOperation  = "payment.cancel"
	RefundOperation  = "payment.refund"
)

type Payment struct {
	ID     string
	UUID   string
	Amount float64
	Method Method
	Status Status
}

type Service struct {
	database *sql.DB
	failAt   string
}

func NewService(database *sql.DB, failAt string) (*Service, error) {
	if err := db.Migrate(database, `CREATE TABLE IF NOT EXISTS payments (id TEXT PRIMARY KEY, uuid TEXT UNIQUE NOT NULL, amount REAL NOT NULL, method INTEGER NOT NULL, status INTEGER NOT NULL)`); err != nil {
		return nil, err
	}
	return &Service{database: database, failAt: failAt}, nil
}

func (s *Service) Create(payment Payment) error {
	if err := s.fail(CreateOperation); err != nil {
		return err
	}
	_, err := s.database.Exec(`INSERT INTO payments (id, uuid, amount, method, status) VALUES (?, ?, ?, ?, ?)`, payment.ID, payment.UUID, payment.Amount, payment.Method, payment.Status)
	return err
}

func (s *Service) Get(uuid string) (Payment, error) {
	if err := s.fail(GetOperation); err != nil {
		return Payment{}, err
	}
	var payment Payment
	err := s.database.QueryRow(`SELECT id, uuid, amount, method, status FROM payments WHERE uuid = ?`, uuid).Scan(&payment.ID, &payment.UUID, &payment.Amount, &payment.Method, &payment.Status)
	return payment, err
}

func (s *Service) Update(payment Payment) error {
	if err := s.fail(UpdateOperation); err != nil {
		return err
	}
	result, err := s.database.Exec(`UPDATE payments SET amount = ?, method = ?, status = ? WHERE uuid = ?`, payment.Amount, payment.Method, payment.Status, payment.UUID)
	return resultError(result, err, payment.UUID)
}

func (s *Service) Delete(uuid string) error {
	if err := s.fail(DeleteOperation); err != nil {
		return err
	}
	result, err := s.database.Exec(`DELETE FROM payments WHERE uuid = ?`, uuid)
	return resultError(result, err, uuid)
}

func (s *Service) Process(uuid string) error { return s.setStatus(ProcessOperation, uuid, Succeeded) }
func (s *Service) Cancel(uuid string) error  { return s.setStatus(CancelOperation, uuid, Failed) }
func (s *Service) Refund(uuid string) error  { return s.setStatus(RefundOperation, uuid, Refunded) }

func (s *Service) setStatus(operation, uuid string, status Status) error {
	if err := s.fail(operation); err != nil {
		return err
	}
	result, err := s.database.Exec(`UPDATE payments SET status = ? WHERE uuid = ?`, status, uuid)
	return resultError(result, err, uuid)
}

func (s *Service) fail(operation string) error {
	if s.failAt == operation {
		return fmt.Errorf("injected failure: %s", operation)
	}
	return nil
}

func resultError(result sql.Result, err error, uuid string) error {
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("payment not found: %s", uuid)
	}
	return nil
}
