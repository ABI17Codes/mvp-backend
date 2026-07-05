package models

import "github.com/google/uuid"

type Address struct {
	BaseModel
	UserID     uuid.UUID `json:"userID" gorm:"type:uuid;not null"`
	StoreID    uuid.UUID `json:"storeID" gorm:"type:uuid;not null;uniqueIndex"`
	Street     string    `json:"street"`
	City       string    `json:"city"`
	State      string    `json:"state"`
	Country    string    `json:"country"`
	PostalCode string    `json:"postalcode"`
	IsDefault  bool      `json:"isdefault"`
}
