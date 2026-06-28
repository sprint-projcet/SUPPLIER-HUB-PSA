package config

import (
	"database/sql"
	"log"
	"os"
	"time"

	"supplierhub-backend/models"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

// ConnectDatabase menginisialisasi hubungan koneksi aplikasi dengan MySQL
func ConnectDatabase() {
	// Memuat konfigurasi environment variables (opsional jika sudah ada OS ENV)
	err := godotenv.Load()
	if err != nil {
		err = godotenv.Load("backend/.env")
	}
	if err != nil {
		log.Println("Peringatan: Tidak dapat memuat file .env atau backend/.env (mungkin menggunakan environment default)")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Default config untuk MySQL (contoh development)
		dsn = "root:@tcp(127.0.0.1:3306)/supplierhub?charset=utf8mb4&parseTime=True&loc=Local"
	}

	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatalf("Gagal terhubung ke database MySQL: %v", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		log.Fatalf("Gagal menyiapkan koneksi database MySQL: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := pingDatabase(sqlDB); err != nil {
		log.Fatalf("Gagal memverifikasi koneksi database MySQL: %v. Pastikan MySQL/XAMPP berjalan di 127.0.0.1:3306 dan database supplierhub sudah ada.", err)
	}

	log.Println("Database MySQL Terhubung!")

	// Menjalankan Auto Migration (Menyesuaikan skema tabel ke Data Models secara otomatis)
	// Disabling FK constraint creation here avoids migration failure on legacy
	// data yang memiliki urutan INSERT / orphaned rows sebelumnya.
	err = database.AutoMigrate(
		&models.User{},
		&models.Product{},
		&models.Order{},
		&models.Payment{},
		&models.FinanceLog{},
		&models.RequestLog{},
		&models.ShipmentLog{},
		&models.Notification{},
		&models.ChatConversation{},
		&models.ChatMessage{},
		&models.Wishlist{},
		&models.Log{},
		&models.Review{},
	)
	if err != nil {
		log.Fatalf("Gagal menjalankan migrasi schema database: %v", err)
	}

	DB = database

	// SEEDER: Membuat akun Admin default jika belum ada
	seedAdmin()
	seedSuppliers()
}

func pingDatabase(sqlDB *sql.DB) error {
	var err error
	for attempt := 1; attempt <= 5; attempt++ {
		if err = sqlDB.Ping(); err == nil {
			return nil
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return err
}

func seedAdmin() {
	var admin models.User
	// Cek apakah admin sudah ada
	if err := DB.Where("email = ?", "admin@supplierhub.com").First(&admin).Error; err != nil {
		// Jika tidak ada, buat admin baru
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)

		newAdmin := models.User{
			BusinessName: "System Administrator",
			Email:        "admin@supplierhub.com",
			PasswordHash: string(hashedPassword),
			Role:         models.RoleAdmin,
			Status:       "active",
		}

		if err := DB.Create(&newAdmin).Error; err == nil {
			log.Println("✅ Akun Admin default berhasil dibuat (admin@supplierhub.com / admin123)")
		} else {
			log.Printf("⚠️ Gagal membuat akun Admin default: %v\n", err)
		}
	}
}

func seedSuppliers() {
	var count int64
	DB.Model(&models.User{}).Where("role = ?", models.RoleSupplier).Count(&count)
	if count > 0 {
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("supplier123"), bcrypt.DefaultCost)

	suppliers := []models.User{
		{
			ID:           "supplier-uuid-1",
			BusinessName: "PT. Tekstil Maju Bersama",
			Email:        "supplier.tekstil@supplierhub.com",
			PasswordHash: string(hashedPassword),
			Role:         models.RoleSupplier,
			Address:      "Jl. Raya Soreang No. 123, Bandung, Jawa Barat",
			Category:     "Tekstil & Pakaian",
			Region:       "Bandung",
			PICName:      "Budi Santoso",
			Phone:        "081234567890",
			Status:       "active",
		},
		{
			ID:           "supplier-uuid-2",
			BusinessName: "Elektronik Jaya Abadi",
			Email:        "supplier.elektronik@supplierhub.com",
			PasswordHash: string(hashedPassword),
			Role:         models.RoleSupplier,
			Address:      "Ruko Mangga Dua Blok A No. 15, Jakarta Pusat",
			Category:     "Elektronik & Gadget",
			Region:       "Jakarta",
			PICName:      "Hendra Wijaya",
			Phone:        "081876543210",
			Status:       "active",
		},
		{
			ID:           "supplier-uuid-3",
			BusinessName: "CV. Packindo Creative",
			Email:        "supplier.packindo@supplierhub.com",
			PasswordHash: string(hashedPassword),
			Role:         models.RoleSupplier,
			Address:      "Kawasan Industri Rungkut Blok F No. 8, Surabaya, Jawa Timur",
			Category:     "Kemasan & Packaging",
			Region:       "Surabaya",
			PICName:      "Siti Aminah",
			Phone:        "081399887766",
			Status:       "active",
		},
	}

	for _, supplier := range suppliers {
		if err := DB.Create(&supplier).Error; err == nil {
			log.Printf("✅ Akun Supplier default berhasil dibuat: %s (%s / supplier123)", supplier.BusinessName, supplier.Email)
			seedProductsForSupplier(supplier)
		}
	}
}

func seedProductsForSupplier(supplier models.User) {
	var products []models.Product
	if supplier.ID == "supplier-uuid-1" {
		products = []models.Product{
			{
				ID:          "prod-tekstil-1",
				SupplierID:  supplier.ID,
				Name:        "Kain Katun Premium 100%",
				Category:    "Tekstil",
				Price:       45000,
				Stock:       1500,
				Description: "Kain katun premium sangat lembut dan cocok untuk pakaian.",
				Location:    supplier.Region,
			},
			{
				ID:          "prod-tekstil-2",
				SupplierID:  supplier.ID,
				Name:        "Kain Linen Serat Alami",
				Category:    "Tekstil",
				Price:       55000,
				Stock:       800,
				Description: "Kain linen berkualitas tinggi terbuat dari serat alami.",
				Location:    supplier.Region,
			},
		}
	} else if supplier.ID == "supplier-uuid-2" {
		products = []models.Product{
			{
				ID:          "prod-elek-1",
				SupplierID:  supplier.ID,
				Name:        "Kabel USB Type-C Fast Charge",
				Category:    "Elektronik",
				Price:       15000,
				Stock:       2000,
				Description: "Kabel USB Type-C mendukung pengisian daya cepat.",
				Location:    supplier.Region,
			},
		}
	} else if supplier.ID == "supplier-uuid-3" {
		products = []models.Product{
			{
				ID:          "prod-pack-1",
				SupplierID:  supplier.ID,
				Name:        "Kardus Box Polos 20x20x10",
				Category:    "Kemasan",
				Price:       2500,
				Stock:       5000,
				Description: "Kardus box tebal multifungsi untuk pengemasan barang.",
				Location:    supplier.Region,
			},
		}
	}

	for _, product := range products {
		if err := DB.Create(&product).Error; err == nil {
			review := models.Review{
				ID:        "rev-" + product.ID,
				OrderID:   "mock-order-id-" + product.ID,
				ProductID: product.ID,
				UmkmID:    "mock-umkm-id",
				Rating:    5,
				Comment:   "Bahan sangat bagus dan pengiriman cepat!",
			}
			DB.Create(&review)
		}
	}
}
