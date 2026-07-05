package main

// InventoryService implements IInventoryService.
type InventoryService struct{}

func (*InventoryService) Create(inventoryId string) error {
	return nil
}

func (*InventoryService) Get(inventoryUUID string) (Inventory, error) {
	return Inventory{}, nil
}

func (*InventoryService) GetProduct(inventoryUUID string, productUUID string) (InventoryProduct, error) {
	return InventoryProduct{}, nil
}

func (*InventoryService) UpdateProduct(inventoryUUID string, productUUID string, stock int, virtualStock int) error {
	return nil
}
