package models

type Config struct {
	Key   string `json:"key" gorm:"primaryKey"`
	Value string `json:"value"`
}
