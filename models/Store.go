package models

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
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

	Products      []Product      `json:"products" gorm:"foreignKey:storeID"`
	Logo          pq.StringArray `json:"logo"           gorm:"type:text[]"`
	BannerImages  pq.StringArray `json:"bannerimages"   gorm:"type:text[]"`
	Slug          string         `json:"slug"           gorm:"uniqueIndex;not null"`
	StoreTemplate string         `json:"store_template" gorm:"type:varchar(50);default:'classic'"`
}
