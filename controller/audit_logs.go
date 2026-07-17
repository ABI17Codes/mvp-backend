package controller

import (
	"backend/db"
	"backend/models"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func AdminGetAuditLogs(c fiber.Ctx) error {
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "50")
	action := c.Query("action")
	resource := c.Query("resource")
	userIDStr := c.Query("user_id")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 50
	}

	offset := (page - 1) * limit

	query := db.DB.Model(&models.AuditLog{}).Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Unscoped()
	})

	if action != "" {
		query = query.Where("action = ?", action)
	}
	if resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if userIDStr != "" {
		query = query.Where("user_id = ?", userIDStr)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to count audit logs",
		})
	}

	var logs []models.AuditLog
	if err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch audit logs",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    logs,
		"pagination": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}
