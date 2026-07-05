package controller

import (
	"backend/db"
	"backend/models"
	"backend/requests"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func CreateSEO(c fiber.Ctx) error {
	storeIDParam := c.Params("storeID")

	storeID, err := uuid.Parse(storeIDParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Invalid store ID",
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
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid user ID in token",
		})
	}

	// var existingSEO models.SEO

	// if db.DB.Where("store_id = ?", storeID).First(&existingSEO).RowsAffected > 0 {
	// 	return c.Status(fiber.StatusConflict).JSON(fiber.Map{
	// 		"success": false,
	// 		"message": "SEO already exists for this store",
	// 	})
	// }

	var existingSEO models.SEO

	if db.DB.Where("store_id = ? AND user_id = ?", storeID, userID).First(&existingSEO).RowsAffected > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"message": "SEO already exists for this store",
		})
	}

	var seoReq requests.CreateSeoRequest

	if err := c.Bind().Body(&seoReq); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	seoReq.Title = strings.TrimSpace(seoReq.Title)
	seoReq.Description = strings.TrimSpace(seoReq.Description)
	seoReq.Keywords = strings.TrimSpace(seoReq.Keywords)

	if seoReq.Title == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "SEO title is required",
		})
	}

	if seoReq.Description == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "SEO discription is required",
		})
	}

	if seoReq.Keywords == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "SEO Keywords is required",
		})
	}

	seoCreation := models.SEO{
		UserID:      userID,
		StoreID:     storeID,
		Title:       strings.TrimSpace(seoReq.Title),
		Description: strings.TrimSpace(seoReq.Description),
		Keywords:    strings.TrimSpace(seoReq.Keywords),
		OgImageUrl:  seoReq.OgImageUrl,
	}

	if err := db.DB.Create(&seoCreation).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Could not create SEO",
			"error":   err.Error(),
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"message": "SEO created successfully",
		"data":    seoCreation,
	})
}

func UpdateSEO(c fiber.Ctx) error {
	storeIDParam := c.Params("storeID")

	storeID, err := uuid.Parse(storeIDParam)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid store ID",
		})
	}

	seoIDParam := c.Params("seoID")

	seoID, err := uuid.Parse(seoIDParam)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid SEO ID",
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
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid user ID in token",
		})
	}

	var store models.Store

	if err := db.DB.Where("id = ? AND user_id = ?", storeID, userID).First(&store).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Store not found for this user",
		})
	}

	var seo models.SEO

	if err := db.DB.Where("id = ? AND store_id = ? AND user_id = ? ", seoID, storeID, userID).First(&seo).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "SEO not found for this store",
		})
	}

	var seoReq requests.UpdateSeoRequest

	if err := c.Bind().Body(&seoReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	seoReq.Title = strings.TrimSpace(seoReq.Title)
	seoReq.Description = strings.TrimSpace(seoReq.Description)
	seoReq.Keywords = strings.TrimSpace(seoReq.Keywords)

	if seoReq.Title == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "SEO title is required",
		})
	}

	if seoReq.Description == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "SEO discription is required",
		})
	}

	if seoReq.Keywords == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"success": false,
			"message": "SEO Keywords is required",
			})
		}

		seo.Title = seoReq.Title
		seo.Description = seoReq.Description
		seo.Keywords = seoReq.Keywords
		seo.OgImageUrl = seoReq.OgImageUrl

		if err := db.DB.Save(&seo).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{
				"success": false,
				"message": "Could not update SEO",
				"error":   err.Error(),
			})
		}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "SEO updated successfully",
		"data":    seo,
	})

}

func GetMyStoreSEO(c fiber.Ctx) error {
	storeIDParam := c.Params("storeID")

	storeID, err := uuid.Parse(storeIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid store ID",
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
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid user ID in token",
		})
	}

	var seo models.SEO

	if err := db.DB.Where("store_id = ? AND user_id = ?", storeID, userID).First(&seo).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Store SEO not found for this store",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Store SEO fetched successfully",
		"data":    seo,
	})

}
