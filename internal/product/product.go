package product

import (
	"database/sql"
	"fmt"

	"github.com/danielalmeidafarias/saga-pattern/pkg/db"
)

const (
	CreateOperation = "product.create"
	GetOperation    = "product.get"
	UpdateOperation = "product.update"
	DeleteOperation = "product.delete"
)

type Product struct {
	ID    string
	UUID  string
	Name  string
	Price float64
}

type Service struct {
	database *sql.DB
	failAt   string
}

func NewService(database *sql.DB, failAt string) (*Service, error) {
	if err := db.Migrate(database, `CREATE TABLE IF NOT EXISTS products (id TEXT PRIMARY KEY, uuid TEXT UNIQUE NOT NULL, name TEXT NOT NULL, price REAL NOT NULL)`); err != nil {
		return nil, err
	}
	return &Service{database: database, failAt: failAt}, nil
}

func (s *Service) Create(product Product) error {
	if err := s.fail(CreateOperation); err != nil {
		return err
	}
	_, err := s.database.Exec(`INSERT INTO products (id, uuid, name, price) VALUES (?, ?, ?, ?)`, product.ID, product.UUID, product.Name, product.Price)
	return err
}

func (s *Service) Get(uuid string) (Product, error) {
	if err := s.fail(GetOperation); err != nil {
		return Product{}, err
	}
	var product Product
	err := s.database.QueryRow(`SELECT id, uuid, name, price FROM products WHERE uuid = ?`, uuid).Scan(&product.ID, &product.UUID, &product.Name, &product.Price)
	return product, err
}

func (s *Service) Update(product Product) error {
	if err := s.fail(UpdateOperation); err != nil {
		return err
	}
	result, err := s.database.Exec(`UPDATE products SET name = ?, price = ? WHERE uuid = ?`, product.Name, product.Price, product.UUID)
	if err != nil {
		return err
	}
	return requireOne(result, product.UUID)
}

func (s *Service) Delete(uuid string) error {
	if err := s.fail(DeleteOperation); err != nil {
		return err
	}
	result, err := s.database.Exec(`DELETE FROM products WHERE uuid = ?`, uuid)
	if err != nil {
		return err
	}
	return requireOne(result, uuid)
}

func (s *Service) fail(operation string) error {
	if s.failAt == operation {
		return fmt.Errorf("injected failure: %s", operation)
	}
	return nil
}

func requireOne(result sql.Result, uuid string) error {
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("product not found: %s", uuid)
	}
	return nil
}
