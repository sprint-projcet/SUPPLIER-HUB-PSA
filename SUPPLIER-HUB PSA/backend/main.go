package main

import (
	"log"

	"supplierhub-backend/config"
	"supplierhub-backend/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Inisialisasi Koneksi Database
	config.ConnectDatabase()
	config.SetupGoogleOAuth()

	// 2. Setup Gin Router
	r := gin.Default()

	// 3. Konfigurasi Middleware CORS
	// (Mengizinkan UI frontend dari port atau origin berbeda untuk memanggil API ini)
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true // Boleh diubah ke Origin spesifik UI untuk Production
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Accept", "X-Requested-With", "X-Internal-Token", "X-Callback-Token"}
	r.Use(cors.New(corsConfig))

	// Serve folder uploads sebagai static files
	r.Static("/uploads", "./uploads")

	// 4. Setup Routes
	routes.SetupRoutes(r)

	// 5. Jalankan Web Server
	port := "8080"
	log.Printf("Server berjalan di port http://localhost:%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}
