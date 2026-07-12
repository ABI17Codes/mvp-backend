package models

import (
	"github.com/google/uuid"
)

type AuditLog struct {
	BaseModel

	UserID     *uuid.UUID `json:"user_id" gorm:"type:uuid"`
	User       *User      `json:"user" gorm:"foreignKey:UserID"`
	TenantID   *uuid.UUID `json:"tenant_id" gorm:"type:uuid"`
	Action     string     `json:"action" gorm:"type:varchar(50);not null"`
	Resource   string     `json:"resource" gorm:"type:varchar(50);not null"`
	ResourceID *uuid.UUID `json:"resource_id" gorm:"type:uuid"`

	Description string `json:"description"`
	Method      string `json:"method" gorm:"type:varchar(10)"`
	Endpoint    string `json:"endpoint"`
	IPAddress   string `json:"ip_address" gorm:"type:varchar(45)"`
	UserAgent   string `json:"user_agent"`
	Success     bool   `json:"success" gorm:"default:true"`
	Metadata    string `json:"metadata" gorm:"type:text"`
}
