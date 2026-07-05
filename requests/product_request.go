package requests

type CreateProductRequest struct {
	Name         string   `json:"productName" validate:"required"`
	Description  string   `json:"description" validate:"required"`
	Price        int      `json:"price" validate:"required"`
	OfferPrice   int      `json:"offerprice"`
	OutOfStock   bool     `json:"outofstock"`
	IsFeatured   bool     `json:"is_featured"`
	IsNewArrival bool     `json:"is_new_arrival"`
	Images       []string `json:"images" validate:"required"`
	CategoryID   string   `json:"categoryID" validate:"required"`
}

type UpdateProductRequest struct {
	Name         string   `json:"productName" validate:"required"`
	Description  string   `json:"description" validate:"required"`
	Price        int      `json:"price" validate:"required"`
	OfferPrice   int      `json:"offerprice"`
	OutOfStock   bool     `json:"outofstock"`
	IsFeatured   bool     `json:"is_featured"`
	IsNewArrival bool     `json:"is_new_arrival"`
	Images       []string `json:"images" validate:"required"`
	CategoryID   string   `json:"categoryID" validate:"required"`
}
