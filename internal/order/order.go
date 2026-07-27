package order

import (
	"database/sql"
	"fmt"

	"github.com/danielalmeidafarias/saga-pattern/pkg/db"
)

type Status int

const (
	Pending Status = iota
	Confirmed
	Canceled
	Failed
)

const (
	CreateOperation  = "order.create"
	GetOperation     = "order.get"
	UpdateOperation  = "order.update"
	DeleteOperation  = "order.delete"
	ConfirmOperation = "order.confirm"
	CancelOperation  = "order.cancel"
)

type Item struct {
	ProductUUID string
	Name        string
	Price       float64
	Quantity    int
}

type Order struct {
	ID           string
	UUID         string
	Items        []Item
	PaymentUUID  string
	ShippingUUID string
	Amount       float64
	Status       Status
}

type Service struct {
	database *sql.DB
	failAt   string
}

func NewService(database *sql.DB, failAt string) (*Service, error) {
	if err := db.Migrate(database,
		`CREATE TABLE IF NOT EXISTS orders (id TEXT PRIMARY KEY, uuid TEXT UNIQUE NOT NULL, payment_uuid TEXT NOT NULL, shipping_uuid TEXT NOT NULL, amount REAL NOT NULL, status INTEGER NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS order_items (order_uuid TEXT NOT NULL, product_uuid TEXT NOT NULL, name TEXT NOT NULL, price REAL NOT NULL, quantity INTEGER NOT NULL, PRIMARY KEY (order_uuid, product_uuid))`,
	); err != nil {
		return nil, err
	}
	return &Service{database: database, failAt: failAt}, nil
}

func (s *Service) Create(order Order) error {
	if err := s.fail(CreateOperation); err != nil {
		return err
	}
	tx, err := s.database.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO orders (id, uuid, payment_uuid, shipping_uuid, amount, status) VALUES (?, ?, ?, ?, ?, ?)`, order.ID, order.UUID, order.PaymentUUID, order.ShippingUUID, order.Amount, order.Status); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err = replaceItems(tx, order.UUID, order.Items); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Service) Get(uuid string) (Order, error) {
	if err := s.fail(GetOperation); err != nil {
		return Order{}, err
	}
	var order Order
	if err := s.database.QueryRow(`SELECT id, uuid, payment_uuid, shipping_uuid, amount, status FROM orders WHERE uuid = ?`, uuid).Scan(&order.ID, &order.UUID, &order.PaymentUUID, &order.ShippingUUID, &order.Amount, &order.Status); err != nil {
		return Order{}, err
	}
	rows, err := s.database.Query(`SELECT product_uuid, name, price, quantity FROM order_items WHERE order_uuid = ? ORDER BY product_uuid`, uuid)
	if err != nil {
		return Order{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ProductUUID, &item.Name, &item.Price, &item.Quantity); err != nil {
			return Order{}, err
		}
		order.Items = append(order.Items, item)
	}
	return order, rows.Err()
}

func (s *Service) Update(order Order) error {
	if err := s.fail(UpdateOperation); err != nil {
		return err
	}
	tx, err := s.database.Begin()
	if err != nil {
		return err
	}
	result, err := tx.Exec(`UPDATE orders SET payment_uuid = ?, shipping_uuid = ?, amount = ?, status = ? WHERE uuid = ?`, order.PaymentUUID, order.ShippingUUID, order.Amount, order.Status, order.UUID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if err = requireOne(result, "order", order.UUID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err = replaceItems(tx, order.UUID, order.Items); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Service) Delete(uuid string) error {
	if err := s.fail(DeleteOperation); err != nil {
		return err
	}
	result, err := s.database.Exec(`DELETE FROM orders WHERE uuid = ?`, uuid)
	if err != nil {
		return err
	}
	return requireOne(result, "order", uuid)
}

func (s *Service) Confirm(uuid string) error { return s.setStatus(ConfirmOperation, uuid, Confirmed) }
func (s *Service) Cancel(uuid string) error  { return s.setStatus(CancelOperation, uuid, Canceled) }

func (s *Service) setStatus(operation, uuid string, status Status) error {
	if err := s.fail(operation); err != nil {
		return err
	}
	result, err := s.database.Exec(`UPDATE orders SET status = ? WHERE uuid = ?`, status, uuid)
	if err != nil {
		return err
	}
	return requireOne(result, "order", uuid)
}

func (s *Service) fail(operation string) error {
	if s.failAt == operation {
		return fmt.Errorf("injected failure: %s", operation)
	}
	return nil
}

func replaceItems(tx *sql.Tx, orderUUID string, items []Item) error {
	if _, err := tx.Exec(`DELETE FROM order_items WHERE order_uuid = ?`, orderUUID); err != nil {
		return err
	}
	for _, item := range items {
		if _, err := tx.Exec(`INSERT INTO order_items (order_uuid, product_uuid, name, price, quantity) VALUES (?, ?, ?, ?, ?)`, orderUUID, item.ProductUUID, item.Name, item.Price, item.Quantity); err != nil {
			return err
		}
	}
	return nil
}

func requireOne(result sql.Result, entity, id string) error {
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("%s not found: %s", entity, id)
	}
	return nil
}
