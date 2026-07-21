package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	PaymentPending  = "pending"
	PaymentApproved = "approved"
	PaymentRejected = "rejected"
)

type Payment struct {
	BaseModel

	UserID uuid.UUID `json:"userID" gorm:"type:uuid;not null"`
	User   User      `json:"user" gorm:"foreignKey:UserID"`

	StoreID uuid.UUID `json:"storeID" gorm:"type:uuid;not null"`
	Store   Store     `json:"store" gorm:"foreignKey:StoreID"`

	PlanID uuid.UUID `json:"planID" gorm:"type:uuid;not null"`
	Plan   Plan      `json:"plan" gorm:"foreignKey:PlanID"`

	Amount        float64 `json:"amount" gorm:"type:numeric;not null"`
	PaymentMethod string  `json:"paymentMethod" gorm:"default:UPI"`

	TransactionID string `json:"transactionID" gorm:"not null"`
	Screenshot    string `json:"screenshot"`

	Status  string `json:"status" gorm:"default:pending"` // pending, approved, rejected
	Remarks string `json:"remarks"`
	Months  int    `json:"months" gorm:"default:1"`

	IsRefunded    bool   `json:"isRefunded" gorm:"default:false"`
	RefundRemarks string `json:"refundRemarks"`
	StoreDeleted  bool   `json:"storeDeleted" gorm:"-"`

	ApprovedBy *uuid.UUID `json:"approved_by" gorm:"type:uuid"`
	ApprovedAt *time.Time `json:"approved_at"`
}
