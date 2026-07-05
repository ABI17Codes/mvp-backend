package requests

type CreateOrderRequest struct {
	// Customer details
	UserName string `json:"userName" validate:"required"`
	Number   string `json:"number" validate:"required"`
	Address  string `json:"address" validate:"required"`

	// Product details
	ProductID   string `json:"productID" validate:"required"`
	ProductName string `json:"productName" validate:"required"`
	Price       int    `json:"price" validate:"required,min=1"`
	Quantity    int    `json:"quantity" validate:"required,min=1"`

	// Store
	StoreID string `json:"storeID" validate:"required"`

	// Optional
	Note string `json:"note" validate:"required"`
}

type UpdateOrderRequest struct {
	// Customer details
	UserName string `json:"userName" validate:"required"`
	Number   string `json:"number" validate:"required"`
	Address  string `json:"address" validate:"required"`

	// Product details
	ProductID   string `json:"productID" validate:"required"`
	ProductName string `json:"productName" validate:"required"`
	Price       int    `json:"price" validate:"required,min=1"`
	Quantity    int    `json:"quantity" validate:"required,min=1"`

	// Store
	StoreID string `json:"storeID" validate:"required"`

	// Optional
	Note string `json:"note" validate:"required"`
}
