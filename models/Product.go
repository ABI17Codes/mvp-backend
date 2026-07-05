package models

import "github.com/google/uuid"

type Product struct {
	BaseModel
	UserID      uuid.UUID `json:"userID" gorm:"type:uuid;not null"`
	StoreID     uuid.UUID `json:"storeID" gorm:"type:uuid;not null;uniqueIndex:idx_store_product_name"`
	Name        string    `json:"productName" gorm:"not null;uniqueIndex:idx_store_product_name"`
	Description string    `json:"description" gorm:"type:text"`
	Price       int       `json:"price" gorm:"default:0"`
	OfferPrice  int       `json:"offerprice"`
	OutOfStock   bool      `json:"outofstock"`
	IsFeatured   bool      `json:"is_featured"`
	IsNewArrival bool      `json:"is_new_arrival"`
	Images       []string  `json:"images" gorm:"serializer:json"`
	CategoryID   uuid.UUID `json:"categoryID"   gorm:"not null"`

	// StoreID     uuid.UUID `json:"storeID" gorm:"type:uuid;not null"`
	// Name        string    `json:"productName" gorm:"not null"`

	// Store       Store     `json:"store" gorm:"foreignKey:StoreID"`
	// Category    Category  `json:"category"      gorm:"foreignKey:CategoryID"`
}
