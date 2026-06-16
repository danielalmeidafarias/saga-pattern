package main

// InventoryService implements IInventoryService.
type InventoryService struct{}

func (*InventoryService) Create(productUUID string, stock int) (Inventory, error) {
	return Inventory{}, nil
}

func (*InventoryService) Get(inventoryUUID string) (Inventory, error) {
	return Inventory{}, nil
}

func (*InventoryService) Update(inventoryUUID string, stock int, virtualStock int) error {
	return nil
}
