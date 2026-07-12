package models

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionUsage struct {
	BaseModel

	SubscriptionID uuid.UUID `json:"subscriptionID" gorm:"type:uuid;not null"`
	Subscription   Subscription `json:"subscription" gorm:"foreignKey:SubscriptionID"`

	StoreID uuid.UUID `json:"storeID" gorm:"type:uuid;not null"`

	BillingPeriodStart time.Time `json:"billingPeriodStart"`
	BillingPeriodEnd   time.Time `json:"billingPeriodEnd"`

	OrdersUsed int `json:"ordersUsed" gorm:"default:0"`
}
