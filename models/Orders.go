package models

import "github.com/google/uuid"

type Orders struct {
	BaseModel
	UserID    uuid.UUID `json:"userID" gorm:"type:uuid;not null"`
	StoreID   uuid.UUID `json:"storeID" gorm:"type:uuid;not null;"`
	ProductID uuid.UUID `json:"productID" gorm:"type:uuid;not null"`

	// Customer details
	UserName string    `json:"userName" gorm:"not null;"`
	Number   string    `json:"number" gorm:"type:text"`
	Address  string    `json:"address" gorm:"type:text"`

	// Product details
	ProductName string `json:"productName" gorm:"not null;"`
	Price       int    `json:"price" gorm:"default:0"`
	Quantity    int    `json:"quantity" gorm:"default:1"`
	TotalAmount int    `json:"totalAmount" gorm:"default:0"`
	
    // Order management
    Status      string    `json:"status" gorm:"default:'pending'"`
    // pending | confirmed | delivered | cancelled
    
    OrderNumber string    `json:"orderNumber" gorm:"unique"`
    // Example: ORD-2024-0001

	Note   string `json:"note" gorm:"type:text"`
}
