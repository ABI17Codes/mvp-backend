package controller

import (
	"backend/db"
	"backend/models"
	"strings"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
)

// GetAllUsers returns all users in the system (paginated optionally)
func GetAllUsers(c fiber.Ctx) error {
	var users []models.User
	if err := db.DB.Find(&users).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Could not fetch users",
		})
	}

	// Remove passwords from response
	for i := range users {
		users[i].Password = ""
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    users,
	})
}

// GetUserByID fetches a specific user by ID
func GetUserByID(c fiber.Ctx) error {
	id := c.Params("id")
	var user models.User

	if err := db.DB.First(&user, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "User not found",
		})
	}

	user.Password = ""

	return c.JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

// CreateUserInput structure for admin creating users directly
type CreateUserInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// CreateUser allows an admin to create a new user manually
func CreateUser(c fiber.Ctx) error {
	input := new(CreateUserInput)

	if err := c.Bind().Body(input); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	if input.Email == "" || input.Password == "" || input.Name == "" {
		return c.Status(422).JSON(fiber.Map{
			"success": false,
			"error":   "Name, Email, and Password are required",
		})
	}

	role := models.RoleCustomer
	if input.Role != "" {
		// Validating role
		switch input.Role {
		case models.RoleStoreOwner, models.RoleStaff, models.RoleCustomer:
			role = input.Role
		default:
			return c.Status(400).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid role specified",
			})
		}
	}

	var existing models.User
	if err := db.DB.Where("email = ?", input.Email).First(&existing).Error; err == nil {
		return c.Status(409).JSON(fiber.Map{
			"success": false,
			"error":   "Email already in use",
		})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), 14)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Something went wrong",
		})
	}

	user := models.User{
		Name:     input.Name,
		Email:    input.Email,
		Password: string(hashedPassword),
		Role:     role,
	}

	if err := db.DB.Create(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Could not create user",
		})
	}

	user.Password = ""

	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"message": "User created successfully",
		"data":    user,
	})
}

// UpdateUserRoleInput structure for role updates
type UpdateUserRoleInput struct {
	Role string `json:"role"`
}

// UpdateUserRole updates the role of a user
func UpdateUserRole(c fiber.Ctx) error {
	id := c.Params("id")
	input := new(UpdateUserRoleInput)

	if err := c.Bind().Body(input); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	switch input.Role {
	case models.RoleStoreOwner, models.RoleStaff, models.RoleCustomer:
		// Valid role
	default:
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid role specified",
		})
	}

	var user models.User
	if err := db.DB.First(&user, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "User not found",
		})
	}

	if user.Role == models.RoleAdmin {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   "Cannot change the role of an admin",
		})
	}

	user.Role = input.Role

	if err := db.DB.Save(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Could not update user role",
		})
	}

	user.Password = ""

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User role updated successfully",
		"data":    user,
	})
}

// DeleteUser removes a user from the system
func DeleteUser(c fiber.Ctx) error {
	id := c.Params("id")
	var user models.User

	if err := db.DB.First(&user, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "User not found",
		})
	}

	if user.Role == models.RoleAdmin {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   "Cannot delete an admin",
		})
	}

	if err := db.DB.Delete(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Could not delete user",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User deleted successfully",
	})
}

// UpdateUserPasswordInput structure for admin password updates
type UpdateUserPasswordInput struct {
	Password string `json:"password"`
}

// UpdateUserPassword allows an admin to change another user's password
func UpdateUserPassword(c fiber.Ctx) error {
	id := c.Params("id")
	input := new(UpdateUserPasswordInput)

	if err := c.Bind().Body(input); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	if len(input.Password) < 6 {
		return c.Status(422).JSON(fiber.Map{
			"success": false,
			"error":   "Password must be at least 6 characters",
		})
	}

	var user models.User
	if err := db.DB.First(&user, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "User not found",
		})
	}

	if user.Role == models.RoleAdmin {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   "Cannot change the password of an admin user",
		})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), 14)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Something went wrong while hashing the password",
		})
	}

	user.Password = string(hashedPassword)
	if err := db.DB.Save(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Could not update user password",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User password reset successfully",
	})
}
