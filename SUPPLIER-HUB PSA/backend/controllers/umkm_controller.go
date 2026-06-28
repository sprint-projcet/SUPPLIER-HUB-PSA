package controllers

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"supplierhub-backend/config"
	"supplierhub-backend/models"
	"supplierhub-backend/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type userProfileInput struct {
	BusinessName string `json:"business_name" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Address      string `json:"address" binding:"required"`
	Category     string `json:"category" binding:"required"`
	PICName      string `json:"pic_name" binding:"required"`
	Phone        string `json:"phone" binding:"required"`
}

func getAuthenticatedUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return "", false
	}

	userIDString, ok := userID.(string)
	if !ok || userIDString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi pengguna tidak valid"})
		return "", false
	}

	return userIDString, true
}

func GetUserStats(c *gin.Context) {
	umkmID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	var totalOrders int64
	var pendingOrders int64
	var shippedOrders int64
	var completedOrders int64
	var totalSpending float64
	pendingStatuses := []models.OrderStatus{
		models.OrderPending,
		models.OrderPendingSupplierConfirmation,
		models.OrderSupplierConfirmed,
		models.OrderPaymentPending,
	}
	shippedStatuses := []models.OrderStatus{
		models.OrderShipped,
		models.OrderShipmentCreated,
	}

	// Hitung total pesanan UMKM
	config.DB.Model(&models.Order{}).Where("umkm_id = ?", umkmID).Count(&totalOrders)
	config.DB.Model(&models.Order{}).Where("umkm_id = ? AND status IN ?", umkmID, pendingStatuses).Count(&pendingOrders)

	// Hitung pesanan yang sedang dikirim (shipped)
	config.DB.Model(&models.Order{}).Where("umkm_id = ? AND status IN ?", umkmID, shippedStatuses).Count(&shippedOrders)
	config.DB.Model(&models.Order{}).Where("umkm_id = ? AND status = ?", umkmID, models.OrderCompleted).Count(&completedOrders)
	config.DB.Model(&models.Order{}).
		Where("umkm_id = ? AND status IN ?", umkmID, []models.OrderStatus{models.OrderPaid, models.OrderShipmentCreated, models.OrderProcessing, models.OrderShipped, models.OrderCompleted}).
		Select("COALESCE(SUM(grand_total), 0)").
		Scan(&totalSpending)

	c.JSON(http.StatusOK, gin.H{
		"total_orders":     totalOrders,
		"pending_orders":   pendingOrders,
		"shipped_orders":   shippedOrders,
		"completed_orders": completedOrders,
		"total_spending":   totalSpending,
		"vouchers":         0,                // Belum ada tabel khusus voucher, set default 0
		"points":           totalOrders * 10, // Contoh logika bisnis: tiap transaksi dapat 10 poin
	})
}

func GetUserOrders(c *gin.Context) {
	umkmID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	var orders []models.Order
	query := config.DB.Preload("Product").Preload("Product.Supplier").Preload("Payment").Where("umkm_id = ?", umkmID)

	if status := c.Query("status"); status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	// Preload Product agar info detail produk ikut terbawa
	if err := query.Order("created_at DESC").Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar pesanan"})
		return
	}

	// Cek status review secara dinamis untuk masing-masing pesanan
	for i := range orders {
		var count int64
		config.DB.Model(&models.Review{}).Where("order_id = ?", orders[i].ID).Count(&count)
		orders[i].IsReviewed = count > 0
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   orders,
	})
}

func userProfilePayload(user models.User) gin.H {
	return gin.H{
		"id":            user.ID,
		"business_name": user.BusinessName,
		"email":         user.Email,
		"role":          user.Role,
		"address":       user.Address,
		"category":      user.Category,
		"region":        user.Region,
		"pic_name":      user.PICName,
		"phone":         user.Phone,
		"status":        user.Status,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
	}
}

func getCurrentUMKM(c *gin.Context) (models.User, bool) {
	var user models.User

	umkmID, ok := getAuthenticatedUserID(c)
	if !ok {
		return user, false
	}

	if err := config.DB.Where("id = ? AND role = ?", umkmID, models.RoleUser).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profil UMKM tidak ditemukan"})
			return user, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil profil UMKM"})
		return user, false
	}

	return user, true
}

func GetUserProfile(c *gin.Context) {
	user, ok := getCurrentUMKM(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   userProfilePayload(user),
	})
}

func UpdateUserProfile(c *gin.Context) {
	user, ok := getCurrentUMKM(c)
	if !ok {
		return
	}

	var input userProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama UMKM, email, kategori, PIC, nomor HP, dan alamat wajib diisi"})
		return
	}

	businessName := strings.TrimSpace(input.BusinessName)
	email := strings.ToLower(strings.TrimSpace(input.Email))
	category := strings.TrimSpace(input.Category)
	picName := strings.TrimSpace(input.PICName)
	phone := strings.TrimSpace(input.Phone)
	address := strings.TrimSpace(input.Address)

	if businessName == "" || email == "" || category == "" || picName == "" || phone == "" || address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama UMKM, email, kategori, PIC, nomor HP, dan alamat wajib diisi"})
		return
	}

	var existing models.User
	if err := config.DB.Where("email = ? AND id <> ?", email, user.ID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email sudah digunakan akun lain"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi email"})
		return
	}

	updates := map[string]interface{}{
		"business_name": businessName,
		"email":         email,
		"address":       address,
		"category":      category,
		"pic_name":      picName,
		"phone":         phone,
	}

	if err := config.DB.Model(&user).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui profil UMKM"})
		return
	}

	if err := config.DB.Where("id = ?", user.ID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat ulang profil UMKM"})
		return
	}

	description := "Profil UMKM " + user.BusinessName + " diperbarui"
	_ = services.CreateActivityLog(nil, user.ID, "UPDATE_UMKM_PROFILE", description)
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     user.ID,
		Role:       string(models.RoleUser),
		Title:      "Profil UMKM Diperbarui",
		Message:    description,
		Type:       "umkm_profile_updated",
		SourceType: "user",
		SourceID:   user.ID,
	})
	_, _ = services.CreateRoleActivityNotifications(nil, models.RoleAdmin, models.Notification{
		Title:      "Profil UMKM Diperbarui",
		Message:    description,
		Type:       "umkm_profile_updated",
		SourceType: "user",
		SourceID:   user.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Profil UMKM berhasil diperbarui",
		"data":    userProfilePayload(user),
	})
}

// --- ALGORITMA INTI ---

// KMPMatch mengimplementasikan Knuth-Morris-Pratt untuk pencarian string
func KMPMatch(text string, pattern string) bool {
	text = strings.ToLower(text)
	pattern = strings.ToLower(pattern)

	n := len(text)
	m := len(pattern)
	if m == 0 {
		return true
	}

	lps := make([]int, m)
	lenLps := 0
	i := 1

	for i < m {
		if pattern[i] == pattern[lenLps] {
			lenLps++
			lps[i] = lenLps
			i++
		} else {
			if lenLps != 0 {
				lenLps = lps[lenLps-1]
			} else {
				lps[i] = 0
				i++
			}
		}
	}

	i = 0
	j := 0
	for i < n {
		if pattern[j] == text[i] {
			j++
			i++
		}
		if j == m {
			return true
		} else if i < n && pattern[j] != text[i] {
			if j != 0 {
				j = lps[j-1]
			} else {
				i++
			}
		}
	}
	return false
}

// QuickSortPrice mengimplementasikan Quick Sort untuk mengurutkan harga
func QuickSortPrice(items []models.Product, low, high int, asc bool) {
	if low < high {
		pi := partitionPrice(items, low, high, asc)
		QuickSortPrice(items, low, pi-1, asc)
		QuickSortPrice(items, pi+1, high, asc)
	}
}

func partitionPrice(items []models.Product, low, high int, asc bool) int {
	pivot := items[high].Price
	i := low - 1
	for j := low; j < high; j++ {
		if asc {
			if items[j].Price <= pivot {
				i++
				items[i], items[j] = items[j], items[i]
			}
		} else {
			if items[j].Price >= pivot {
				i++
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	items[i+1], items[high] = items[high], items[i+1]
	return i + 1
}

// IsItemValid mengimplementasikan Binary Search untuk validasi ketersediaan produk (berdasarkan ID)
func IsItemValid(sortedIDs []string, targetID string) bool {
	low, high := 0, len(sortedIDs)-1
	for low <= high {
		mid := low + (high-low)/2
		if sortedIDs[mid] == targetID {
			return true // Item Valid
		} else if sortedIDs[mid] < targetID {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	return false // Item Tidak Ditemukan / Invalid
}

// ----------------------

// GetProducts mengambil katalog produk untuk UMKM dengan fitur Pencarian (KMP) dan Sorting (Quick Sort)
func GetProducts(c *gin.Context) {
	var allProducts []models.Product

	// Tarik seluruh data produk dari database
	if err := config.DB.Preload("Supplier").Find(&allProducts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data produk"})
		return
	}

	// Hitung rating rata-rata & jumlah ulasan untuk tiap produk secara dinamis
	for i := range allProducts {
		var stats struct {
			Average float64
			Count   int64
		}
		config.DB.Model(&models.Review{}).
			Where("product_id = ?", allProducts[i].ID).
			Select("COALESCE(AVG(rating), 0) as average, COUNT(id) as count").
			Scan(&stats)

		allProducts[i].RatingAverage = stats.Average
		allProducts[i].ReviewCount = int(stats.Count)
	}

	// 1. Filtering menggunakan KMP (jika ada query 'search')
	searchQuery := c.Query("search")
	var filteredProducts []models.Product

	if searchQuery != "" {
		for _, product := range allProducts {
			// Mencari kecocokan pattern KMP pada nama produk, kategori, lokasi, atau nama supplier
			if KMPMatch(product.Name, searchQuery) ||
				KMPMatch(product.Category, searchQuery) ||
				KMPMatch(product.Location, searchQuery) ||
				KMPMatch(product.Supplier.BusinessName, searchQuery) {
				filteredProducts = append(filteredProducts, product)
			}
		}
	} else {
		filteredProducts = allProducts
	}

	// 2. Sorting menggunakan Quick Sort (jika ada query 'sort_by')
	sortBy := c.Query("sort_by")
	if len(filteredProducts) > 0 {
		if sortBy == "price_asc" {
			QuickSortPrice(filteredProducts, 0, len(filteredProducts)-1, true)
		} else if sortBy == "price_desc" {
			QuickSortPrice(filteredProducts, 0, len(filteredProducts)-1, false)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Berhasil mengambil katalog produk",
		"data":    filteredProducts,
	})
}

func GetPublicCatalog(c *gin.Context) {
	GetProducts(c)
}

type CreateOrderInput struct {
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
}

func CreateOrder(c *gin.Context) {
	umkmID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	var input CreateOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Tarik semua ID produk yang valid dan pastikan diurutkan dengan aturan Go (ASCII)
	var sortedIDs []string
	if err := config.DB.Model(&models.Product{}).Pluck("id", &sortedIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi produk"})
		return
	}
	sort.Strings(sortedIDs)

	// 2. Validasi ID Produk menggunakan Binary Search (Sesuai Aturan PRD 6.3)
	if !IsItemValid(sortedIDs, input.ItemID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Item ID tidak valid atau tidak ditemukan (Binary Search Check Failed)"})
		return
	}

	// 3. Ambil data produk asli dari DB untuk kalkulasi harga
	var product models.Product
	if err := config.DB.First(&product, "id = ?", input.ItemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
		return
	}

	if product.Stock < input.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           "Jumlah pesanan melebihi stok tersedia.",
			"available_stock": product.Stock,
			"requested_qty":   input.Quantity,
		})
		return
	}

	// 4. Kalkulasi harga pesanan
	totalBasePrice := product.Price * float64(input.Quantity)
	systemFee := totalBasePrice * 0.03
	grandTotal := totalBasePrice + systemFee

	// 5. Buat dan Simpan Order ke Database
	order := models.Order{
		UmkmID:         umkmID,
		SupplierID:     product.SupplierID,
		ProductID:      product.ID,
		Quantity:       input.Quantity,
		TotalBasePrice: totalBasePrice,
		SystemFee:      systemFee,
		GrandTotal:     grandTotal,
		Status:         models.OrderPendingSupplierConfirmation,
		StockDeducted:  false,
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		return services.CreateFinanceLog(tx, order, nil, "order_created", "Order bahan dibuat melalui endpoint UMKM lama")
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat pesanan"})
		return
	}

	description := "UMKM membuat pesanan produk " + product.Name + " sebanyak " + strconv.Itoa(order.Quantity) + " unit"
	_ = services.CreateActivityLog(nil, umkmID, "CREATE_ORDER", description)
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     umkmID,
		Role:       string(models.RoleUser),
		Title:      "Pesanan Dibuat",
		Message:    description,
		Type:       "order_created",
		SourceType: "order",
		SourceID:   order.ID,
	})
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     product.SupplierID,
		Role:       string(models.RoleSupplier),
		Title:      "Pesanan Baru Masuk",
		Message:    description,
		Type:       "order_created",
		SourceType: "order",
		SourceID:   order.ID,
	})
	_, _ = services.CreateRoleActivityNotifications(nil, models.RoleAdmin, models.Notification{
		Title:      "Pesanan Baru Dibuat",
		Message:    description,
		Type:       "order_created",
		SourceType: "order",
		SourceID:   order.ID,
	})

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Pesanan berhasil dibuat",
		"data":    order,
	})
}

func CancelOrder(c *gin.Context) {
	umkmID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	orderID := c.Param("id")
	var order models.Order
	if err := config.DB.Preload("Product").Where("id = ? AND umkm_id = ?", orderID, umkmID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pesanan tidak ditemukan"})
		return
	}

	if order.Status != models.OrderPending && order.Status != models.OrderPendingSupplierConfirmation {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pesanan hanya bisa dibatalkan saat status pending"})
		return
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if order.StockDeducted {
			if err := tx.Model(&models.Product{}).
				Where("id = ?", order.ProductID).
				Update("stock", gorm.Expr("stock + ?", order.Quantity)).Error; err != nil {
				return err
			}
		}
		return tx.Model(&order).Update("status", models.OrderCancelled).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membatalkan pesanan"})
		return
	}

	description := "UMKM membatalkan pesanan produk " + order.Product.Name
	_ = services.CreateActivityLog(nil, umkmID, "CANCEL_ORDER", description)
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     umkmID,
		Role:       string(models.RoleUser),
		Title:      "Pesanan Dibatalkan",
		Message:    description,
		Type:       "order_cancelled",
		SourceType: "order",
		SourceID:   order.ID,
	})
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     order.SupplierID,
		Role:       string(models.RoleSupplier),
		Title:      "Pesanan Dibatalkan UMKM",
		Message:    description,
		Type:       "order_cancelled",
		SourceType: "order",
		SourceID:   order.ID,
	})
	_, _ = services.CreateRoleActivityNotifications(nil, models.RoleAdmin, models.Notification{
		Title:      "Pesanan Dibatalkan",
		Message:    description,
		Type:       "order_cancelled",
		SourceType: "order",
		SourceID:   order.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Pesanan berhasil dibatalkan",
	})
}

func CompleteOrder(c *gin.Context) {
	umkmID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	orderID := c.Param("id")
	var order models.Order
	if err := config.DB.Preload("Product").Where("id = ? AND umkm_id = ?", orderID, umkmID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pesanan tidak ditemukan"})
		return
	}

	if order.Status != models.OrderShipped && order.Status != models.OrderShipmentCreated && order.Status != models.OrderPaid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pesanan belum dapat dikonfirmasi selesai"})
		return
	}

	if err := config.DB.Model(&order).Update("status", models.OrderCompleted).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan pesanan"})
		return
	}

	description := "UMKM mengonfirmasi pesanan produk " + order.Product.Name + " sebagai selesai"
	_ = services.CreateActivityLog(nil, umkmID, "COMPLETE_ORDER", description)
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     umkmID,
		Role:       string(models.RoleUser),
		Title:      "Pesanan Selesai",
		Message:    description,
		Type:       "order_completed",
		SourceType: "order",
		SourceID:   order.ID,
	})
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     order.SupplierID,
		Role:       string(models.RoleSupplier),
		Title:      "Pesanan Dikonfirmasi Selesai",
		Message:    description,
		Type:       "order_completed",
		SourceType: "order",
		SourceID:   order.ID,
	})
	_, _ = services.CreateRoleActivityNotifications(nil, models.RoleAdmin, models.Notification{
		Title:      "Pesanan Selesai",
		Message:    description,
		Type:       "order_completed",
		SourceType: "order",
		SourceID:   order.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Pesanan berhasil dikonfirmasi selesai",
	})
}
