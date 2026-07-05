package models

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type SEO struct {
	BaseModel
	UserID      uuid.UUID      `json:"userID" gorm:"type:uuid;not null"`
	StoreID     uuid.UUID      `json:"storeID" gorm:"type:uuid;not null;uniqueIndex"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Keywords    string         `json:"keywords"`
	OgImageUrl  pq.StringArray `json:"ogimageurl" gorm:"type:text[]"`
	// OgImageUrl  []string  `json:"ogimageurl" gorm:"type:text[]"`
}
