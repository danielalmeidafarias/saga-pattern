package order

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const (
	ProductCreateOperation          = "product.create"
	ProductGetOperation             = "product.get"
	ProductUpdateOperation          = "product.update"
	ProductDeleteOperation          = "product.delete"
	InventoryCreateOperation        = "inventory.create"
	InventoryGetOperation           = "inventory.get"
	InventoryProductGetOperation    = "inventory.product.get"
	InventoryProductUpdateOperation = "inventory.product.update"
	PaymentCreateOperation          = "payment.create"
	PaymentGetOperation             = "payment.get"
	PaymentProcessOperation         = "payment.process"
	PaymentCancelOperation          = "payment.cancel"
	PaymentRefundOperation          = "payment.refund"
	ShippingCreateOperation         = "shipping.create"
	ShippingGetOperation            = "shipping.get"
	ShippingStartOperation          = "shipping.start"
	ShippingDeliverOperation        = "shipping.deliver"
	ShippingCancelOperation         = "shipping.cancel"
	OrderCreateOperation            = "order.create"
	OrderGetOperation               = "order.get"
	OrderUpdateOperation            = "order.update"
	OrderConfirmOperation           = "order.confirm"
	OrderCancelOperation            = "order.cancel"
)

type FailureInjector struct {
	Operation string
	Err       error
}

func NewFailureInjector(operation string) *FailureInjector {
	return &FailureInjector{Operation: operation}
}

func (f *FailureInjector) error(operation string) error {
	if f == nil || f.Operation != operation {
		return nil
	}
	if f.Err != nil {
		return f.Err
	}
	return fmt.Errorf("injected failure: %s", operation)
}

func OpenSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := InitializeSQLite(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func InitializeSQLite(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS products (id TEXT PRIMARY KEY, uuid TEXT UNIQUE NOT NULL, name TEXT NOT NULL, price REAL NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS inventories (id TEXT PRIMARY KEY, uuid TEXT UNIQUE NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS inventory_products (inventory_uuid TEXT NOT NULL, product_uuid TEXT NOT NULL, stock INTEGER NOT NULL, virtual_stock INTEGER NOT NULL, PRIMARY KEY (inventory_uuid, product_uuid))`,
		`CREATE TABLE IF NOT EXISTS payments (id TEXT PRIMARY KEY, uuid TEXT UNIQUE NOT NULL, status INTEGER NOT NULL, refund_uuid TEXT, amount REAL NOT NULL, method INTEGER NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS shippings (id TEXT PRIMARY KEY, uuid TEXT UNIQUE NOT NULL, order_uuid TEXT NOT NULL, status INTEGER NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS orders (id TEXT PRIMARY KEY, uuid TEXT UNIQUE NOT NULL, payment_uuid TEXT NOT NULL, shipping_uuid TEXT NOT NULL, amount REAL NOT NULL, status INTEGER NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS order_products (order_uuid TEXT NOT NULL, product_uuid TEXT NOT NULL, product_id TEXT NOT NULL, product_name TEXT NOT NULL, product_price REAL NOT NULL, quantity INTEGER NOT NULL, PRIMARY KEY (order_uuid, product_uuid))`,
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

type SQLiteProductService struct {
	db       *sql.DB
	failures *FailureInjector
}

var _ IProductService = (*SQLiteProductService)(nil)

func NewSQLiteProductService(db *sql.DB, failures *FailureInjector) *SQLiteProductService {
	return &SQLiteProductService{db: db, failures: failures}
}

func (s *SQLiteProductService) Create(productID string) error {
	if err := s.failures.error(ProductCreateOperation); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO products (id, uuid, name, price) VALUES (?, ?, '', 0)`, productID, productID)
	return err
}

func (s *SQLiteProductService) Get(productUUID string) (Product, error) {
	if err := s.failures.error(ProductGetOperation); err != nil {
		return Product{}, err
	}
	var product Product
	err := s.db.QueryRow(`SELECT id, uuid, name, price FROM products WHERE uuid = ?`, productUUID).Scan(&product.Id, &product.UUID, &product.Name, &product.Price)
	return product, err
}

func (s *SQLiteProductService) Update(product Product) error {
	if err := s.failures.error(ProductUpdateOperation); err != nil {
		return err
	}
	result, err := s.db.Exec(`UPDATE products SET name = ?, price = ? WHERE uuid = ?`, product.Name, product.Price, product.UUID)
	if err != nil {
		return err
	}
	return requireOneRow(result, "product", product.UUID)
}

func (s *SQLiteProductService) Delete(productUUID string) error {
	if err := s.failures.error(ProductDeleteOperation); err != nil {
		return err
	}
	result, err := s.db.Exec(`DELETE FROM products WHERE uuid = ?`, productUUID)
	if err != nil {
		return err
	}
	return requireOneRow(result, "product", productUUID)
}

type SQLiteInventoryService struct {
	db       *sql.DB
	failures *FailureInjector
}

var _ IInventoryService = (*SQLiteInventoryService)(nil)

func NewSQLiteInventoryService(db *sql.DB, failures *FailureInjector) *SQLiteInventoryService {
	return &SQLiteInventoryService{db: db, failures: failures}
}

func (s *SQLiteInventoryService) Create(inventoryID string) error {
	if err := s.failures.error(InventoryCreateOperation); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO inventories (id, uuid) VALUES (?, ?)`, inventoryID, inventoryID)
	return err
}

func (s *SQLiteInventoryService) Get(inventoryUUID string) (Inventory, error) {
	if err := s.failures.error(InventoryGetOperation); err != nil {
		return Inventory{}, err
	}
	var inventory Inventory
	err := s.db.QueryRow(`SELECT id, uuid FROM inventories WHERE uuid = ?`, inventoryUUID).Scan(&inventory.Id, &inventory.UUID)
	return inventory, err
}

func (s *SQLiteInventoryService) AddProduct(inventoryUUID, productUUID string, stock, virtualStock int) error {
	_, err := s.db.Exec(`INSERT INTO inventory_products (inventory_uuid, product_uuid, stock, virtual_stock) VALUES (?, ?, ?, ?)`, inventoryUUID, productUUID, stock, virtualStock)
	return err
}

func (s *SQLiteInventoryService) GetProduct(inventoryUUID, productUUID string) (InventoryProduct, error) {
	if err := s.failures.error(InventoryProductGetOperation); err != nil {
		return InventoryProduct{}, err
	}
	var product InventoryProduct
	err := s.db.QueryRow(`SELECT inventory_uuid, product_uuid, stock, virtual_stock FROM inventory_products WHERE inventory_uuid = ? AND product_uuid = ?`, inventoryUUID, productUUID).Scan(&product.InventoryUUID, &product.ProductUUID, &product.Stock, &product.VirtualStock)
	return product, err
}

func (s *SQLiteInventoryService) UpdateProduct(inventoryUUID, productUUID string, stock, virtualStock int) error {
	if err := s.failures.error(InventoryProductUpdateOperation); err != nil {
		return err
	}
	result, err := s.db.Exec(`UPDATE inventory_products SET stock = ?, virtual_stock = ? WHERE inventory_uuid = ? AND product_uuid = ?`, stock, virtualStock, inventoryUUID, productUUID)
	if err != nil {
		return err
	}
	return requireOneRow(result, "inventory product", inventoryUUID+"/"+productUUID)
}

func (s *SQLiteInventoryService) Delete(inventoryUUID string) error {
	result, err := s.db.Exec(`DELETE FROM inventories WHERE uuid = ?`, inventoryUUID)
	return resultError(result, err, "inventory", inventoryUUID)
}

type SQLitePaymentService struct {
	db       *sql.DB
	failures *FailureInjector
}

var _ IPaymentService = (*SQLitePaymentService)(nil)

func NewSQLitePaymentService(db *sql.DB, failures *FailureInjector) *SQLitePaymentService {
	return &SQLitePaymentService{db: db, failures: failures}
}

func (s *SQLitePaymentService) Create(paymentUUID string, amount float64) error {
	if err := s.failures.error(PaymentCreateOperation); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO payments (id, uuid, status, amount, method) VALUES (?, ?, ?, ?, ?)`, paymentUUID, paymentUUID, PaymentPending, amount, PaymentMethodCreditCard)
	return err
}

func (s *SQLitePaymentService) Get(paymentUUID string) (Payment, error) {
	if err := s.failures.error(PaymentGetOperation); err != nil {
		return Payment{}, err
	}
	var payment Payment
	var refundUUID sql.NullString
	err := s.db.QueryRow(`SELECT id, uuid, status, refund_uuid, amount, method FROM payments WHERE uuid = ?`, paymentUUID).Scan(&payment.Id, &payment.UUID, &payment.Status, &refundUUID, &payment.Amont, &payment.Method)
	if err == nil && refundUUID.Valid {
		payment.RefundUUID = &refundUUID.String
	}
	return payment, err
}

func (s *SQLitePaymentService) Process(paymentUUID string) error {
	return s.updateStatus(PaymentProcessOperation, paymentUUID, PaymentSuccess)
}

func (s *SQLitePaymentService) Cancel(paymentUUID string) error {
	return s.updateStatus(PaymentCancelOperation, paymentUUID, PaymentFailed)
}

func (s *SQLitePaymentService) Refund(paymentUUID string) error {
	return s.updateStatus(PaymentRefundOperation, paymentUUID, PaymentRefunded)
}

func (s *SQLitePaymentService) Delete(paymentUUID string) error {
	result, err := s.db.Exec(`DELETE FROM payments WHERE uuid = ?`, paymentUUID)
	return resultError(result, err, "payment", paymentUUID)
}

func (s *SQLitePaymentService) updateStatus(operation, paymentUUID string, status PaymentStatus) error {
	if err := s.failures.error(operation); err != nil {
		return err
	}
	result, err := s.db.Exec(`UPDATE payments SET status = ? WHERE uuid = ?`, status, paymentUUID)
	if err != nil {
		return err
	}
	return requireOneRow(result, "payment", paymentUUID)
}

type SQLiteShippingService struct {
	db       *sql.DB
	failures *FailureInjector
}

var _ IShippingService = (*SQLiteShippingService)(nil)

func NewSQLiteShippingService(db *sql.DB, failures *FailureInjector) *SQLiteShippingService {
	return &SQLiteShippingService{db: db, failures: failures}
}

func (s *SQLiteShippingService) Create(shippingUUID, orderUUID string) error {
	if err := s.failures.error(ShippingCreateOperation); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO shippings (id, uuid, order_uuid, status) VALUES (?, ?, ?, ?)`, shippingUUID, shippingUUID, orderUUID, ShippingPending)
	return err
}

func (s *SQLiteShippingService) Get(shippingUUID string) (Shipping, error) {
	if err := s.failures.error(ShippingGetOperation); err != nil {
		return Shipping{}, err
	}
	var shipping Shipping
	err := s.db.QueryRow(`SELECT id, uuid, order_uuid, status FROM shippings WHERE uuid = ?`, shippingUUID).Scan(&shipping.Id, &shipping.UUID, &shipping.OrderUUID, &shipping.Status)
	return shipping, err
}

func (s *SQLiteShippingService) Start(shippingUUID string) error {
	return s.updateStatus(ShippingStartOperation, shippingUUID, ShippingStarted)
}

func (s *SQLiteShippingService) Deliver(shippingUUID string) error {
	return s.updateStatus(ShippingDeliverOperation, shippingUUID, ShippingDelivered)
}

func (s *SQLiteShippingService) Cancel(shippingUUID string) error {
	return s.updateStatus(ShippingCancelOperation, shippingUUID, ShippingCanceled)
}

func (s *SQLiteShippingService) Delete(shippingUUID string) error {
	result, err := s.db.Exec(`DELETE FROM shippings WHERE uuid = ?`, shippingUUID)
	return resultError(result, err, "shipping", shippingUUID)
}

func (s *SQLiteShippingService) updateStatus(operation, shippingUUID string, status ShippingStatus) error {
	if err := s.failures.error(operation); err != nil {
		return err
	}
	result, err := s.db.Exec(`UPDATE shippings SET status = ? WHERE uuid = ?`, status, shippingUUID)
	if err != nil {
		return err
	}
	return requireOneRow(result, "shipping", shippingUUID)
}

type SQLiteOrderService struct {
	db       *sql.DB
	failures *FailureInjector
}

var _ IOderService = (*SQLiteOrderService)(nil)

func NewSQLiteOrderService(db *sql.DB, failures *FailureInjector) *SQLiteOrderService {
	return &SQLiteOrderService{db: db, failures: failures}
}

func (s *SQLiteOrderService) Create(order Order) error {
	if err := s.failures.error(OrderCreateOperation); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO orders (id, uuid, payment_uuid, shipping_uuid, amount, status) VALUES (?, ?, ?, ?, ?, ?)`, order.UUID, order.UUID, order.PaymentUUID, order.ShippingUUID, order.Amount, order.Status); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err = insertOrderProducts(tx, order); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *SQLiteOrderService) Get(orderUUID string) (Order, error) {
	if err := s.failures.error(OrderGetOperation); err != nil {
		return Order{}, err
	}
	var order Order
	err := s.db.QueryRow(`SELECT id, uuid, payment_uuid, shipping_uuid, amount, status FROM orders WHERE uuid = ?`, orderUUID).Scan(&order.Id, &order.UUID, &order.PaymentUUID, &order.ShippingUUID, &order.Amount, &order.Status)
	if err != nil {
		return Order{}, err
	}
	rows, err := s.db.Query(`SELECT product_id, product_uuid, product_name, product_price, quantity FROM order_products WHERE order_uuid = ? ORDER BY product_uuid`, orderUUID)
	if err != nil {
		return Order{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var product OrderProduct
		if err := rows.Scan(&product.Product.Id, &product.Product.UUID, &product.Product.Name, &product.Product.Price, &product.Quantity); err != nil {
			return Order{}, err
		}
		order.Products = append(order.Products, product)
	}
	return order, rows.Err()
}

func (s *SQLiteOrderService) Update(order Order) error {
	if err := s.failures.error(OrderUpdateOperation); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	result, err := tx.Exec(`UPDATE orders SET payment_uuid = ?, shipping_uuid = ?, amount = ?, status = ? WHERE uuid = ?`, order.PaymentUUID, order.ShippingUUID, order.Amount, order.Status, order.UUID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if err = requireOneRow(result, "order", order.UUID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err = insertOrderProducts(tx, order); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *SQLiteOrderService) Confirm(orderUUID string) error {
	return s.updateStatus(OrderConfirmOperation, orderUUID, OrderConfirmed)
}

func (s *SQLiteOrderService) Cancel(orderUUID string) error {
	return s.updateStatus(OrderCancelOperation, orderUUID, OrderCanceled)
}

func (s *SQLiteOrderService) Delete(orderUUID string) error {
	result, err := s.db.Exec(`DELETE FROM orders WHERE uuid = ?`, orderUUID)
	return resultError(result, err, "order", orderUUID)
}

func (s *SQLiteOrderService) updateStatus(operation, orderUUID string, status OrderStatus) error {
	if err := s.failures.error(operation); err != nil {
		return err
	}
	result, err := s.db.Exec(`UPDATE orders SET status = ? WHERE uuid = ?`, status, orderUUID)
	if err != nil {
		return err
	}
	return requireOneRow(result, "order", orderUUID)
}

func insertOrderProducts(tx *sql.Tx, order Order) error {
	if _, err := tx.Exec(`DELETE FROM order_products WHERE order_uuid = ?`, order.UUID); err != nil {
		return err
	}
	for _, item := range order.Products {
		if _, err := tx.Exec(`INSERT INTO order_products (order_uuid, product_uuid, product_id, product_name, product_price, quantity) VALUES (?, ?, ?, ?, ?, ?)`, order.UUID, item.Product.UUID, item.Product.Id, item.Product.Name, item.Product.Price, item.Quantity); err != nil {
			return err
		}
	}
	return nil
}

func requireOneRow(result sql.Result, entity, id string) error {
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("%s not found: %s", entity, id)
	}
	return nil
}

func resultError(result sql.Result, err error, entity, id string) error {
	if err != nil {
		return err
	}
	return requireOneRow(result, entity, id)
}
