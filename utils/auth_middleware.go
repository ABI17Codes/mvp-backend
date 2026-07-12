package utils

import (
	"strings"
	"log"

	"backend/db"
	"backend/models"
	"github.com/gofiber/fiber/v3"
)

func JwtMiddleware(c fiber.Ctx) error {
	tokenStr := c.Cookies("access_token")

	if tokenStr == "" {
		authHeader := c.Get("Authorization")

		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if tokenStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Authentication required",
		})
	}
	log.Println("COOKIE =", tokenStr)

	claims, err := ValidateJWT(tokenStr)
	if err != nil {
		log.Println("JWT ERROR =", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid or expired token",
		})
	}
	log.Println("CLAIMS =", claims)
	userID, ok := claims["userID"].(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid token claims",
		})
	}

	role, _ := claims["role"].(string)

	c.Locals("userID", userID)
	c.Locals("role", role)

	// Check if user is suspended
	var user models.User
	if err := db.DB.First(&user, "id = ?", userID).Error; err == nil {
		if user.IsSuspended {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "Your account has been suspended by the platform administrator.",
			})
		}
	}

	return c.Next()
}

func AdminOnly(c fiber.Ctx) error {
	roleValue := c.Locals("role")

	role, ok := roleValue.(string)
	if !ok || role != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "Admin access only",
		})
	}

	return c.Next()
}

func StoreownerOnly(c fiber.Ctx) error {
	roleValue := c.Locals("role")

	role, ok := roleValue.(string)
	if !ok || role != models.RoleStoreOwner {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "Storeowner access only",
		})
	}

	return c.Next()
}

// func CustomerOnly(c fiber.Ctx) error {
// 	roleValue := c.Locals("role")

// 	role, ok := roleValue.(string)
// 	if !ok || role != models.RoleCustomer {
// 		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
// 			"success": false,
// 			"message": "Customer access only",
// 		})
// 	}

// 	return c.Next()
// }
