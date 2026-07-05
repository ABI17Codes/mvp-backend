package controller

import (
	"backend/db"
	"backend/models"
	"backend/requests"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func CreateAddress(c fiber.Ctx) error {
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

	var store models.Store

	if err := db.DB.Where("id = ? AND user_id = ?", storeID, userID).First(&store).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Store not found for this user",
		})
	}

	var existingAddress models.Address

	// if db.DB.Where("store_id = ?", storeID).First(&existingAddress).RowsAffected > 0 {
	// 	return c.Status(fiber.StatusConflict).JSON(fiber.Map{
	// 		"success": false,
	// 		"message": "Address already exists for this store",
	// 	})
	// }
	if db.DB.Where("store_id = ? AND user_id = ?", storeID, userID).First(&existingAddress).RowsAffected > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"message": "Address already exists for this store",
		})
	}

	var addressReq requests.CreateAddressRequest

	if err := c.Bind().Body(&addressReq); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	addressReq.Street = strings.TrimSpace(addressReq.Street)
	addressReq.City = strings.TrimSpace(addressReq.City)
	addressReq.State = strings.TrimSpace(addressReq.State)
	addressReq.Country = strings.TrimSpace(addressReq.Country)
	addressReq.PostalCode = strings.TrimSpace(addressReq.PostalCode)

	if addressReq.Street == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Address Street is required",
		})
	}
	if addressReq.City == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Address City is required",
		})
	}
	if addressReq.State == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Address State is required",
		})
	}
	if addressReq.Country == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Address Country is required",
		})
	}
	if addressReq.PostalCode == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Address PostalCode is required",
		})
	}

	addressCreation := models.Address{
		UserID:     userID,
		StoreID:    storeID,
		Street:     strings.TrimSpace(addressReq.Street),
		City:       strings.TrimSpace(addressReq.City),
		State:      strings.TrimSpace(addressReq.State),
		Country:    strings.TrimSpace(addressReq.Country),
		PostalCode: strings.TrimSpace(addressReq.PostalCode),
	}

	if err := db.DB.Create(&addressCreation).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Could not create address",
			"error":   err.Error(),
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"message": "Address created successfully",
		"data":    addressCreation,
	})

}

func UpdateAddress(c fiber.Ctx) error {
	storeIDParam := c.Params("storeID")

	storeID, err := uuid.Parse(storeIDParam)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid store ID",
		})
	}

	addressIDParam := c.Params("addressID")

	addressID, err := uuid.Parse(addressIDParam)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid Address ID",
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

	var address models.Address

	if err := db.DB.Where("id = ? AND store_id = ? AND user_id = ? ", addressID, storeID, userID).First(&address).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Address not found for this store",
		})
	}

	var addressReq requests.UpdateAddressRequest

	if err := c.Bind().Body(&addressReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	addressReq.Street = strings.TrimSpace(addressReq.Street)
	addressReq.City = strings.TrimSpace(addressReq.City)
	addressReq.State = strings.TrimSpace(addressReq.State)
	addressReq.Country = strings.TrimSpace(addressReq.Country)
	addressReq.PostalCode = strings.TrimSpace(addressReq.PostalCode)

	if addressReq.Street == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Address Street is required",
		})
	}
	if addressReq.City == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Address City is required",
		})
	}
	if addressReq.State == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Address State is required",
		})
	}
	if addressReq.Country == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Address Country is required",
		})
	}
	if addressReq.PostalCode == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Address PostalCode is required",
		})
	}

	address.Street = addressReq.Street
	address.City = addressReq.City
	address.State = addressReq.State
	address.Country = addressReq.Country
	address.PostalCode = addressReq.PostalCode

	if err := db.DB.Save(&address).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Could not update address",
			"error":   err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "address updated successfully",
		"data":    address,
	})

}

func GetMyStoreAddress(c fiber.Ctx) error {
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

	var address models.Address

	if err := db.DB.Where("store_id = ? AND user_id = ?", storeID, userID).First(&address).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Store Address not found for this store",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Store Address fetched successfully",
		"data":    address,
	})

}
