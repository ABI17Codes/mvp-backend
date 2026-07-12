package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	SubscriptionActive    = "active"
	SubscriptionExpired   = "expired"
	SubscriptionCancelled = "cancelled"
	SubscriptionPending   = "pending"
	SubscriptionRejected  = "rejected"
	SubscriptionSuspended = "suspended"
)

type Subscription struct {
	BaseModel

	UserID  uuid.UUID `json:"userID" gorm:"type:uuid;not null"`
	StoreID uuid.UUID `json:"storeID" gorm:"type:uuid;not null"`

	PlanID uuid.UUID `json:"planID" gorm:"type:uuid;not null"`
	Plan   Plan      `json:"plan" gorm:"foreignKey:PlanID"`

	Status string `json:"status" gorm:"default:pending"` // pending, active, expired, cancelled, rejected, suspended

	StartDate  time.Time `json:"startDate"`
	ExpiryDate time.Time `json:"expiryDate"`

}