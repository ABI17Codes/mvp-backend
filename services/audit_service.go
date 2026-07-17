package services

import (
	"backend/db"
	"backend/models"
	"encoding/json"
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// Actions
const (
	ActionCreateProduct   = "CREATE_PRODUCT"
	ActionUpdateProduct   = "UPDATE_PRODUCT"
	ActionDeleteProduct   = "DELETE_PRODUCT"
	ActionCreateOrder     = "CREATE_ORDER"
	ActionUpdateOrder     = "UPDATE_ORDER"
	ActionLogin           = "LOGIN"
	ActionLogout          = "LOGOUT"
	ActionRegister        = "REGISTER"
	ActionPaymentApproved = "PAYMENT_APPROVED"
	ActionPaymentRejected = "PAYMENT_REJECTED"
	ActionUpgradeRequested = "UPGRADE_REQUESTED"
	ActionCreateStore     = "CREATE_STORE"
	ActionUpdateStore     = "UPDATE_STORE"
	ActionDeleteStore     = "DELETE_STORE"
	ActionCreateUser      = "CREATE_USER"
	ActionUpdateUserRole  = "UPDATE_USER_ROLE"
	ActionUpdateUserPass  = "UPDATE_USER_PASSWORD"
	ActionDeleteUser      = "DELETE_USER"
)

// Resources
const (
	ResourceUser         = "USER"
	ResourceStore        = "STORE"
	ResourceProduct      = "PRODUCT"
	ResourceOrder        = "ORDER"
	ResourceSubscription = "SUBSCRIPTION"
	ResourcePayment      = "PAYMENT"
	ResourceConfig       = "CONFIG"
)

type Activity struct {
	Action      string
	Resource    string
	ResourceID  *uuid.UUID
	Description string
	Success     bool
	Metadata    interface{}
}

func Log(c fiber.Ctx, act Activity) {
	// Parse user_id from context
	userIDStr, ok := c.Locals("userID").(string)
	var userID *uuid.UUID
	if ok && userIDStr != "" {
		parsed, err := uuid.Parse(userIDStr)
		if err == nil {
			userID = &parsed
		}
	}

	// Parse store_id from context (if set, else we can lookup if store owner has one)
	var tenantID *uuid.UUID
	storeIDStr, ok := c.Locals("storeID").(string)
	if ok && storeIDStr != "" {
		parsed, err := uuid.Parse(storeIDStr)
		if err == nil {
			tenantID = &parsed
		}
	}

	// Extract IP Address and User-Agent
	ipAddress := c.IP()
	if xff := c.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		ipAddress = strings.TrimSpace(ips[0])
	}
	
	if len(ipAddress) > 45 {
		ipAddress = ipAddress[:45]
	}
	userAgent := c.Get("User-Agent")
	method := c.Method()
	endpoint := c.Path()

	// Serialize Metadata
	var metadataStr string
	if act.Metadata != nil {
		bytes, err := json.Marshal(act.Metadata)
		if err == nil {
			metadataStr = string(bytes)
		}
	}

	auditLog := models.AuditLog{
		UserID:      userID,
		TenantID:    tenantID,
		Action:      act.Action,
		Resource:    act.Resource,
		ResourceID:  act.ResourceID,
		Description: act.Description,
		Method:      method,
		Endpoint:    endpoint,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Success:     act.Success,
		Metadata:    metadataStr,
	}

	// Write to database in background
	go func() {
		if err := db.DB.Create(&auditLog).Error; err != nil {
			log.Printf("Failed to create audit log in DB: %v", err)
		}
	}()
}
