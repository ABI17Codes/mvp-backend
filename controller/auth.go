package controller

import (
	"backend/db"
	"backend/models"
	"backend/services"
	"backend/utils"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// validateRegisterInput

func validateRegisterInput(input *RegisterInput) []fiber.Map {
	var errs []fiber.Map

	if strings.TrimSpace(input.Name) == "" {
		errs = append(errs, fiber.Map{"field": "name", "message": "Name is required"})
	}
	if strings.TrimSpace(input.Email) == "" {
		errs = append(errs, fiber.Map{"field": "email", "message": "Email is required"})
	} else if !strings.Contains(input.Email, "@") {
		errs = append(errs, fiber.Map{"field": "email", "message": "Invalid email format"})
	}
	if strings.TrimSpace(input.Password) == "" {
		errs = append(errs, fiber.Map{"field": "password", "message": "Password is required"})
	} else if len(input.Password) < 6 {
		errs = append(errs, fiber.Map{"field": "password", "message": "Password must be at least 6 characters"})
	}

	return errs
}

var dummyHash []byte

func init() {
	dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy"), 14)
}

// ───────────────────── Register ──────────────────────────────────────────────────────

func Register(c fiber.Ctx) error {

	input := new(RegisterInput)

	if err := c.Bind().Body(input); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	if len(input.Password) >= 20 {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "You cannot create this account",
		})
	}

	input.Name = strings.TrimSpace(input.Name)
	// input.Email = strings.TrimSpace(input.Email)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	if errs := validateRegisterInput(input); len(errs) > 0 {
		return c.Status(422).JSON(fiber.Map{
			"success": false,
			"errors":  errs,
		})
	}

	var existing models.User

	// if db.DB.Where("email = ?", input.Email).First(&existing).RowsAffected > 0 {
	// 	return c.Status(409).JSON(fiber.Map{
	// 		"success": false,
	// 		"error":   "Email already in use",
	// 	})F
	// }

	if err := db.DB.Where("email = ?", input.Email).First(&existing).Error; err == nil {
		return c.Status(409).JSON(fiber.Map{
			"success": false,
			"error":   "Email already in use",
		})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), 14)
	if err != nil {
		log.Printf("[Register] bcrypt error: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Something went wrong",
		})
	}

	user := models.User{
		Name:     input.Name,
		Email:    input.Email,
		Password: string(hashedPassword),
		Role:     models.RoleCustomer, // ✅ always hardcoded, never from client
	}

	if err := db.DB.Create(&user).Error; err != nil {
		log.Printf("[Register] DB error: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Could not create account",
		})
	}

 

	// 6. Check if account is active
	// if !user.IsActive {
	// 	return c.Status(403).JSON(fiber.Map{
	// 		"success": false,
	// 		"error":   "Account has been deactivated",
	// 	})
	// }

	// 7. Generate JWT
	token, err := utils.GenerateJWT(user.ID.String(), user.Role)
	if err != nil {
		log.Printf("[Login] JWT error: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Could not generate token",
		})
	}

	user.Password = ""

	secure := os.Getenv("APP_ENV") == "production"
	sameSite := "Lax"
	if secure {
		sameSite = "None"
	}

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   30 * 24 * 60 * 60,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		Path:     "/",
	})

	c.Locals("userID", user.ID.String())
	services.Log(c, services.Activity{
		Action:      services.ActionRegister,
		Resource:    services.ResourceUser,
		ResourceID:  &user.ID,
		Description: "User registered account: " + user.Email,
		Success:     true,
	})

	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"message": "Account created successfully",
		"data":    user,
		"token":   token,
	})
}

// ── Login ─────────────────────────────────────────────────────────
func Login(c fiber.Ctx) error {

	// 1. Parse body into DTO
	input := new(LoginInput)
	if err := c.Bind().Body(input); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	if input.Email == "" || input.Password == "" {
		return c.Status(422).JSON(fiber.Map{
			"success": false,
			"error":   "Email and password are required",
		})
	}

	var user models.User
	if err := db.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(input.Password))
		return c.Status(401).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid email or password", // ✅ don't reveal if email exists
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return c.Status(401).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid email or password",
		})
	}

	if user.IsSuspended {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "Your account has been suspended by the platform administrator.",
		})
	}

	// 6. Check if account is active
	// if !user.IsActive {
	// 	return c.Status(403).JSON(fiber.Map{
	// 		"success": false,
	// 		"error":   "Account has been deactivated",
	// 	})
	// }

	// 7. Generate JWT
	token, err := utils.GenerateJWT(user.ID.String(), user.Role)
	if err != nil {
		log.Printf("[Login] JWT error: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Could not generate token",
		})
	}

	user.Password = ""

	secure := os.Getenv("APP_ENV") == "production"
	sameSite := "Lax"
	if secure {
		sameSite = "None"
	}

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   30 * 24 * 60 * 60,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		Path:     "/",
	})

	c.Locals("userID", user.ID.String())
	services.Log(c, services.Activity{
		Action:      services.ActionLogin,
		Resource:    services.ResourceUser,
		ResourceID:  &user.ID,
		Description: "User logged in: " + user.Email,
		Success:     true,
	})

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Login successful",
		"data":    user,
		"token":   token,
	})
}

func Me(c fiber.Ctx) error {
	userID := c.Locals("userID")

	var user models.User

	if err := db.DB.First(&user, "id = ?", userID).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
		})
	}

	user.Password = ""

	return c.JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

func Logout(c fiber.Ctx) error {
	secure := os.Getenv("APP_ENV") == "production"
	sameSite := "Lax"
	if secure {
		sameSite = "None"
	}

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   -1,
		Path:     "/",
	})

	userIDStr, ok := c.Locals("userID").(string)
	if ok && userIDStr != "" {
		if uID, err := uuid.Parse(userIDStr); err == nil {
			services.Log(c, services.Activity{
				Action:      services.ActionLogout,
				Resource:    services.ResourceUser,
				ResourceID:  &uID,
				Description: "User logged out",
				Success:     true,
			})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Logout successful",
	})
}
