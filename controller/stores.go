package controller

import (
	"backend/db"
	"backend/models"
	"backend/requests"
	"backend/services"
	"backend/utils"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func CreateStore(c fiber.Ctx) error {
	var storeReq requests.CreateStoreRequest

	// if err := c.Bind().Body(&store); err != nil {
	// 	return c.Status(400).JSON(fiber.Map{
	// 		"err": "Invalid Request Body",
	// 	})
	// }
	if err := c.Bind().Body(&storeReq); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"err": err.Error(),
		})
	}

	storeReq.Name = strings.TrimSpace(storeReq.Name)
	storeReq.Description = strings.TrimSpace(storeReq.Description)
	storeReq.Number = strings.TrimSpace(storeReq.Number)
	// storeReq.Logo = strings.TrimSpace(storeReq.Logo)
	storeReq.FacebookURL = strings.TrimSpace(storeReq.FacebookURL)
	storeReq.InstaURL = strings.TrimSpace(storeReq.InstaURL)
	storeReq.MapLink = strings.TrimSpace(storeReq.MapLink)

	if storeReq.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Store name is required",
		})
	}

	if storeReq.Number == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Store number is required",
		})
	}

	userIDValue := c.Locals("userID")
	userIDString, ok := userIDValue.(string)
	if !ok || userIDString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User not found in token",
		})
	}

	userID, err := uuid.Parse(userIDString)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Invalid user ID",
		})
	}

	var existingStore models.Store

	if db.DB.Where("user_id = ?", userID).First(&existingStore).RowsAffected > 0 {
		return c.Status(409).JSON(fiber.Map{
			"success": false,
			"message": "This user already has a store",
		})
	}

	slug := strings.TrimSpace(storeReq.Slug)
	if slug == "" {
		slug = strings.ToLower(storeReq.Name)
		slug = strings.ReplaceAll(slug, " ", "-")
		var cleaned []rune
		for _, r := range slug {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
				cleaned = append(cleaned, r)
			}
		}
		slug = string(cleaned)
	}

	var slugCheck models.Store
	if db.DB.Where("slug = ?", slug).First(&slugCheck).RowsAffected > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"message": "Store URL slug is already in use",
		})
	}

	var nameCheck models.Store
	if db.DB.Where("name = ?", storeReq.Name).First(&nameCheck).RowsAffected > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"message": "Store name is already in use",
		})
	}

	storeCreation := models.Store{
		UserID:        userID,
		Name:          storeReq.Name,
		Description:   strings.TrimSpace(storeReq.Description),
		Number:        storeReq.Number,
		Logo:          storeReq.Logo,
		FacebookURL:   strings.TrimSpace(storeReq.FacebookURL),
		InstaURL:      strings.TrimSpace(storeReq.InstaURL),
		MapLink:       strings.TrimSpace(storeReq.MapLink),
		BannerImages:  storeReq.BannerImages,
		Slug:          slug,
		StoreTemplate: "classic",
		UpiId:         storeReq.UpiId,
		UpiName:       storeReq.UpiName,
	}

	tx := db.DB.Begin()

	if err := tx.Create(&storeCreation).Error; err != nil {
		tx.Rollback()
		return c.Status(400).JSON(fiber.Map{
			"err": "Could Not Create Store",
		})
	}

	if strings.TrimSpace(storeReq.Address) != "" {
		addressCreation := models.Address{
			UserID:    userID,
			StoreID:   storeCreation.ID,
			Street:    strings.TrimSpace(storeReq.Address),
			IsDefault: true,
		}
		if err := tx.Create(&addressCreation).Error; err != nil {
			tx.Rollback()
			return c.Status(500).JSON(fiber.Map{
				"success": false,
				"message": "Could not create address for store",
			})
		}
	}

	if err := tx.Model(&models.User{}).
		Where("id = ?", userID).
		Update("role", models.RoleStoreOwner).Error; err != nil {

		tx.Rollback()

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not update user role",
		})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Transaction failed",
		})
	}

	var user models.User
	_ = user // suppress unused warning
	// Generate a new JWT with the updated storeowner role
	token, err := utils.GenerateJWT(userID.String(), models.RoleStoreOwner)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not generate token",
		})
	}

	secure := os.Getenv("APP_ENV") == "production"
	sameSite := "Lax"
	if secure {
		sameSite = "None"
	}

	// Replace the old cookie
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})

	populateStorePlan(&storeCreation)

	services.Log(c, services.Activity{
		Action:      services.ActionCreateStore,
		Resource:    services.ResourceStore,
		ResourceID:  &storeCreation.ID,
		Description: "Created store: " + storeCreation.Name + " with URL slug: " + storeCreation.Slug,
		Success:     true,
		Metadata:    fiber.Map{"name": storeCreation.Name, "slug": storeCreation.Slug},
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Store created successfully",
		"data":    storeCreation,
	})
}

func UpdateStore(c fiber.Ctx) error {
	userIDValue := c.Locals("userID")
	userIDString, ok := userIDValue.(string)
	if !ok || userIDString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User not found in token",
		})
	}

	userID, err := uuid.Parse(userIDString)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid user ID in token",
		})
	}

	var store models.Store
	if err := db.DB.Where("user_id = ?", userID).First(&store).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Store not found for this user",
		})
	}

	var storeReq requests.UpdateStoreRequest
	if err := c.Bind().Body(&storeReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
			"error":   err.Error(),
		})
	}

	slug := strings.TrimSpace(storeReq.Slug)
	if slug != "" && slug != store.Slug {
		var slugCheck models.Store
		if db.DB.Where("slug = ? AND id <> ?", slug, store.ID).First(&slugCheck).RowsAffected > 0 {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"success": false,
				"message": "Store URL slug is already in use by another store",
			})
		}
		store.Slug = slug
	}

	if storeReq.Name != "" && storeReq.Name != store.Name {
		var nameCheck models.Store
		if db.DB.Where("name = ? AND id <> ?", strings.TrimSpace(storeReq.Name), store.ID).First(&nameCheck).RowsAffected > 0 {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"success": false,
				"message": "Store name is already in use by another store",
			})
		}
		store.Name = strings.TrimSpace(storeReq.Name)
	}
	store.Description = strings.TrimSpace(storeReq.Description)
	if storeReq.Number != "" {
		store.Number = strings.TrimSpace(storeReq.Number)
	}
	if storeReq.Logo != nil {
		store.Logo = storeReq.Logo
	}
	store.FacebookURL = strings.TrimSpace(storeReq.FacebookURL)
	store.InstaURL = strings.TrimSpace(storeReq.InstaURL)
	store.MapLink = strings.TrimSpace(storeReq.MapLink)
	if storeReq.BannerImages != nil {
		store.BannerImages = storeReq.BannerImages
	}
	if storeReq.DeliveryCharge != nil {
		store.DeliveryCharge = *storeReq.DeliveryCharge
	}
	store.UpiId = strings.TrimSpace(storeReq.UpiId)
	store.UpiName = strings.TrimSpace(storeReq.UpiName)
	store.PrivacyPolicy = strings.TrimSpace(storeReq.PrivacyPolicy)
	store.TermsConditions = strings.TrimSpace(storeReq.TermsConditions)
	if storeReq.StoreTemplate != "" {
		if storeReq.StoreTemplate == "growth" {
			var sub models.Subscription
			err := db.DB.Preload("Plan").Where("store_id = ? AND status = ? AND expiry_date > ?", store.ID, "active", time.Now()).Order("created_at desc").First(&sub).Error
			isOnFree := false
			if err != nil {
				isOnFree = true
			} else if sub.Plan.Name == "free" {
				isOnFree = true
			}
			if isOnFree {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"success": false,
					"message": "The Growth template is only available on premium plans. Please upgrade your subscription first.",
				})
			}
		}
		store.StoreTemplate = storeReq.StoreTemplate
	}

	if err := db.DB.Save(&store).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not update store",
			"error":   err.Error(),
		})
	}

	populateStorePlan(&store)

	services.Log(c, services.Activity{
		Action:      services.ActionUpdateStore,
		Resource:    services.ResourceStore,
		ResourceID:  &store.ID,
		Description: "Updated store: " + store.Name,
		Success:     true,
		Metadata:    fiber.Map{"name": store.Name, "slug": store.Slug},
	})

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Store updated successfully",
		"data":    store,
	})
}

func GetStoreBySlug(c fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Slug parameter is required",
		})
	}

	var store models.Store
	err := db.DB.
		Preload("Address").
		Preload("SEO").
		Preload("Products").
		Where("slug = ?", slug).
		First(&store).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"message": "Store not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not query store",
			"error":   err.Error(),
		})
	}

	if store.IsSuspended {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "This store has been temporarily suspended by the administrator",
		})
	}

	populateStorePlan(&store)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Store fetched successfully",
		"data":    store,
	})
}

func GetAllStores(c fiber.Ctx) error {
	var stores []models.Store

	if err := db.DB.
		Preload("Address").
		Preload("SEO").
		Preload("Products").
		Find(&stores).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not fetch stores",
		})
	}

	for i := range stores {
		populateStorePlan(&stores[i])
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Stores fetched successfully",
		"count":   len(stores),
		"data":    stores,
	})
}

func GetMyStore(c fiber.Ctx) error {
	userIDValue := c.Locals("userID")
	userIDString, ok := userIDValue.(string)
	if !ok || userIDString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User Not Found In Token",
		})
	}
	userID, err := uuid.Parse(userIDString)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid user ID in token",
		})
	}

	var store models.Store

	err = db.DB.Preload("Address").Preload("SEO").Preload("Products").Where("user_id = ?", userID).First(&store).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Store not found for this user",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not get store",
			"error":   err.Error(),
		})
	}

	populateStorePlan(&store)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Store fetched successfully",
		"data":    store,
	})
}

type AdminUpdateStoreInput struct {
	Name              string `json:"storeName"`
	Slug              string `json:"slug"`
	Number            string `json:"number"`
	Description       string `json:"description"`
	StoreTemplate     string `json:"storeTemplate"`
	PlanName          string `json:"planName"`
	IsSuspended       *bool  `json:"isSuspended"`
	PauseSubscription *bool  `json:"pauseSubscription"`
	ExtraOrderLimit   *int   `json:"extraOrderLimit"`
}

func AdminUpdateStore(c fiber.Ctx) error {
	id := c.Params("id")
	storeUUID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid store ID format"})
	}

	input := new(AdminUpdateStoreInput)
	if err := c.Bind().Body(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid request body"})
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Slug = strings.TrimSpace(input.Slug)
	input.Number = strings.TrimSpace(input.Number)
	input.Description = strings.TrimSpace(input.Description)
	input.StoreTemplate = strings.TrimSpace(input.StoreTemplate)
	input.PlanName = strings.ToLower(strings.TrimSpace(input.PlanName))

	if input.Name == "" || input.Slug == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"success": false, "message": "Store name and Slug are required"})
	}

	var store models.Store
	if err := db.DB.First(&store, "id = ?", storeUUID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "Store not found"})
	}

	// Check slug conflict
	var slugCheck models.Store
	if db.DB.Where("slug = ? AND id <> ?", input.Slug, store.ID).First(&slugCheck).RowsAffected > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"success": false, "message": "Slug is already in use"})
	}

	tx := db.DB.Begin()

	store.Name = input.Name
	store.Slug = input.Slug
	store.Number = input.Number
	store.Description = input.Description
	store.StoreTemplate = input.StoreTemplate
	if input.IsSuspended != nil {
		store.IsSuspended = *input.IsSuspended
	}
	if input.ExtraOrderLimit != nil {
		store.ExtraOrderLimit = *input.ExtraOrderLimit
	}

	if err := tx.Save(&store).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "Failed to update store details"})
	}

	// Update Plan if provided
	if input.PlanName != "" {
		var plan models.Plan
		if err := tx.Where("name = ?", input.PlanName).First(&plan).Error; err == nil {
			var sub models.Subscription
			errSub := tx.Where("store_id = ?", store.ID).First(&sub).Error
			if errSub != nil {
				if errors.Is(errSub, gorm.ErrRecordNotFound) {
					sub = models.Subscription{
						UserID:     store.UserID,
						StoreID:    store.ID,
						PlanID:     plan.ID,
						Status:     models.SubscriptionActive,
						StartDate:  time.Now(),
						ExpiryDate: time.Now().AddDate(0, 0, plan.DurationDay),
					}
					tx.Create(&sub)
				}
			} else {
				sub.PlanID = plan.ID
				sub.Status = models.SubscriptionActive
				sub.StartDate = time.Now()
				sub.ExpiryDate = time.Now().AddDate(0, 0, plan.DurationDay)
				tx.Save(&sub)
			}
		}
	}

	// Process pauseSubscription toggle
	if input.PauseSubscription != nil {
		var sub models.Subscription
		if errSub := tx.Where("store_id = ?", store.ID).Order("created_at desc").First(&sub).Error; errSub == nil {
			if *input.PauseSubscription {
				sub.Status = models.SubscriptionSuspended
			} else {
				sub.Status = "active"
			}
			tx.Save(&sub)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "Failed to commit store update"})
	}

	populateStorePlan(&store)

	services.Log(c, services.Activity{
		Action:      services.ActionUpdateStore,
		Resource:    services.ResourceStore,
		ResourceID:  &store.ID,
		Description: "Admin updated store profile for " + store.Name + " (Plan: " + store.Plan + ")",
		Success:     true,
		Metadata:    fiber.Map{"name": store.Name, "slug": store.Slug, "plan": store.Plan},
	})

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Store updated successfully",
		"data":    store,
	})
}

func AdminDeleteStore(c fiber.Ctx) error {
	id := c.Params("id")
	storeUUID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid store ID format"})
	}

	var store models.Store
	if err := db.DB.First(&store, "id = ?", storeUUID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "Store not found"})
	}

	tx := db.DB.Begin()

	// Soft-delete matching store elements
	tx.Where("store_id = ?", store.ID).Delete(&models.Product{})
	tx.Where("store_id = ?", store.ID).Delete(&models.Category{})
	tx.Where("store_id = ?", store.ID).Delete(&models.Orders{})
	tx.Where("store_id = ?", store.ID).Delete(&models.Subscription{})
	
	if err := tx.Delete(&store).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "Could not delete store"})
	}

	tx.Commit()

	services.Log(c, services.Activity{
		Action:      services.ActionDeleteStore,
		Resource:    services.ResourceStore,
		ResourceID:  &store.ID,
		Description: "Admin deleted store " + store.Name,
		Success:     true,
		Metadata:    fiber.Map{"name": store.Name, "slug": store.Slug},
	})

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Store deleted successfully",
	})
}

func populateStorePlan(store *models.Store) {
	var sub models.Subscription
	err := db.DB.Preload("Plan").Where("store_id = ?", store.ID).Order("created_at desc").First(&sub).Error
	if err == nil {
		store.SubscriptionStatus = sub.Status
		store.SubscriptionExpiry = sub.ExpiryDate.Format(time.RFC3339)
		if sub.Status == "active" {
			store.Plan = sub.Plan.Name
			store.PlanFeatures = sub.Plan.Features
		} else {
			store.Plan = "free"
		}
	} else {
		store.SubscriptionStatus = "none"
		store.SubscriptionExpiry = ""
		store.Plan = "free"
	}
}
