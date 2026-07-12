package models

import "gorm.io/datatypes"

const (
	PlanFree   = "free"
	PlanLaunch = "launch"
	PlanGrowth = "growth"
	PlanScale  = "scale"
)

type Plan struct {
	BaseModel

	Name        string `json:"name" gorm:"unique;not null"` // Free, Starter, Pro
	Description string `json:"description"`

	Price       float64 `json:"price" gorm:"default:0"`
	OfferPrice  float64 `json:"offer_price" gorm:"default:0"`
	IsOfferActive bool  `json:"is_offer_active" gorm:"default:false"`
	DurationDay int     `json:"duration_day"`

	ProductLimit      int `json:"product_limit"`
	MaxStaff          int `json:"max_staff"`
	MonthlyOrderLimit int `json:"monthly_order_limit"`

	Features datatypes.JSON `json:"features"`

	IsActive bool `json:"is_active" gorm:"default:true"`
}
