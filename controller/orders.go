package controller

import (
	"backend/db"
	"backend/models"
	"backend/requests"
	"backend/services"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
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

	if isStoreOnFreePlan(store.ID) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "Orders are not available on the Free plan. Please upgrade to a premium plan.",
		})
	}

	// 2. Check monthly order limit
	var activeSub models.Subscription
	if errSub := db.DB.Preload("Plan").Where("store_id = ? AND status = ? AND expiry_date > ?", store.ID, "active", time.Now()).Order("created_at desc").First(&activeSub).Error; errSub == nil {
		if activeSub.Plan.MonthlyOrderLimit > 0 {
			now := time.Now()
			
			// Determine current billing cycle
			var periodStart time.Time = activeSub.StartDate
			var periodEnd time.Time
			for {
				periodEnd = periodStart.AddDate(0, 1, 0)
				if periodEnd.After(activeSub.ExpiryDate) {
					periodEnd = activeSub.ExpiryDate
				}
				if now.Before(periodEnd) || now.Equal(periodEnd) {
					break
				}
				// Break if we exceed current time somehow without catching it (failsafe)
				if periodStart.After(now) {
					break
				}
				periodStart = periodEnd
			}

			var usage models.SubscriptionUsage
			if err := db.DB.Where("subscription_id = ? AND billing_period_start = ?", activeSub.ID, periodStart).First(&usage).Error; err != nil {
				usage = models.SubscriptionUsage{
					SubscriptionID:     activeSub.ID,
					StoreID:            store.ID,
					BillingPeriodStart: periodStart,
					BillingPeriodEnd:   periodEnd,
					OrdersUsed:         0,
				}
				db.DB.Create(&usage)
			}
			
			extraLimit := 0
			if store.ExtraOrdersExpiry != nil && time.Now().Before(*store.ExtraOrdersExpiry) {
				extraLimit = store.ExtraOrderLimit
			}
			
			limit := activeSub.Plan.MonthlyOrderLimit + extraLimit
			if usage.OrdersUsed >= limit {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"success": false,
					"message": "Monthly order limit reached. Please upgrade your subscription or contact support.",
				})
			}

			// Store usage ID in context or variable for increment later
			c.Locals("usage_id", usage.ID)
		}
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

	// Increment usage if tracked
	usageID := c.Locals("usage_id")
	if usageID != nil {
		db.DB.Model(&models.SubscriptionUsage{}).Where("id = ?", usageID).UpdateColumn("orders_used", gorm.Expr("orders_used + ?", 1))
	}

	services.Log(c, services.Activity{
		Action:      services.ActionCreateOrder,
		Resource:    services.ResourceOrder,
		ResourceID:  &order.ID,
		Description: "Created order " + order.OrderNumber + " for product: " + order.ProductName + " (Quantity: " + fmt.Sprintf("%d", order.Quantity) + ")",
		Success:     true,
		Metadata:    fiber.Map{"order_number": order.OrderNumber, "product_name": order.ProductName, "quantity": order.Quantity, "total_amount": order.TotalAmount},
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

	if isStoreOnFreePlan(store.ID) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "Orders are not available on the Free plan. Please upgrade to a premium plan.",
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

	if isStoreOnFreePlan(store.ID) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "Orders are not available on the Free plan. Please upgrade to a premium plan.",
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

	services.Log(c, services.Activity{
		Action:      services.ActionUpdateOrder,
		Resource:    services.ResourceOrder,
		ResourceID:  &order.ID,
		Description: "Updated order status of " + order.OrderNumber + " to " + order.Status,
		Success:     true,
		Metadata:    fiber.Map{"order_number": order.OrderNumber, "status": order.Status},
	})

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Order status updated successfully",
		"data":    order,
	})
}

func isStoreOnFreePlan(storeID uuid.UUID) bool {
	var sub models.Subscription
	err := db.DB.Preload("Plan").Where("store_id = ? AND status = ? AND expiry_date > ?", storeID, "active", time.Now()).Order("created_at desc").First(&sub).Error
	if err != nil {
		return true // No active subscription found = Free plan
	}
	return sub.Plan.Name == "free"
}
