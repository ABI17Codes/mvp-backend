package requests

type CreateSeoRequest struct { 
	Title       string   `json:"title" validate:"required"`
	Description string   `json:"description" validate:"required"`
	Keywords    string   `json:"keywords" validate:"required"`
	OgImageUrl  []string `json:"ogimageurl" validate:"required"`
}

type UpdateSeoRequest struct { 
	Title       string   `json:"title" validate:"required"`
	Description string   `json:"description" validate:"required"`
	Keywords    string   `json:"keywords" validate:"required"`
	OgImageUrl  []string `json:"ogimageurl" validate:"required"`
}
