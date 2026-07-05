package controller

import (
	"backend/db"
	"backend/models"
	"backend/requests"
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

	// return c.Status(201).JSON(storeReq)
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
	if storeReq.StoreTemplate != "" {
		store.StoreTemplate = storeReq.StoreTemplate
	}

	if err := db.DB.Save(&store).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not update store",
			"error":   err.Error(),
		})
	}

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

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Store fetched successfully",
		"data":    store,
	})
}
