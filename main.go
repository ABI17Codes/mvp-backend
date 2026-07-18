package main

import (
	"backend/db"
	"backend/models"
	"backend/routes"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	db.ConnectDB()

	db.DB.AutoMigrate(
		&models.User{},
		&models.Store{},
		&models.Product{},
		&models.Category{},
		&models.Address{},
		&models.SEO{},
		&models.Orders{},
		&models.Plan{},
		&models.Subscription{},
		&models.SubscriptionUsage{},
		&models.Payment{},
		&models.AuditLog{},
		&models.Config{},
	)

	db.SeedAdmin()

	app := fiber.New()

	app.Use(recover.New())

	app.Use(logger.New())

	// app.Use(logger.New(logger.Config{
	// 	Format: "[${ip}] : ${port} ${status} - ${method} ${path}\n",
	// }))

	app.Use(requestid.New())
	// app.Use(logger.New(logger.Config{
	// 	Format: "${requestid} ${status} ${method} ${path}\n",
	// }))

	url := os.Getenv("FRONTEND_URL")

	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: time.Minute,

		Next: func(c fiber.Ctx) bool {
			return c.Path() == "/"
		},
	}))

	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			url,
		},
		AllowCredentials: true,
		AllowMethods: []string{
			fiber.MethodGet,
			fiber.MethodPost,
			fiber.MethodPut,
			fiber.MethodPatch,
			fiber.MethodOptions,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
		},
	}))

	apiv1 := app.Group("/api/v1")

	routes.SetupRoutes(apiv1)

	apiv1.Get("/fix-plans", func(c fiber.Ctx) error {
		var plans []models.Plan
		db.DB.Find(&plans)

		for _, p := range plans {
			name := strings.ToLower(p.Name)
			if strings.Contains(name, "basic") {
				p.ProductLimit = 50
				p.MonthlyOrderLimit = 1000
			} else if strings.Contains(name, "growth") {
				p.ProductLimit = 200
				p.MonthlyOrderLimit = 5000
			} else if strings.Contains(name, "scale") {
				p.ProductLimit = -1
				p.MonthlyOrderLimit = -1
			} else if strings.Contains(name, "free") {
				p.ProductLimit = 20
				p.MonthlyOrderLimit = 50
			}
			db.DB.Save(&p)
		}
		return c.SendString("Plans have been successfully fixed! You can now check your subscription page.")
	})

	// Serve uploaded images as static files
	app.Get("/uploads/*", func(c fiber.Ctx) error {
		filePath := "." + c.Path()
		return c.SendFile(filePath)
	})

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Server running!")
	})



	port := os.Getenv("PORT")

	if port != "" && port[0] != ':' {
		port = ":" + port
	}

	log.Println("🚀 Server running on", port)
	log.Fatal(app.Listen(port))

}








