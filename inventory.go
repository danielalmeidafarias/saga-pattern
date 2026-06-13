package main

// InventoryService implements IInventoryService.
type InventoryService struct{}

func (*InventoryService) Create(inventoryId string, stock int) (Inventory, error) {
	return Inventory{}, nil
}

func (*InventoryService) Get(inventoryId string) (Inventory, error) {
	return Inventory{}, nil
}

func (*InventoryService) Update(inventoryId string, stock int, virtualStock int) error {
	return nil
}
