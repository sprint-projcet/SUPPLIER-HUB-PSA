package routes

import (
	"supplierhub-backend/controllers"
	"supplierhub-backend/middlewares"

	"github.com/gin-gonic/gin"
)

// SetupRoutes mengatur pemetaan semua rute API dalam aplikasi Gin
func SetupRoutes(router *gin.Engine) {
	api := router.Group("/api")

	// Auth Routes (Public)
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/register", controllers.Register)
		authGroup.POST("/login", controllers.Login)
		authGroup.GET("/google", controllers.GoogleLogin)
		authGroup.GET("/google/callback", controllers.GoogleCallback)
	}

	// Public Catalog
	api.GET("/catalog", controllers.GetPublicCatalog)
	api.GET("/catalog/products/:id/reviews", controllers.GetProductReviews)
	api.GET("/public/suppliers", controllers.GetPublicSuppliers)
	api.POST("/webhook/payment", controllers.HandleSmartBankCallback)

	// Mock services for local integration testing.
	router.POST("/smartbank/payment/request", controllers.MockSmartBankPaymentRequest)
	router.POST("/smartbank/payment/simulate-callback", controllers.MockSmartBankSimulateCallback)
	router.POST("/logistikita/shipment/create", controllers.MockLogistiKitaShipmentCreate)

	router.POST(
		"/supplierhub/pembayaran",
		middlewares.RequireAuth(),
		middlewares.RequireRole("user"),
		controllers.CreatePaymentRequest,
	)

	supplierHubGroup := router.Group("/supplierhub")
	supplierHubGroup.Use(middlewares.RequestLogger())
	{
		supplierHubGroup.GET(
			"/manajemen_bahan_baku",
			middlewares.RequireAuth(),
			middlewares.RequireRole("supplier", "admin"),
			controllers.GetSupplierHubMaterials,
		)
		supplierHubGroup.POST(
			"/manajemen_bahan_baku",
			middlewares.RequireAuth(),
			middlewares.RequireRole("supplier"),
			controllers.CreateSupplierHubMaterial,
		)
		supplierHubGroup.PUT(
			"/manajemen_bahan_baku/:id",
			middlewares.RequireAuth(),
			middlewares.RequireRole("supplier", "admin"),
			controllers.UpdateSupplierHubMaterial,
		)
		supplierHubGroup.DELETE(
			"/manajemen_bahan_baku/:id",
			middlewares.RequireAuth(),
			middlewares.RequireRole("supplier", "admin"),
			controllers.DeleteSupplierHubMaterial,
		)
		supplierHubGroup.POST(
			"/order_bahan",
			middlewares.RequireAuth(),
			middlewares.RequireRole("user"),
			controllers.CreateSupplierHubOrder,
		)
		supplierHubGroup.PUT(
			"/konfirmasi_stok/:order_id",
			middlewares.RequireAuth(),
			middlewares.RequireRole("supplier"),
			controllers.ConfirmSupplierHubStock,
		)
		supplierHubGroup.GET(
			"/biaya_layanan_supplier",
			middlewares.RequireAuth(),
			middlewares.RequireRole("supplier", "admin"),
			controllers.GetSupplierHubServiceFeeSummary,
		)
		supplierHubGroup.POST("/payment/callback", controllers.HandleSupplierHubPaymentCallback)
	}

	// Semua rute di bawah ini wajib melampirkan JWT token
	api.Use(middlewares.RequireAuth())

	// Supplier Details (Authenticated)
	api.GET("/suppliers/:id", controllers.GetSupplierDetail)

	chatGroup := api.Group("/chat")
	chatGroup.Use(middlewares.RequireRole("user", "supplier"))
	{
		chatGroup.GET("/conversations", controllers.GetChatConversations)
		chatGroup.POST("/conversations", controllers.CreateChatConversation)
		chatGroup.GET("/conversations/:id/messages", controllers.GetChatMessages)
		chatGroup.POST("/conversations/:id/messages", controllers.SendChatMessage)
		chatGroup.PUT("/conversations/:id/read", controllers.MarkChatConversationRead)
	}

	// UMKM (User) Routes
	userGroup := api.Group("/user")
	userGroup.Use(middlewares.RequireRole("user"))
	{
		userGroup.GET("/profile", controllers.GetUserProfile)
		userGroup.PUT("/profile", controllers.UpdateUserProfile)
		userGroup.GET("/stats", controllers.GetUserStats)
		userGroup.GET("/orders", controllers.GetUserOrders)
		// Produk katalog (umkm viewing products)
		userGroup.GET("/products", controllers.GetProducts)
		userGroup.POST("/orders", controllers.CreateOrder)
		userGroup.PUT("/orders/:id/cancel", controllers.CancelOrder)
		userGroup.PUT("/orders/:id/complete", controllers.CompleteOrder)
		userGroup.POST("/reviews", controllers.CreateReview)
	}

	wishlistGroup := api.Group("/wishlist")
	wishlistGroup.Use(middlewares.RequireRole("user"))
	{
		wishlistGroup.GET("", controllers.GetWishlistItems)
		wishlistGroup.POST("", controllers.AddWishlistItem)
		wishlistGroup.DELETE("/:id", controllers.DeleteWishlistItem)
	}

	// Supplier Routes
	supplierGroup := api.Group("/supplier")
	supplierGroup.Use(middlewares.RequireRole("supplier"))
	{
		supplierGroup.GET("/profile", controllers.GetSupplierProfile)
		supplierGroup.PUT("/profile", controllers.UpdateSupplierProfile)
		supplierGroup.GET("/stats", controllers.GetSupplierStats)
		supplierGroup.GET("/products", controllers.GetSupplierProducts)
		supplierGroup.POST("/products", controllers.CreateProduct)
		supplierGroup.PUT("/products/:id", controllers.UpdateProduct)
		supplierGroup.DELETE("/products/:id", controllers.DeleteProduct)
		supplierGroup.GET("/notifications", controllers.GetSupplierNotifications)
		supplierGroup.PUT("/notifications/:id/read", controllers.MarkSupplierNotificationRead)
		supplierGroup.GET("/orders", controllers.GetSupplierOrders)
		supplierGroup.POST("/orders/update-status", controllers.UpdateOrderStatus)
		supplierGroup.PUT("/orders/update-status", controllers.UpdateOrderStatus)
		supplierGroup.PUT("/orders/:id", controllers.UpdateOrderStatus)
	}

	api.POST(
		"/products",
		middlewares.RequireRole("supplier"),
		controllers.CreateProduct,
	)

	// Admin Routes
	adminGroup := api.Group("/admin")
	adminGroup.Use(middlewares.RequireRole("admin"))
	{
		adminGroup.GET("/stats", controllers.GetAdminStats)
		adminGroup.GET("/profile", controllers.GetAdminProfile)
		adminGroup.PUT("/profile", controllers.UpdateAdminProfile)
		adminGroup.GET("/suppliers", controllers.GetAdminSuppliers)
		adminGroup.PUT("/suppliers/:id/verify", controllers.VerifySupplier)
		adminGroup.PUT("/suppliers/:id/status", controllers.UpdateSupplierStatus)
		adminGroup.GET("/logs", controllers.GetAdminLogs)
		adminGroup.POST("/logs", controllers.CreateAdminLog)
		adminGroup.GET("/finance", controllers.GetAdminFinanceSummary)
		adminGroup.GET("/stocks", controllers.GetAdminStockSummary)
		adminGroup.POST("/products/:id/stock-alert", controllers.SendLowStockAlert)
		adminGroup.GET("/notifications", controllers.GetAdminNotifications)
		adminGroup.PUT("/notifications/:id/read", controllers.MarkAdminNotificationRead)
	}
}
