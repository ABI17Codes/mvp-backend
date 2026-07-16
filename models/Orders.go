package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)
type Orders struct {
	BaseModel
	UserID    uuid.UUID `json:"userID" gorm:"type:uuid;not null"`
	StoreID   uuid.UUID `json:"storeID" gorm:"type:uuid;not null;"`


	// Customer details
	UserName string    `json:"userName" gorm:"not null;"`
	Number   string    `json:"number" gorm:"type:text"`
	Address  string    `json:"address" gorm:"type:text"`

	// Order Items (Cart)
	OrderItems datatypes.JSON `json:"orderItems" gorm:"type:jsonb"`

	// Pricing
	Subtotal       int `json:"subtotal" gorm:"default:0"`
	DeliveryCharge int `json:"deliveryCharge" gorm:"default:0"`
	TotalAmount    int `json:"totalAmount" gorm:"default:0"`

	// Deprecated fields (kept for backward compatibility during migration)
	ProductID   uuid.UUID `json:"productID" gorm:"type:uuid"`
	ProductName string    `json:"productName"`
	Price       int       `json:"price" gorm:"default:0"`
	Quantity    int       `json:"quantity" gorm:"default:1"`
	
    // Order management
    Status      string    `json:"status" gorm:"default:'Pending Payment'"`
    // Pending Payment | Confirmed | Preparing | Ready | Delivered | Cancelled

	// Payment management
	PaymentMethod      string `json:"paymentMethod" gorm:"default:'UPI'"`
	PaymentStatus      string `json:"paymentStatus" gorm:"default:'Pending'"`
	// Pending | Paid | Failed
	PaymentReference   string `json:"paymentReference"`
	PaymentScreenshot  string `json:"paymentScreenshot"`
    
    OrderNumber string    `json:"orderNumber" gorm:"unique"`
    // Example: ORD-2024-0001

	Note   string `json:"note" gorm:"type:text"`
}
