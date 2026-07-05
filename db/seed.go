package db

import (
	"backend/models"
	"golang.org/x/crypto/bcrypt"
	"log"
	"os"
)

func SeedAdmin() {
	var existing models.User

	result := DB.Where("role = ? ", models.RoleAdmin).First(&existing)
	if result.RowsAffected > 0 {
		log.Println("⚡ Admin already exists — skipping seed")
		return
	}
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		log.Println("⚠️  ADMIN_PASSWORD not set")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(adminPassword), 14)
	if err != nil {
		log.Fatal("❌ Could not hash admin password:", err)
	}
	admin := models.User{
		Name:     "Super Admin",
		Email:    os.Getenv("ADMIN_EMAIL"),
		Password: string(hashed),
		Role:     models.RoleAdmin,
		// IsActive: true,
	}
	// Fallback email for dev
	if admin.Email == "" {
		log.Println("⚠️  ADMIN_EMAIL not set")
	}

	if err := DB.Create(&admin).Error; err != nil {
		log.Fatal("❌ Could not seed admin:", err)
	}
	
	log.Println("✅ Admin seeded successfully!")
	log.Println("   Email   :", admin.Email)
	log.Println("   Password:", admin.Password)
	log.Println("   Role    :", admin.Role)
	log.Println("   ID      :", admin.ID)
}
