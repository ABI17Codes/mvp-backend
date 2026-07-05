package controller

import (
	"backend/db"
	"backend/models"
	"backend/requests"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func CreateCategory(c fiber.Ctx) error {
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

	var store models.Store

	if err := db.DB.Where("id = ? AND user_id = ?", storeID, userID).First(&store).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Store not found for this user",
		})
	}

	var categoryReq requests.CreateCategoryRequest

	if err := c.Bind().Body(&categoryReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	categoryReq.Name = strings.TrimSpace(categoryReq.Name)

	if categoryReq.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Category name is required",
		})
	}

	category := models.Category{
		UserID:  userID,
		StoreID: storeID,
		Name:    categoryReq.Name,
	}

	var existingCategory models.Category

	if db.DB.
		Where("store_id = ? AND user_id = ? AND LOWER(name) = LOWER(?)", storeID, userID, categoryReq.Name).
		First(&existingCategory).RowsAffected > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"message": "Category already exists in this store",
		})
	}

	if err := db.DB.Create(&category).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not create category",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Category created successfully",
		"data":    category,
	})
}

func GetCategory(c fiber.Ctx) error {
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

	var category []models.Category

	if err := db.DB.Where("store_id = ? AND user_id = ?", storeID, userID).Find(&category).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Category not found for this store",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Category fetched successfully",
		"data":    category,
	})

}

func GetCategoryByID(c fiber.Ctx) error {
	storeIDParam := c.Params("storeID")

	storeID, err := uuid.Parse(storeIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid store ID",
		})
	}

	categoryIDParam := c.Params("categoryID")

	categoryID, err := uuid.Parse(categoryIDParam)
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

	var category models.Category

	if err := db.DB.Where("id = ? AND store_id = ? AND user_id = ?", categoryID, storeID, userID).First(&category).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Category not found for this store",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Category fetched successfully",
		"data":    category,
	})

}

func UpdateCategory(c fiber.Ctx) error {

	storeIDParam := c.Params("storeID")

	storeID, err := uuid.Parse(storeIDParam)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid store ID",
		})
	}

	categoryIDParam := c.Params("categoryID")

	categoryID, err := uuid.Parse(categoryIDParam)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid Category ID",
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

	var category models.Category

	if err := db.DB.Where("id = ? AND store_id = ? AND user_id = ?", categoryID, storeID, userID).First(&category).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Category not found for this store",
		})
	}

	var categoryReq requests.UpdateCategoryRequest

	if err := c.Bind().Body(&categoryReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	// if err := db.DB.Where("id = ? AND store_id = ? AND user_id = ? ", categoryID, storeID, userID).First(&category).Error; err != nil {
	// 	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
	// 		"success": false,
	// 		"message": "Category not found for this store",
	// 	})
	// }

	categoryReq.Name = strings.TrimSpace(categoryReq.Name)

	if categoryReq.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Category Name is required",
		})
	}

	var existingCategory models.Category

	if db.DB.
		Where("store_id = ? AND user_id = ? AND LOWER(name) = LOWER(?) AND id <> ?", storeID, userID, categoryReq.Name, categoryID).
		First(&existingCategory).RowsAffected > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"message": "Category already exists in this store",
		})
	}
	
	category.Name = categoryReq.Name


	if err := db.DB.Save(&category).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Could not update Category",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Category updated successfully",
		"data":    category,
	})

}

func DeleteCategory(c fiber.Ctx) error {
	storeID, err := uuid.Parse(c.Params("storeID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid Store ID",
		})
	}

	categoryID, err := uuid.Parse(c.Params("categoryID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid Category ID",
		})
	}

	userIDString, ok := c.Locals("userID").(string)
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
			"message": "Invalid User ID",
		})
	}

	var category models.Category

	if err := db.DB.
		Where("id = ? AND store_id = ? AND user_id = ?",
			categoryID, storeID, userID).
		First(&category).Error; err != nil {

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Category not found",
		})
	}

	// Check if any products use this category
	var productCount int64

	// db.DB.Model(&models.Product{}).
	// 	Where("category_id = ?", categoryID).
	// 	Count(&productCount)

	db.DB.Model(&models.Product{}).
		Where(
			"category_id = ? AND store_id = ? AND user_id = ?",
			categoryID,
			storeID,
			userID,
		).
		Count(&productCount)

	if productCount > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"message": "Cannot delete category. Products are assigned to this category",
		})
	}

	if err := db.DB.Delete(&category).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not delete category",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Category deleted successfully",
	})

}

func GetCategoriesPublic(c fiber.Ctx) error {
	storeIDParam := c.Params("storeID")
	storeID, err := uuid.Parse(storeIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid store ID",
		})
	}

	var categories []models.Category
	if err := db.DB.Where("store_id = ?", storeID).Find(&categories).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Categories not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Categories fetched successfully",
		"data":    categories,
	})
}
