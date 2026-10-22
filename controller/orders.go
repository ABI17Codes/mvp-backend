package controller

import (
	"backend/db"
	"backend/models"
	"backend/requests"
	"backend/services"
	"encoding/json"
	"fmt"
	"strings" 

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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
	orderReq.Note = strings.TrimSpace(orderReq.Note)
	orderReq.PaymentMethod = strings.TrimSpace(orderReq.PaymentMethod)
	orderReq.PaymentReference = strings.TrimSpace(orderReq.PaymentReference)
	orderReq.PaymentScreenshot = strings.TrimSpace(orderReq.PaymentScreenshot)

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

	if orderReq.TotalAmount <= 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Total amount must be greater than 0",
		})
	}

	// Serialize OrderItems
	orderItemsJson, err := json.Marshal(orderReq.OrderItems)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid cart items format",
		})
	}

	var count int64
	db.DB.Model(&models.Orders{}).Where("store_id = ?", storeID).Count(&count)

	// Order create
	order := models.Orders{
		UserID:            userID,
		StoreID:           storeID,
		UserName:          orderReq.UserName,
		Number:            orderReq.Number,
		Address:           orderReq.Address,
		OrderItems:        datatypes.JSON(orderItemsJson),
		Subtotal:          orderReq.Subtotal,
		DeliveryCharge:    orderReq.DeliveryCharge,
		TotalAmount:       orderReq.TotalAmount,
		Note:              orderReq.Note,
		Status:            "Pending Payment",
		PaymentMethod:     orderReq.PaymentMethod,
		PaymentStatus:     "Pending",
		PaymentReference:  orderReq.PaymentReference,
		PaymentScreenshot: orderReq.PaymentScreenshot,
		OrderNumber:       fmt.Sprintf("ORD-%s-%d", strings.ToUpper(uuid.New().String()[:4]), count+1),
	}

	if err := db.DB.Create(&order).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to create order",
		})
	}

	// Increment usage if tracked
	usageID := c.Locals("usage_id")
	if usageID != nil {
		db.DB.Model(&models.SubscriptionUsage{}).Where("id = ?", usageID).UpdateColumn("orders_used", gorm.Expr("orders_used + ?", 1))
	}

	if c.Locals("using_extra_order") != nil {
		db.DB.Model(&models.Store{}).Where("id = ?", store.ID).UpdateColumn("extra_order_limit", gorm.Expr("extra_order_limit - ?", 1))
	}

	services.Log(c, services.Activity{
		Action:      services.ActionCreateOrder,
		Resource:    services.ResourceOrder,
		ResourceID:  &order.ID,
		Description: "Created order " + order.OrderNumber + " for " + order.UserName + " (Total: " + fmt.Sprintf("%d", order.TotalAmount) + ")",
		Success:     true,
		Metadata:    fiber.Map{"order_number": order.OrderNumber, "total_amount": order.TotalAmount},
	})

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
		Status        string `json:"status"`
		PaymentStatus string `json:"paymentStatus"`
	}

	var req UpdateStatusRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	req.Status = strings.TrimSpace(req.Status)
	req.PaymentStatus = strings.TrimSpace(req.PaymentStatus)

	if req.Status == "" && req.PaymentStatus == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Status or PaymentStatus is required",
		})
	}

	if req.Status != "" {
		order.Status = req.Status
	}
	if req.PaymentStatus != "" {
		order.PaymentStatus = req.PaymentStatus
	}
	
	if err := db.DB.Save(&order).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not update order",
			"error":   err.Error(),
		})
	}

	services.Log(c, services.Activity{
		Action:      services.ActionUpdateOrder,
		Resource:    services.ResourceOrder,
		ResourceID:  &order.ID,
		Description: "Updated order " + order.OrderNumber + " status to " + order.Status + ", payment to " + order.PaymentStatus,
		Success:     true,
		Metadata:    fiber.Map{"order_number": order.OrderNumber, "status": order.Status, "payment_status": order.PaymentStatus},
	})

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Order status updated successfully",
		"data":    order,
	})
}

func PublicCreateOrder(c fiber.Ctx) error {
	storeIDParam := c.Params("storeID")

	storeID, err := uuid.Parse(storeIDParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Invalid store ID",
		})
	}

	var store models.Store

	if err := db.DB.Where("id = ?", storeID).First(&store).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Store not found",
		})
	}

	var orderReq requests.CreateOrderRequest

	if err := c.Bind().Body(&orderReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	orderReq.UserName = strings.TrimSpace(orderReq.UserName)
	orderReq.Number = strings.TrimSpace(orderReq.Number)
	orderReq.Address = strings.TrimSpace(orderReq.Address) 
	orderReq.Note = strings.TrimSpace(orderReq.Note)
	orderReq.PaymentMethod = strings.TrimSpace(orderReq.PaymentMethod)
	orderReq.PaymentReference = strings.TrimSpace(orderReq.PaymentReference)
	orderReq.PaymentScreenshot = strings.TrimSpace(orderReq.PaymentScreenshot)

	if orderReq.UserName == "" || orderReq.Number == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Name and Phone number are required",
		})
	}

	if orderReq.TotalAmount <= 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success": false,
			"message": "Total amount must be greater than 0",
		})
	}

	if orderReq.PaymentMethod == "COD" && !store.CodEnabled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Cash on Delivery (COD) is disabled for this store",
		})
	}

	if orderReq.PaymentMethod == "UPI" {
		if orderReq.PaymentReference == "" {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"success": false,
				"message": "UPI Transaction ID / Reference number is required",
			})
		}
		if orderReq.PaymentScreenshot == "" {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"success": false,
				"message": "Payment screenshot is required for UPI orders",
			})
		}
	}

	orderItemsJson, err := json.Marshal(orderReq.OrderItems)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid cart items format",
		})
	}

	var count int64
	db.DB.Model(&models.Orders{}).Where("store_id = ?", storeID).Count(&count)

	order := models.Orders{
		StoreID:           storeID,
		UserName:          orderReq.UserName,
		Number:            orderReq.Number,
		Address:           orderReq.Address,
		OrderItems:        datatypes.JSON(orderItemsJson),
		Subtotal:          orderReq.Subtotal,
		DeliveryCharge:    orderReq.DeliveryCharge,
		TotalAmount:       orderReq.TotalAmount,
		Note:              orderReq.Note,
		Status:            "Pending Payment",
		PaymentMethod:     orderReq.PaymentMethod,
		PaymentStatus:     "Pending",
		PaymentReference:  orderReq.PaymentReference,
		PaymentScreenshot: orderReq.PaymentScreenshot,
		OrderNumber:       fmt.Sprintf("ORD-%s-%d", strings.ToUpper(uuid.New().String()[:4]), count+1),
	}

	if err := db.DB.Create(&order).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to create order",
		})
	}

	usageID := c.Locals("usage_id")
	if usageID != nil {
		db.DB.Model(&models.SubscriptionUsage{}).Where("id = ?", usageID).UpdateColumn("orders_used", gorm.Expr("orders_used + ?", 1))
	}

	if c.Locals("using_extra_order") != nil {
		db.DB.Model(&models.Store{}).Where("id = ?", store.ID).UpdateColumn("extra_order_limit", gorm.Expr("extra_order_limit - ?", 1))
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Order created successfully",
		"data":    order,
	})
}

func isStoreOnFreePlan(storeID uuid.UUID) bool {
	// Free access to all functions
	return false
}

func GetPublicOrderByOrderNumber(c fiber.Ctx) error {
	orderNumber := strings.TrimSpace(c.Params("orderNumber"))
	if orderNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Order number is required",
		})
	}

	var order models.Orders
	if err := db.DB.Where("order_number = ?", orderNumber).First(&order).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Order not found",
		})
	}

	var store models.Store
	if err := db.DB.Where("id = ?", order.StoreID).First(&store).Error; err == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"order": order,
				"store": fiber.Map{
					"name":        store.Name,
					"number":      store.Number,
					"logo":        store.Logo,
					"upiId":       store.UpiId,
					"upiName":     store.UpiName,
					"description": store.Description,
				},
			},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"order": order,
		},
	})
}
