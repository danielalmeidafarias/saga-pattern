package inventory

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/danielalmeidafarias/saga-pattern/pkg/db"
)

const (
	CreateOperation        = "inventory.create"
	GetOperation           = "inventory.get"
	DeleteOperation        = "inventory.delete"
	CreateProductOperation = "inventory.product.create"
	GetProductOperation    = "inventory.product.get"
	UpdateProductOperation = "inventory.product.update"
	DeleteProductOperation = "inventory.product.delete"
	ReserveOperation       = "inventory.reserve"
	ReleaseOperation       = "inventory.release"
)

type Inventory struct {
	ID   string
	UUID string
}

type Product struct {
	InventoryUUID string
	ProductUUID   string
	Stock         int
	VirtualStock  int
}

type ReservationStatus int

const (
	Reserved ReservationStatus = iota
	Released
)

type Reservation struct {
	UUID      string
	OrderUUID string
	Status    ReservationStatus
}

type Service struct {
	database *sql.DB
	mu       sync.RWMutex
	failAt   string
}

func NewService(database *sql.DB, failAt string) (*Service, error) {
	if err := db.Migrate(database,
		`CREATE TABLE IF NOT EXISTS inventories (id TEXT PRIMARY KEY, uuid TEXT UNIQUE NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS inventory_products (inventory_uuid TEXT NOT NULL, product_uuid TEXT NOT NULL, stock INTEGER NOT NULL, virtual_stock INTEGER NOT NULL, PRIMARY KEY (inventory_uuid, product_uuid))`,
		`CREATE TABLE IF NOT EXISTS inventory_reservations (uuid TEXT PRIMARY KEY, order_uuid TEXT UNIQUE NOT NULL, status INTEGER NOT NULL)`,
	); err != nil {
		return nil, err
	}
	return &Service{database: database, failAt: failAt}, nil
}

func (s *Service) Create(inventory Inventory) error {
	if err := s.fail(CreateOperation); err != nil {
		return err
	}
	_, err := s.database.Exec(`INSERT INTO inventories (id, uuid) VALUES (?, ?)`, inventory.ID, inventory.UUID)
	return err
}

func (s *Service) Get(uuid string) (Inventory, error) {
	if err := s.fail(GetOperation); err != nil {
		return Inventory{}, err
	}
	var inventory Inventory
	err := s.database.QueryRow(`SELECT id, uuid FROM inventories WHERE uuid = ?`, uuid).Scan(&inventory.ID, &inventory.UUID)
	return inventory, err
}

func (s *Service) Delete(uuid string) error {
	if err := s.fail(DeleteOperation); err != nil {
		return err
	}
	result, err := s.database.Exec(`DELETE FROM inventories WHERE uuid = ?`, uuid)
	return resultError(result, err, "inventory", uuid)
}

func (s *Service) CreateProduct(product Product) error {
	if err := s.fail(CreateProductOperation); err != nil {
		return err
	}
	_, err := s.database.Exec(`INSERT INTO inventory_products (inventory_uuid, product_uuid, stock, virtual_stock) VALUES (?, ?, ?, ?)`, product.InventoryUUID, product.ProductUUID, product.Stock, product.VirtualStock)
	return err
}

func (s *Service) GetProduct(inventoryUUID, productUUID string) (Product, error) {
	if err := s.fail(GetProductOperation); err != nil {
		return Product{}, err
	}
	var product Product
	err := s.database.QueryRow(`SELECT inventory_uuid, product_uuid, stock, virtual_stock FROM inventory_products WHERE inventory_uuid = ? AND product_uuid = ?`, inventoryUUID, productUUID).Scan(&product.InventoryUUID, &product.ProductUUID, &product.Stock, &product.VirtualStock)
	return product, err
}

func (s *Service) UpdateProduct(product Product) error {
	if err := s.fail(UpdateProductOperation); err != nil {
		return err
	}
	result, err := s.database.Exec(`UPDATE inventory_products SET stock = ?, virtual_stock = ? WHERE inventory_uuid = ? AND product_uuid = ?`, product.Stock, product.VirtualStock, product.InventoryUUID, product.ProductUUID)
	return resultError(result, err, "inventory product", product.InventoryUUID+"/"+product.ProductUUID)
}

func (s *Service) DeleteProduct(inventoryUUID, productUUID string) error {
	if err := s.fail(DeleteProductOperation); err != nil {
		return err
	}
	result, err := s.database.Exec(`DELETE FROM inventory_products WHERE inventory_uuid = ? AND product_uuid = ?`, inventoryUUID, productUUID)
	return resultError(result, err, "inventory product", inventoryUUID+"/"+productUUID)
}

func (s *Service) Reserve(reservation Reservation) error {
	if err := s.fail(ReserveOperation); err != nil {
		return err
	}
	_, err := s.database.Exec(`INSERT INTO inventory_reservations (uuid, order_uuid, status) VALUES (?, ?, ?) ON CONFLICT(order_uuid) DO UPDATE SET status = excluded.status`, reservation.UUID, reservation.OrderUUID, Reserved)
	return err
}

func (s *Service) Release(orderUUID string) error {
	if err := s.fail(ReleaseOperation); err != nil {
		return err
	}
	result, err := s.database.Exec(`UPDATE inventory_reservations SET status = ? WHERE order_uuid = ?`, Released, orderUUID)
	return resultError(result, err, "reservation", orderUUID)
}

func (s *Service) GetReservation(orderUUID string) (Reservation, error) {
	var reservation Reservation
	err := s.database.QueryRow(`SELECT uuid, order_uuid, status FROM inventory_reservations WHERE order_uuid = ?`, orderUUID).Scan(&reservation.UUID, &reservation.OrderUUID, &reservation.Status)
	return reservation, err
}

func (s *Service) SetFailure(operation string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failAt = operation
}

func (s *Service) fail(operation string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.failAt == operation {
		return fmt.Errorf("injected failure: %s", operation)
	}
	return nil
}

func resultError(result sql.Result, err error, entity, id string) error {
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("%s not found: %s", entity, id)
	}
	return nil
}
