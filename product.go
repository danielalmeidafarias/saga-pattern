package main

// ProductService implements IProductService.
type ProductService struct{}

func (*ProductService) Create(productId string) (Product, error) {
	return Product{}, nil
}

func (*ProductService) Get(productUUID string) (Product, error) {
	return Product{}, nil
}
