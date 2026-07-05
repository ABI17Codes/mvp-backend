package models

import "github.com/google/uuid"

type Category struct {
	BaseModel
	UserID  uuid.UUID `json:"userID" gorm:"type:uuid;not null"`
	StoreID uuid.UUID `json:"storeID" gorm:"type:uuid;not null;uniqueIndex:idx_store_category_name"`
	Name    string    `json:"name" gorm:"not null;uniqueIndex:idx_store_category_name"`

	// StoreID uuid.UUID `json:"storeID" gorm:"type:uuid;not null"`
	// Name string `json:"name"`

	// Products []Product `json:"products"`
}
