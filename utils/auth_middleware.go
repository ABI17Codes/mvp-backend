package utils

import (
	"strings"

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

	claims, err := ValidateJWT(tokenStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid or expired token",
		})
	}
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
