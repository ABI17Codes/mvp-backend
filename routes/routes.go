package routes

import (
	"backend/controller"
	"backend/utils"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

func SetupRoutes(router fiber.Router) {

	// auth routes
	auth := router.Group("/auth")

	auth.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: time.Minute,
	}))

	auth.Post("/register", controller.Register)
	auth.Post("/login", controller.Login)
	auth.Post("/logout", controller.Logout)

	auth.Get("/me", utils.JwtMiddleware, controller.Me)

	// store group - requires JWT
	storeBase := router.Group("/store", utils.JwtMiddleware)

	storeBase.Use(limiter.New(limiter.Config{
		Max:        200,
		Expiration: time.Minute,
	}))

	// Anyone with a JWT (including customers) can create a store
	storeBase.Post("/create", controller.CreateStore)

	// Sub-group requiring storeowner role
	store := storeBase.Group("", utils.StoreownerOnly)
	// Image upload endpoint for store owners
	store.Post("/upload", controller.UploadImage) // POST /store/upload

	store.Get("/me", controller.GetMyStore)
	store.Put("/update", controller.UpdateStore)
	// store.Post("/delete/:storeID", controller.CreateStore)

	// subscription
	store.Get("/subscription", controller.GetMySubscription)
	store.Post("/subscription/upgrade", controller.UpgradeSubscription)

	// orders
	store.Post("/:storeID/order/create", controller.CreateOrder)
	store.Get("/:storeID/order/get", controller.GetMyStoreOrders)
	store.Put("/:storeID/order/:orderID/status", controller.UpdateOrderStatus)

	// seo
	store.Post("/:storeID/seo/create", controller.CreateSEO)
	store.Get("/:storeID/seo/get", controller.GetMyStoreSEO)
	store.Put("/:storeID/seo/update", controller.UpdateSEO)
	// store.Delete("/:storeID/seo/delete", controller.)

	// address
	store.Post("/:storeID/address/create", controller.CreateAddress)
	store.Get("/:storeID/address/get", controller.GetMyStoreAddress)
	store.Put("/:storeID/address/update", controller.UpdateAddress)
	// store.Delete("/:storeID/address/delete", controller.)

	// product
	store.Post("/:storeID/product/create", controller.CreateProduct)
	store.Get("/:storeID/product/get", controller.GetMyStoreProduct)
	store.Get("/:storeID/product/:productID/getbyid", controller.GetMyStoreProductByID)
	store.Put("/:storeID/product/:productID/update", controller.UpdateProduct)
	store.Delete("/:storeID/product/:productID/delete", controller.DeleteProduct)

	// categories
	store.Post("/:storeID/category/create", controller.CreateCategory)
	store.Get("/:storeID/category/get", controller.GetCategory)
	store.Get("/:storeID/category/:categoryID/getbyid", controller.GetCategoryByID)
	store.Put("/:storeID/category/:categoryID/update", controller.UpdateCategory)
	store.Delete("/:storeID/category/:categoryID/delete", controller.DeleteCategory)

	// Public store routes (no JWT required)
	router.Get("/plans", controller.GetPlans)
	router.Get("/config/upi", controller.GetUPIConfig)
	publicStore := router.Group("/store-public")
	publicStore.Get("/:slug", controller.GetStoreBySlug)
	publicStore.Get("/:storeID/categories", controller.GetCategoriesPublic)

	// Public order and upload
	router.Post("/upload", controller.PublicUploadImage)
	router.Post("/public/store/:storeID/orders", controller.PublicCreateOrder)
	router.Get("/public/orders/track/:orderNumber", controller.GetPublicOrderByOrderNumber)

	// client admin page

	// admin
	admin := router.Group("/admin", utils.JwtMiddleware, utils.AdminOnly)

	admin.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: time.Minute,
	}))

	admin.Get("/stores", controller.GetAllStores)

	// Admin User Management routes
	admin.Get("/users", controller.GetAllUsers)
	admin.Get("/users/:id", controller.GetUserByID)
	admin.Post("/users", controller.CreateUser)
	admin.Put("/users/:id/role", controller.UpdateUserRole)
	admin.Put("/users/:id/password", controller.UpdateUserPassword)
	admin.Delete("/users/:id", controller.DeleteUser)
	admin.Put("/users/:id", controller.UpdateUser)

	// Admin Store Management routes
	admin.Put("/stores/:id", controller.AdminUpdateStore)
	admin.Delete("/stores/:id", controller.AdminDeleteStore)

	// Payment requests verification
	admin.Get("/payments", controller.AdminGetPayments)
	admin.Post("/payments/:id/verify", controller.AdminVerifyPayment)
	admin.Put("/payments/:id/refund", controller.AdminRefundPayment)

	// Audit logs
	admin.Get("/audit-logs", controller.AdminGetAuditLogs)

	// Platform Config
	admin.Put("/config/upi", controller.UpdateUPIConfig)

	// Admin Plans Management routes
	admin.Get("/plans", controller.AdminGetPlans)
	admin.Post("/plans", controller.AdminCreatePlan)
	admin.Put("/plans/:id", controller.AdminUpdatePlan)
	admin.Delete("/plans/:id", controller.AdminDeletePlan)

}
