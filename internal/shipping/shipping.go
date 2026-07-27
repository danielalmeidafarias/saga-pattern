package shipping

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/danielalmeidafarias/saga-pattern/pkg/db"
)

type Status int

const (
	Pending Status = iota
	Created
	Started
	Delivered
	Canceled
	Failed
)

const (
	CreateOperation  = "shipping.create"
	GetOperation     = "shipping.get"
	UpdateOperation  = "shipping.update"
	DeleteOperation  = "shipping.delete"
	StartOperation   = "shipping.start"
	DeliverOperation = "shipping.deliver"
	CancelOperation  = "shipping.cancel"
)

type Shipping struct {
	ID        string
	UUID      string
	OrderUUID string
	Status    Status
}

type Service struct {
	database *sql.DB
	mu       sync.RWMutex
	failAt   string
}

func NewService(database *sql.DB, failAt string) (*Service, error) {
	if err := db.Migrate(database, `CREATE TABLE IF NOT EXISTS shippings (id TEXT PRIMARY KEY, uuid TEXT UNIQUE NOT NULL, order_uuid TEXT NOT NULL, status INTEGER NOT NULL)`); err != nil {
		return nil, err
	}
	return &Service{database: database, failAt: failAt}, nil
}

func (s *Service) Create(shipping Shipping) error {
	if err := s.fail(CreateOperation); err != nil {
		return err
	}
	_, err := s.database.Exec(`INSERT INTO shippings (id, uuid, order_uuid, status) VALUES (?, ?, ?, ?)`, shipping.ID, shipping.UUID, shipping.OrderUUID, shipping.Status)
	return err
}

func (s *Service) Get(uuid string) (Shipping, error) {
	if err := s.fail(GetOperation); err != nil {
		return Shipping{}, err
	}
	var shipping Shipping
	err := s.database.QueryRow(`SELECT id, uuid, order_uuid, status FROM shippings WHERE uuid = ?`, uuid).Scan(&shipping.ID, &shipping.UUID, &shipping.OrderUUID, &shipping.Status)
	return shipping, err
}

func (s *Service) Update(shipping Shipping) error {
	if err := s.fail(UpdateOperation); err != nil {
		return err
	}
	result, err := s.database.Exec(`UPDATE shippings SET order_uuid = ?, status = ? WHERE uuid = ?`, shipping.OrderUUID, shipping.Status, shipping.UUID)
	return resultError(result, err, shipping.UUID)
}

func (s *Service) Delete(uuid string) error {
	if err := s.fail(DeleteOperation); err != nil {
		return err
	}
	result, err := s.database.Exec(`DELETE FROM shippings WHERE uuid = ?`, uuid)
	return resultError(result, err, uuid)
}

func (s *Service) Start(uuid string) error   { return s.setStatus(StartOperation, uuid, Started) }
func (s *Service) Deliver(uuid string) error { return s.setStatus(DeliverOperation, uuid, Delivered) }
func (s *Service) Cancel(uuid string) error  { return s.setStatus(CancelOperation, uuid, Canceled) }

func (s *Service) SetFailure(operation string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failAt = operation
}

func (s *Service) setStatus(operation, uuid string, status Status) error {
	if err := s.fail(operation); err != nil {
		return err
	}
	result, err := s.database.Exec(`UPDATE shippings SET status = ? WHERE uuid = ?`, status, uuid)
	return resultError(result, err, uuid)
}

func (s *Service) fail(operation string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
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
		return fmt.Errorf("shipping not found: %s", uuid)
	}
	return nil
}
