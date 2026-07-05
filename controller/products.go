package controller

import (
	"backend/db"
	"backend/models"
	"backend/requests"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func CreateProduct(c fiber.Ctx) error {
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
			"message": "User Not Found in Token",
		})
	}

	userID, err := uuid.Parse(userIDString)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid User ID in Token",
		})
	}

	var store models.Store

	if err := db.DB.Where("id = ? AND user_id = ?", storeID, userID).First(&store).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Store not found for this user",
		})
	}

	var productReq requests.CreateProductRequest

	if err := c.Bind().Body(&productReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	productReq.Name = strings.TrimSpace(productReq.Name)
	productReq.Description = strings.TrimSpace(productReq.Description)
	productReq.CategoryID = strings.TrimSpace(productReq.CategoryID)

	if productReq.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Product name is required",
		})
	}

	if productReq.Price <= 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Product price must be greater than 0",
		})
	}

	if productReq.CategoryID == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Category ID is required",
		})
	}

	categoryID, err := uuid.Parse(productReq.CategoryID)

	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Invalid category ID",
		})
	}

	var category models.Category

	if err := db.DB.
		Where("id = ? AND store_id = ? AND user_id = ?", categoryID, storeID, userID).
		First(&category).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Category not found for this store",
		})
	}

	// var existingProduct models.Product

	// if db.DB.Where("store_id = ? AND name = ?", storeID, productReq.Name).First(&existingProduct).RowsAffected > 0 {
	// 	return c.Status(fiber.StatusConflict).JSON(fiber.Map{
	// 		"success": false,
	// 		"message": "Product already exists in this store",
	// 	})
	// }

	var existingProduct models.Product

	if db.DB.Where("store_id = ? AND user_id = ? AND LOWER(name) = LOWER(?)", storeID, userID, productReq.Name).First(&existingProduct).RowsAffected > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"message": "Product already exists in this store",
		})
	}

	product := models.Product{
		UserID:       userID,
		StoreID:      storeID,
		Name:         productReq.Name,
		Description:  productReq.Description,
		Price:        productReq.Price,
		OfferPrice:   productReq.OfferPrice,
		OutOfStock:   productReq.OutOfStock,
		IsFeatured:   productReq.IsFeatured,
		IsNewArrival: productReq.IsNewArrival,
		Images:       productReq.Images,
		CategoryID:   categoryID,
	}

	if err := db.DB.Create(&product).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not create product",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Product created successfully",
		"data":    product,
	})
}

func GetMyStoreProduct(c fiber.Ctx) error {
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

	var product []models.Product

	if err := db.DB.Where("store_id = ? AND user_id = ?", storeID, userID).Find(&product).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Store Products not found for this store",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Store Product fetched successfully",
		"data":    product,
	})

}

func GetMyStoreProductByID(c fiber.Ctx) error {
	storeIDParam := c.Params("storeID")

	storeID, err := uuid.Parse(storeIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid store ID",
		})
	}

	productIDParam := c.Params("productID")

	productID, err := uuid.Parse(productIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid Product ID",
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

	var product models.Product

	if err := db.DB.Where("id = ? AND store_id = ? AND user_id = ?", productID, storeID, userID).First(&product).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Product not found for this store",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Product fetched successfully",
		"data":    product,
	})

}

func UpdateProduct(c fiber.Ctx) error {
	storeIDParam := c.Params("storeID")

	storeID, err := uuid.Parse(storeIDParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Invalid store ID",
		})
	}

	productIDParam := c.Params("productID")

	productID, err := uuid.Parse(productIDParam)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid Product ID",
		})
	}

	userIDValue := c.Locals("userID")
	userIDString, ok := userIDValue.(string)

	if !ok || userIDString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User Not Found in Token",
		})
	}

	userID, err := uuid.Parse(userIDString)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid User ID in Token",
		})
	}

	var store models.Store

	if err := db.DB.Where("id = ? AND user_id = ?", storeID, userID).First(&store).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Store not found for this user",
		})
	}

	var product models.Product

	if err := db.DB.Where("id = ? AND store_id = ? AND user_id = ? ", productID, storeID, userID).First(&product).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Product not found for this store",
		})
	}

	var productReq requests.CreateProductRequest

	if err := c.Bind().Body(&productReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	productReq.Name = strings.TrimSpace(productReq.Name)
	productReq.Description = strings.TrimSpace(productReq.Description)
	productReq.CategoryID = strings.TrimSpace(productReq.CategoryID)

	if productReq.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Product name is required",
		})
	}

	if productReq.Price <= 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Product price must be greater than 0",
		})
	}

	if productReq.CategoryID == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Category ID is required",
		})
	}

	categoryID, err := uuid.Parse(productReq.CategoryID)

	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Invalid category ID",
		})
	}

	var category models.Category

	if err := db.DB.
		Where("id = ? AND store_id = ? AND user_id = ?", categoryID, storeID, userID).
		First(&category).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Category not found for this store",
		})
	}

	var existingProduct models.Product

	if db.DB.Where("store_id = ? AND user_id = ? AND LOWER(name) = LOWER(?) AND id <> ?", storeID, userID, productReq.Name, productID).First(&existingProduct).RowsAffected > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"message": "Product already exists in this store",
		})
	}

	product.Name = productReq.Name
	product.Description = productReq.Description
	product.Price = productReq.Price
	product.OfferPrice = productReq.OfferPrice
	product.OutOfStock = productReq.OutOfStock
	product.IsFeatured = productReq.IsFeatured
	product.IsNewArrival = productReq.IsNewArrival
	product.Images = productReq.Images
	product.CategoryID = categoryID

	if err := db.DB.Save(&product).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not update product",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Product Updated successfully",
		"data":    product,
	})
}

func DeleteProduct(c fiber.Ctx) error {
	storeID, err := uuid.Parse(c.Params("storeID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid Store ID",
		})
	}

	productID, err := uuid.Parse(c.Params("productID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid Product ID",
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

	var product models.Product

	if err := db.DB.
		Where("id = ? AND store_id = ? AND user_id = ?",
			productID, storeID, userID).
		First(&product).Error; err != nil {

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Product not found",
		})
	}

	if err := db.DB.Delete(&product).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not delete Product",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Product deleted successfully",
	})

}
