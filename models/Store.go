package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
)

type Store struct {
	BaseModel
	UserID uuid.UUID   `json:"userID"  gorm:"type:uuid;not null;uniqueIndex"`

	Name        string `json:"storeName"  gorm:"not null"`
	Description string `json:"description"`
	Number      string `json:"number"  gorm:"not null"`

	FacebookURL string `json:"facebookurl"`
	InstaURL    string `json:"instaurl"`
	MapLink     string `json:"maplink"`

	Address []Address `json:"address" gorm:"foreignKey:storeID"`
	SEO     SEO       `json:"seo" gorm:"foreignKey:storeID"`

	DeliveryCharge float64 `json:"deliveryCharge" gorm:"default:0"`

	UpiId   string    `json:"upiId"`
	UpiName string    `json:"upiName"`

	PrivacyPolicy   string `json:"privacyPolicy"   gorm:"type:text"`
	TermsConditions string `json:"termsConditions" gorm:"type:text"`

	Products      []Product      `json:"products" gorm:"foreignKey:storeID"`
	Logo          pq.StringArray `json:"logo"           gorm:"type:text[]"`
	BannerImages  pq.StringArray `json:"bannerimages"   gorm:"type:text[]"`
	Slug          string         `json:"slug"           gorm:"uniqueIndex;not null"`
	StoreTemplate string         `json:"store_template" gorm:"type:varchar(50);default:'classic'"`
	
	Plan               string         `json:"plan"           gorm:"-"`
	PlanFeatures       datatypes.JSON `json:"planFeatures"   gorm:"-"`
	SubscriptionStatus string         `json:"subscriptionStatus" gorm:"-"`
	SubscriptionExpiry string         `json:"subscriptionExpiry" gorm:"-"`
	ExtraOrderLimit    int            `json:"extraOrderLimit" gorm:"default:0"`
	ExtraOrdersExpiry  *time.Time     `json:"extraOrdersExpiry"`


	IsActive bool `gorm:"default:true"`
	IsSuspended   bool           `json:"isSuspended"    gorm:"default:false"`
}
