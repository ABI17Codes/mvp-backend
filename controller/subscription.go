package controller

import (
	"backend/db"
	"backend/models"
	"backend/services"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UpgradeInput struct {
	PlanID        string `json:"planID"`
	TransactionID string `json:"transactionID"`
	Screenshot    string `json:"screenshot"`
	Months        int    `json:"months"`
}

type VerifyInput struct {
	Action  string `json:"action"` // "approve" or "reject"
	Remarks string `json:"remarks"`
}

func GetPlans(c fiber.Ctx) error {
	var plans []models.Plan
	if err := db.DB.Where("is_active = ?", true).Order("price asc").Find(&plans).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch plans",
		})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    plans,
	})
}

func GetMySubscription(c fiber.Ctx) error {
	userIDStr, ok := c.Locals("userID").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User not authenticated",
		})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid user ID",
		})
	}

	var store models.Store
	if err := db.DB.Where("user_id = ?", userID).First(&store).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"message": "Store not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not lookup store",
		})
	}

	var activeProductsCount int64
	db.DB.Model(&models.Product{}).Where("store_id = ? AND is_active = ?", store.ID, true).Count(&activeProductsCount)

	var sub models.Subscription
	err = db.DB.Preload("Plan").Where("store_id = ?", store.ID).Order("created_at desc").First(&sub).Error
	if err == nil {
		var usage models.SubscriptionUsage
		now := time.Now()
		db.DB.Where("subscription_id = ? AND billing_period_start <= ? AND billing_period_end >= ?", sub.ID, now, now).First(&usage)

		var pendingAddon bool
		var pendingReq models.Payment
		if err := db.DB.Preload("Plan").Where("store_id = ? AND status = ?", store.ID, models.PaymentPending).First(&pendingReq).Error; err == nil {
			name := strings.ToLower(pendingReq.Plan.Name)
			if strings.Contains(name, "addon") || strings.Contains(name, "add-on") {
				pendingAddon = true
			}
		}

		extraLimit := 0
		if store.ExtraOrdersExpiry != nil && time.Now().Before(*store.ExtraOrdersExpiry) {
			extraLimit = store.ExtraOrderLimit
		}

		return c.JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"id": sub.ID,
				"userID": sub.UserID,
				"storeID": sub.StoreID,
				"planID": sub.PlanID,
				"plan": sub.Plan,
				"status": sub.Status,
				"startDate": sub.StartDate,
				"expiryDate": sub.ExpiryDate,
				"pendingAddon": pendingAddon,
				"usage": fiber.Map{
					"products_active": activeProductsCount,
					"orders_used": usage.OrdersUsed,
					"extra_orders": extraLimit,
				},
			},
		})
	}

	// Default to free plan if none exists
	var freePlan models.Plan
	if err := db.DB.Where("name = ?", "free").First(&freePlan).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Plans database is not seeded",
		})
	}

	var ordersUsed int64
	startOfMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)
	db.DB.Model(&models.Orders{}).Where("store_id = ? AND created_at >= ?", store.ID, startOfMonth).Count(&ordersUsed)

	extraLimit := 0
	if store.ExtraOrdersExpiry != nil && time.Now().Before(*store.ExtraOrdersExpiry) {
		extraLimit = store.ExtraOrderLimit
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"status":      "active",
			"plan_id":     freePlan.ID,
			"plan":        freePlan,
			"startDate":   store.CreatedAt,
			"expiryDate":  store.CreatedAt.AddDate(10, 0, 0), // Mock long duration
			"usage": fiber.Map{
				"products_active": activeProductsCount,
				"orders_used": ordersUsed,
				"extra_orders": extraLimit,
			},
		},
	})
}

func UpgradeSubscription(c fiber.Ctx) error {
	userIDStr, ok := c.Locals("userID").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User not authenticated",
		})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid user ID",
		})
	}

	input := new(UpgradeInput)
	if err := c.Bind().Body(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	if input.PlanID == "" || (input.TransactionID == "" && input.Screenshot == "") {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Plan ID, and either Transaction ID or payment screenshot proof is required",
		})
	}

	planUUID, err := uuid.Parse(input.PlanID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid Plan ID format",
		})
	}

	var plan models.Plan
	if err := db.DB.First(&plan, "id = ?", planUUID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Selected plan not found",
		})
	}

	var store models.Store
	if err := db.DB.Where("user_id = ?", userID).First(&store).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Store not found for this user",
		})
	}

	// 0. Prevent upgrade if subscription is paused
	var activeSub models.Subscription
	if errSub := db.DB.Where("store_id = ? AND status = ?", store.ID, models.SubscriptionSuspended).First(&activeSub).Error; errSub == nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "Cannot apply for new subscriptions or upgrades while your current subscription is suspended. Please contact support.",
		})
	}

	// 1. Prevent duplicate pending requests
	var pendingReq models.Payment
	if err := db.DB.Where("store_id = ? AND status = ?", store.ID, models.PaymentPending).First(&pendingReq).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"message": "You already have a pending verification request. Please wait for the admin's approval.",
		})
	}

	// Initialize database transaction
	tx := db.DB.Begin()

	months := input.Months
	if months <= 0 {
		months = 1
	}

	payReq := models.Payment{
		UserID:        userID,
		StoreID:       store.ID,
		PlanID:        plan.ID,
		Amount:        plan.Price * float64(months), // strictly resolve price from DB and multiply by months
		PaymentMethod: "UPI",
		TransactionID: input.TransactionID,
		Screenshot:    input.Screenshot,
		Status:        models.PaymentPending,
		Months:        months,
	}

	if err := tx.Create(&payReq).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to create payment request",
		})
	}

	planNameLower := strings.ToLower(plan.Name)
	if strings.Contains(planNameLower, "addon") || strings.Contains(planNameLower, "add-on") {
		if err := tx.Commit().Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Failed to commit transaction",
			})
		}
		services.Log(c, services.Activity{
			Action:      services.ActionUpgradeRequested,
			Resource:    services.ResourcePayment,
			ResourceID:  &payReq.ID,
			Description: "Requested purchase of Add-on " + plan.Name,
			Success:     true,
			Metadata: fiber.Map{
				"plan_name":      plan.Name,
				"resolved_price": plan.Price,
				"transaction_id": payReq.TransactionID,
			},
		})

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"success": true,
			"message": "Add-on purchase request submitted successfully. Awaiting verification.",
			"data":    payReq,
		})
	}

	var sub models.Subscription
	err = tx.Where("store_id = ?", store.ID).Order("created_at desc").First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sub = models.Subscription{
				UserID:  userID,
				StoreID: store.ID,
				PlanID:  plan.ID,
				Status:  models.SubscriptionPending,
			}
			if err := tx.Create(&sub).Error; err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"message": "Failed to initialize subscription row",
				})
			}
		} else {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Failed to query subscriptions",
			})
		}
	} else {
		sub.PlanID = plan.ID
		sub.Status = models.SubscriptionPending
		if err := tx.Save(&sub).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Failed to update subscription row",
			})
		}
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to commit transaction",
		})
	}

	// Audit log
	services.Log(c, services.Activity{
		Action:      services.ActionUpgradeRequested,
		Resource:    services.ResourceSubscription,
		ResourceID:  &sub.ID,
		Description: "Requested upgrade to plan " + plan.Name,
		Success:     true,
		Metadata: fiber.Map{
			"plan_name":      plan.Name,
			"resolved_price": plan.Price,
			"transaction_id": payReq.TransactionID,
		},
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Upgrade request submitted successfully. Awaiting verification.",
		"data":    payReq,
	})
}

func AdminGetPayments(c fiber.Ctx) error {
	status := c.Query("status")
	query := db.DB.Preload("User").Preload("Store").Preload("Plan")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var requests []models.Payment
	if err := query.Order("created_at desc").Find(&requests).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to retrieve payment requests",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    requests,
	})
}

func AdminVerifyPayment(c fiber.Ctx) error {
	adminIDStr, ok := c.Locals("userID").(string)
	if !ok || adminIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Admin user not authenticated",
		})
	}
	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid admin user ID",
		})
	}

	reqIDStr := c.Params("id")
	reqUUID, err := uuid.Parse(reqIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request ID format",
		})
	}

	input := new(VerifyInput)
	if err := c.Bind().Body(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	if input.Action != "approve" && input.Action != "reject" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Action must be either 'approve' or 'reject'",
		})
	}

	var payReq models.Payment
	if err := db.DB.Preload("Plan").First(&payReq, "id = ?", reqUUID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"message": "Payment request not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to look up request",
		})
	}

	if payReq.Status != models.PaymentPending {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"message": "This request has already been processed and is in " + payReq.Status + " status.",
		})
	}

	tx := db.DB.Begin()

	now := time.Now()
	payReq.ApprovedBy = &adminID
	payReq.ApprovedAt = &now
	payReq.Remarks = input.Remarks

	var sub models.Subscription
	if err := tx.Where("store_id = ?", payReq.StoreID).Order("created_at desc").First(&sub).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Subscription for the target store not found",
		})
	}

	var auditAction string
	var auditDescription string

	if input.Action == "approve" {
		payReq.Status = models.PaymentApproved

		payReqPlanNameLower := strings.ToLower(payReq.Plan.Name)
		if strings.Contains(payReqPlanNameLower, "addon") || strings.Contains(payReqPlanNameLower, "add-on") {
			var store models.Store
			if err := tx.First(&store, "id = ?", payReq.StoreID).Error; err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"message": "Store not found",
				})
			}
			if store.ExtraOrdersExpiry != nil && time.Now().After(*store.ExtraOrdersExpiry) {
				store.ExtraOrderLimit = 0
			}
			store.ExtraOrderLimit += 1000 // Currently hardcoded to 1000
			
			now := time.Now()
			endOfMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
			store.ExtraOrdersExpiry = &endOfMonth

			if err := tx.Save(&store).Error; err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"message": "Failed to update store extra order limit",
				})
			}
			auditAction = services.ActionPaymentApproved
			auditDescription = "Approved manual payment for Extra Orders Add-on (+1000)"
		} else {
			var baseTime time.Time
			var startDate time.Time
			if sub.Status == models.SubscriptionActive && sub.PlanID == payReq.PlanID && sub.ExpiryDate.After(now) {
				baseTime = sub.ExpiryDate
				startDate = sub.ExpiryDate
			} else {
				baseTime = now
				startDate = now
			}

			monthsToAdd := payReq.Months
			if monthsToAdd <= 0 {
				monthsToAdd = 1
			}
			daysToAdd := payReq.Plan.DurationDay * monthsToAdd
			expiryDate := baseTime.AddDate(0, 0, daysToAdd)

			// Create a new subscription record
			newSub := models.Subscription{
				UserID:     sub.UserID,
				StoreID:    sub.StoreID,
				PlanID:     payReq.PlanID,
				Status:     models.SubscriptionActive,
				StartDate:  startDate,
				ExpiryDate: expiryDate,
			}

			// Mark old subscription as renewed/archived to maintain history
			sub.Status = "renewed"
			if err := tx.Save(&sub).Error; err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"message": "Failed to archive old subscription",
				})
			}

			if err := tx.Create(&newSub).Error; err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"message": "Failed to create new subscription details",
				})
			}

			auditAction = services.ActionPaymentApproved
			auditDescription = "Approved manual payment for store upgrade to plan " + payReq.Plan.Name
		}
	} else {
		payReq.Status = models.PaymentRejected

		sub.Status = models.SubscriptionRejected
		if err := tx.Save(&sub).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Failed to update subscription details",
			})
		}

		auditAction = services.ActionPaymentRejected
		auditDescription = "Rejected manual payment for store upgrade to plan " + payReq.Plan.Name
	}

	if err := tx.Save(&payReq).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to update payment request status",
		})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to commit verification transaction",
		})
	}

	// Log audit event
	services.Log(c, services.Activity{
		Action:      auditAction,
		Resource:    services.ResourcePayment,
		ResourceID:  &payReq.ID,
		Description: auditDescription,
		Success:     true,
		Metadata: fiber.Map{
			"remarks":        input.Remarks,
			"payment_req_id": payReq.ID.String(),
			"plan_name":      payReq.Plan.Name,
		},
	})

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Payment request has been successfully " + payReq.Status,
		"data":    payReq,
	})
}

func GetUPIConfig(c fiber.Ctx) error {
	var upiID models.Config
	var upiName models.Config
	
	db.DB.First(&upiID, "key = ?", "upi_id")
	db.DB.First(&upiName, "key = ?", "upi_name")

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"upi_id":   upiID.Value,
			"upi_name": upiName.Value,
		},
	})
}

type UPIConfigInput struct {
	UPIID   string `json:"upi_id"`
	UPIName string `json:"upi_name"`
}

func UpdateUPIConfig(c fiber.Ctx) error {
	input := new(UPIConfigInput)
	if err := c.Bind().Body(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	input.UPIID = strings.TrimSpace(input.UPIID)
	input.UPIName = strings.TrimSpace(input.UPIName)

	if input.UPIID == "" || input.UPIName == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "UPI ID and UPI name are required",
		})
	}

	db.DB.Model(&models.Config{}).Where("key = ?", "upi_id").Update("value", input.UPIID)
	db.DB.Model(&models.Config{}).Where("key = ?", "upi_name").Update("value", input.UPIName)

	services.Log(c, services.Activity{
		Action:      "UPDATE_UPI_CONFIG",
		Resource:    "SYSTEM_CONFIG",
		Description: "Admin updated platform UPI config to ID: " + input.UPIID + ", Name: " + input.UPIName,
		Success:     true,
		Metadata:    fiber.Map{"upi_id": input.UPIID, "upi_name": input.UPIName},
	})

	return c.JSON(fiber.Map{
		"success": true,
		"message": "UPI config updated successfully",
	})
}

func AdminGetPlans(c fiber.Ctx) error {
	var plans []models.Plan
	if err := db.DB.Order("price asc").Find(&plans).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch plans for administration",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    plans,
	})
}

type UpdatePlanInput struct {
	Price         *float64 `json:"price"`
	OfferPrice    *float64 `json:"offer_price"`
	IsOfferActive *bool    `json:"is_offer_active"`
	Description   string   `json:"description"`
}

func AdminUpdatePlan(c fiber.Ctx) error {
	id := c.Params("id")
	planUUID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid plan ID format"})
	}

	input := new(UpdatePlanInput)
	if err := c.Bind().Body(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid request body"})
	}

	var plan models.Plan
	if err := db.DB.First(&plan, "id = ?", planUUID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "Plan not found"})
	}

	if input.Price != nil {
		plan.Price = *input.Price
	}
	if input.OfferPrice != nil {
		plan.OfferPrice = *input.OfferPrice
	}
	if input.IsOfferActive != nil {
		plan.IsOfferActive = *input.IsOfferActive
	}
	plan.Description = strings.TrimSpace(input.Description)

	if err := db.DB.Save(&plan).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "Failed to save plan details"})
	}

	services.Log(c, services.Activity{
		Action:      "UPDATE_PLAN",
		Resource:    "PLAN",
		ResourceID:  &plan.ID,
		Description: "Admin updated plan details for " + plan.Name,
		Success:     true,
		Metadata:    fiber.Map{"name": plan.Name, "price": plan.Price, "description": plan.Description},
	})

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Plan updated successfully",
		"data":    plan,
	})
}

type CreatePlanInput struct {
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	OfferPrice    float64 `json:"offer_price"`
	IsOfferActive bool    `json:"is_offer_active"`
	Description   string  `json:"description"`
}

func AdminCreatePlan(c fiber.Ctx) error {
	input := new(CreatePlanInput)
	if err := c.Bind().Body(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid request body"})
	}

	plan := models.Plan{
		Name:          strings.ToLower(strings.TrimSpace(input.Name)),
		Price:         input.Price,
		OfferPrice:    input.OfferPrice,
		IsOfferActive: input.IsOfferActive,
		Description:   strings.TrimSpace(input.Description),
		IsActive:      true,
		DurationDay:   30, // Default to 30 days
	}

	if err := db.DB.Create(&plan).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "Failed to create plan"})
	}

	services.Log(c, services.Activity{
		Action:      "CREATE_PLAN",
		Resource:    "PLAN",
		ResourceID:  &plan.ID,
		Description: "Admin created a new plan: " + plan.Name,
		Success:     true,
		Metadata:    fiber.Map{"name": plan.Name, "price": plan.Price},
	})

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Plan created successfully",
		"data":    plan,
	})
}
