package controller

import (
	"backend/db"
	"backend/models"
	"backend/requests"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func CreateOrder(c fiber.Ctx) error {
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

	var orderReq requests.CreateOrderRequest

	if err := c.Bind().Body(&orderReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	// Trim all strings
	orderReq.UserName = strings.TrimSpace(orderReq.UserName)
	orderReq.Number = strings.TrimSpace(orderReq.Number)
	orderReq.Address = strings.TrimSpace(orderReq.Address) 
	orderReq.ProductName = strings.TrimSpace(orderReq.ProductName) 
	orderReq.Note = strings.TrimSpace(orderReq.Note)

	// Validate required fields
	if orderReq.UserName == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "UserName is required",
		})
	}

	if orderReq.Number == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Phone number is required",
		})
	}


	if orderReq.ProductName == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "ProductName is required",
		})
	}
 

	if orderReq.Price <= 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Price must be greater than 0",
		})
	}

	if orderReq.Quantity <= 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Quantity must be greater than 0",
		})
	}

	totalAmount := orderReq.Price * orderReq.Quantity

	// ProductID parse
	productID, err := uuid.Parse(orderReq.ProductID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid Product ID",
		})
	}

	var count int64
	db.DB.Model(&models.Orders{}).Where("store_id = ?", storeID).Count(&count)

	// Order create
	order := models.Orders{
		UserID:      userID,
		StoreID:     storeID,
		ProductID:   productID,
		UserName:    orderReq.UserName,
		Number:      orderReq.Number,
		Address:     orderReq.Address,
		ProductName: orderReq.ProductName,
		Price:       orderReq.Price,
		Quantity:    orderReq.Quantity,
		TotalAmount: totalAmount,
		Note:        orderReq.Note,
		Status:      "pending",
		OrderNumber: fmt.Sprintf("ORD-%s-%d", strings.ToUpper(uuid.New().String()[:4]), count+1),
	}

	if err := db.DB.Create(&order).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to create order",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Order created successfully",
		"data":    order,
	})

}

func GetMyStoreOrders(c fiber.Ctx) error {
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

	var orders []models.Orders
	if err := db.DB.Where("store_id = ?", storeID).Order("created_at desc").Find(&orders).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not fetch orders",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Orders fetched successfully",
		"data":    orders,
	})
}

func UpdateOrderStatus(c fiber.Ctx) error {
	storeIDParam := c.Params("storeID")
	storeID, err := uuid.Parse(storeIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid store ID",
		})
	}

	orderIDParam := c.Params("orderID")
	orderID, err := uuid.Parse(orderIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid order ID",
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

	var order models.Orders
	if err := db.DB.Where("id = ? AND store_id = ?", orderID, storeID).First(&order).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Order not found",
		})
	}

	type UpdateStatusRequest struct {
		Status string `json:"status"`
	}

	var req UpdateStatusRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	req.Status = strings.TrimSpace(req.Status)
	if req.Status == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Status is required",
		})
	}

	order.Status = req.Status
	if err := db.DB.Save(&order).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not update order status",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Order status updated successfully",
		"data":    order,
	})
}
