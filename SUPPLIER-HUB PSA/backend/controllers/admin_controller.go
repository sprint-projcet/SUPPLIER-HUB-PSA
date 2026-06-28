package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"supplierhub-backend/config"
	"supplierhub-backend/models"
	"supplierhub-backend/services"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type adminSupplierResponse struct {
	ID           string `json:"id"`
	BusinessName string `json:"business_name"`
	Email        string `json:"email"`
	Address      string `json:"address"`
	Category     string `json:"category"`
	Region       string `json:"region"`
	DocumentURL  string `json:"document_url"`
	Status       string `json:"status"`
	ProductCount int64  `json:"product_count"`
}

type createAdminLogInput struct {
	Action      string `json:"action" binding:"required"`
	Description string `json:"description" binding:"required"`
	UserID      string `json:"user_id"`
}

type updateSupplierStatusInput struct {
	Status string `json:"status" binding:"required"`
}

type updateAdminProductStockInput struct {
	Stock int    `json:"stock" binding:"min=0"`
	Note  string `json:"note"`
}

type stockAlertInput struct {
	Message string `json:"message"`
}

type adminStockProductResponse struct {
	models.Product
	LastStockAlertAt      *time.Time `json:"last_stock_alert_at,omitempty"`
	LastStockAlertID      string     `json:"last_stock_alert_id,omitempty"`
	StockAlertCount       int64      `json:"stock_alert_count"`
	NeedsReminder         bool       `json:"needs_reminder"`
	UpdatedAfterLastAlert bool       `json:"updated_after_last_alert"`
}

type updateAdminProfileInput struct {
	BusinessName    string `json:"business_name" binding:"required"`
	Email           string `json:"email" binding:"required,email"`
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func latestProductStockAlert(productID string) (models.Notification, bool, error) {
	var notification models.Notification
	err := config.DB.
		Where("role = ? AND type = ? AND source_type = ? AND source_id = ?", string(models.RoleSupplier), "low_stock", "product", productID).
		Order("created_at DESC").
		First(&notification).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return notification, false, nil
	}

	return notification, err == nil, err
}

func countProductStockAlerts(productID string) (int64, error) {
	var count int64
	err := config.DB.Model(&models.Notification{}).
		Where("role = ? AND type = ? AND source_type = ? AND source_id = ?", string(models.RoleSupplier), "low_stock", "product", productID).
		Count(&count).Error
	return count, err
}

func productNeedsStockReminder(product models.Product, latestAlert models.Notification) bool {
	return product.Stock <= 10 && !product.UpdatedAt.After(latestAlert.CreatedAt)
}

func GetAdminStats(c *gin.Context) {
	var totalSuppliers int64
	var totalTransactions int64
	var activeOrders int64
	var pendingSuppliers int64
	var totalRevenue float64
	var systemFeeRevenue float64
	var recentOrders []models.Order

	activeStatuses := []models.OrderStatus{
		models.OrderPending,
		models.OrderPendingSupplierConfirmation,
		models.OrderSupplierConfirmed,
		models.OrderPaymentPending,
		models.OrderPaid,
		models.OrderProcessing,
		models.OrderShipped,
		models.OrderShipmentCreated,
	}

	if err := config.DB.Model(&models.User{}).Where("role = ?", models.RoleSupplier).Count(&totalSuppliers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung total supplier"})
		return
	}
	if err := config.DB.Model(&models.User{}).Where("role = ? AND status = ?", models.RoleSupplier, "pending").Count(&pendingSuppliers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung supplier pending"})
		return
	}
	if err := config.DB.Model(&models.Order{}).Count(&totalTransactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung total transaksi"})
		return
	}
	if err := config.DB.Model(&models.Order{}).Where("status IN ?", activeStatuses).Count(&activeOrders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung pesanan aktif"})
		return
	}
	if err := config.DB.Model(&models.Order{}).
		Where("status <> ?", models.OrderCancelled).
		Select("COALESCE(SUM(grand_total), 0)").
		Scan(&totalRevenue).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung total revenue"})
		return
	}
	if err := config.DB.Model(&models.Order{}).
		Where("status <> ?", models.OrderCancelled).
		Select("COALESCE(SUM(system_fee), 0)").
		Scan(&systemFeeRevenue).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung biaya layanan"})
		return
	}
	if err := config.DB.Preload("Product").Order("created_at DESC").Limit(5).Find(&recentOrders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil transaksi terbaru"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":             "success",
		"total_suppliers":    totalSuppliers,
		"pending_suppliers":  pendingSuppliers,
		"total_transactions": totalTransactions,
		"active_orders":      activeOrders,
		"total_revenue":      totalRevenue,
		"system_fee_revenue": systemFeeRevenue,
		"revenue_growth":     "0%",
		"recent_orders":      recentOrders,
	})
}

func GetAdminSuppliers(c *gin.Context) {
	var suppliers []models.User
	query := config.DB.Where("role = ?", models.RoleSupplier)

	if status := strings.TrimSpace(c.Query("status")); status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		likeSearch := "%" + search + "%"
		query = query.Where(
			"business_name LIKE ? OR email LIKE ? OR category LIKE ? OR region LIKE ?",
			likeSearch,
			likeSearch,
			likeSearch,
			likeSearch,
		)
	}

	if err := query.Order("created_at DESC").Find(&suppliers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar supplier"})
		return
	}

	response := make([]adminSupplierResponse, 0, len(suppliers))
	for _, supplier := range suppliers {
		var productCount int64
		if err := config.DB.Model(&models.Product{}).Where("supplier_id = ?", supplier.ID).Count(&productCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung produk supplier"})
			return
		}

		response = append(response, adminSupplierResponse{
			ID:           supplier.ID,
			BusinessName: supplier.BusinessName,
			Email:        supplier.Email,
			Address:      supplier.Address,
			Category:     supplier.Category,
			Region:       supplier.Region,
			DocumentURL:  supplier.DocumentURL,
			Status:       supplier.Status,
			ProductCount: productCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   response,
	})
}

func VerifySupplier(c *gin.Context) {
	supplierID := c.Param("id")
	var supplier models.User

	if err := config.DB.Where("id = ? AND role = ?", supplierID, models.RoleSupplier).First(&supplier).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Supplier tidak ditemukan"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data supplier"})
		return
	}

	if err := config.DB.Model(&supplier).Update("status", "active").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memverifikasi supplier"})
		return
	}

	adminID, _ := c.Get("user_id")
	description := "Supplier " + supplier.BusinessName + " diverifikasi oleh admin"
	_ = services.CreateActivityLog(nil, toString(adminID), "VERIFY_SUPPLIER", description)
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     supplier.ID,
		Role:       string(models.RoleSupplier),
		Title:      "Akun Supplier Diverifikasi",
		Message:    "Akun toko " + supplier.BusinessName + " sudah diverifikasi dan dapat mengelola produk.",
		Type:       "supplier_verified",
		SourceType: "supplier",
		SourceID:   supplier.ID,
	})
	_, _ = services.CreateRoleActivityNotifications(nil, models.RoleAdmin, models.Notification{
		Title:      "Supplier Diverifikasi",
		Message:    description,
		Type:       "supplier_verified",
		SourceType: "supplier",
		SourceID:   supplier.ID,
	})

	supplier.Status = "active"
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Supplier berhasil diverifikasi",
		"data": gin.H{
			"id":            supplier.ID,
			"business_name": supplier.BusinessName,
			"email":         supplier.Email,
			"status":        supplier.Status,
		},
	})
}

func UpdateSupplierStatus(c *gin.Context) {
	supplierID := c.Param("id")

	var input updateSupplierStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status supplier wajib diisi"})
		return
	}

	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status != "active" && status != "pending" && status != "suspended" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status supplier tidak valid"})
		return
	}

	var supplier models.User
	if err := config.DB.Where("id = ? AND role = ?", supplierID, models.RoleSupplier).First(&supplier).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Supplier tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil supplier"})
		return
	}

	if err := config.DB.Model(&supplier).Update("status", status).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status supplier"})
		return
	}

	adminID, _ := c.Get("user_id")
	description := "Status supplier " + supplier.BusinessName + " diubah menjadi " + status
	_ = services.CreateActivityLog(nil, toString(adminID), "UPDATE_SUPPLIER_STATUS", description)
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     supplier.ID,
		Role:       string(models.RoleSupplier),
		Title:      "Status Supplier Diperbarui",
		Message:    description,
		Type:       "supplier_status_updated",
		SourceType: "supplier",
		SourceID:   supplier.ID,
	})
	_, _ = services.CreateRoleActivityNotifications(nil, models.RoleAdmin, models.Notification{
		Title:      "Status Supplier Diperbarui",
		Message:    description,
		Type:       "supplier_status_updated",
		SourceType: "supplier",
		SourceID:   supplier.ID,
	})

	supplier.Status = status
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Status supplier berhasil diperbarui",
		"data": gin.H{
			"id":            supplier.ID,
			"business_name": supplier.BusinessName,
			"email":         supplier.Email,
			"status":        supplier.Status,
		},
	})
}

func GetAdminProfile(c *gin.Context) {
	adminID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	var admin models.User
	if err := config.DB.Where("id = ? AND role = ?", adminID, models.RoleAdmin).First(&admin).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profil admin tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"id":            admin.ID,
			"business_name": admin.BusinessName,
			"email":         admin.Email,
			"role":          admin.Role,
			"status":        admin.Status,
			"created_at":    admin.CreatedAt,
			"updated_at":    admin.UpdatedAt,
		},
	})
}

func UpdateAdminProfile(c *gin.Context) {
	adminID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	var input updateAdminProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama dan email admin wajib diisi"})
		return
	}

	businessName := strings.TrimSpace(input.BusinessName)
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if businessName == "" || email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama dan email admin wajib diisi"})
		return
	}

	var admin models.User
	if err := config.DB.Where("id = ? AND role = ?", adminID, models.RoleAdmin).First(&admin).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profil admin tidak ditemukan"})
		return
	}

	var existing models.User
	if err := config.DB.Where("email = ? AND id <> ?", email, admin.ID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email sudah digunakan akun lain"})
		return
	}

	updates := map[string]interface{}{
		"business_name": businessName,
		"email":         email,
	}

	newPassword := strings.TrimSpace(input.NewPassword)
	if newPassword != "" {
		if len(newPassword) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password baru minimal 6 karakter"})
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(input.CurrentPassword)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password saat ini tidak sesuai"})
			return
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password baru"})
			return
		}
		updates["password_hash"] = string(hashedPassword)
	}

	if err := config.DB.Model(&admin).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui profil admin"})
		return
	}

	description := "Profil admin " + businessName + " diperbarui"
	_ = services.CreateActivityLog(nil, admin.ID, "UPDATE_ADMIN_PROFILE", description)
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     admin.ID,
		Role:       string(models.RoleAdmin),
		Title:      "Profil Admin Diperbarui",
		Message:    description,
		Type:       "admin_profile_updated",
		SourceType: "admin",
		SourceID:   admin.ID,
	})

	admin.BusinessName = businessName
	admin.Email = email
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Profil admin berhasil diperbarui",
		"data": gin.H{
			"id":            admin.ID,
			"business_name": admin.BusinessName,
			"email":         admin.Email,
			"role":          admin.Role,
			"status":        admin.Status,
		},
	})
}

func GetAdminLogs(c *gin.Context) {
	limit := 50
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit < 1 || parsedLimit > 200 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Limit log harus di antara 1 sampai 200"})
			return
		}
		limit = parsedLimit
	}

	var logs []models.Log
	if err := config.DB.Order("created_at DESC").Limit(limit).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil log sistem"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   logs,
	})
}

func CreateAdminLog(c *gin.Context) {
	var input createAdminLogInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Action dan description wajib diisi"})
		return
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		adminID, _ := c.Get("user_id")
		userID = toString(adminID)
	}

	logEntry := models.Log{
		UserID:      userID,
		Action:      strings.ToUpper(strings.TrimSpace(input.Action)),
		Description: strings.TrimSpace(input.Description),
	}

	if err := config.DB.Create(&logEntry).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan log sistem"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Log aktivitas berhasil dicatat",
		"data":    logEntry,
	})
}

func GetAdminFinanceSummary(c *gin.Context) {
	successStatuses := []models.OrderStatus{
		models.OrderPaid,
		models.OrderShipmentCreated,
		models.OrderProcessing,
		models.OrderShipped,
		models.OrderCompleted,
	}

	var transactionCount int64
	var grossOrderValue float64
	var baseOrderValue float64
	var systemFeeRevenue float64
	var supplierNetRevenue float64
	var pendingPaymentValue float64
	var successPayments int64
	var pendingPayments int64
	var failedPayments int64
	var latestOrders []models.Order

	if err := config.DB.Model(&models.Order{}).
		Where("status IN ?", successStatuses).
		Count(&transactionCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung transaksi keuangan"})
		return
	}

	if err := config.DB.Model(&models.Order{}).
		Where("status IN ?", successStatuses).
		Select("COALESCE(SUM(grand_total), 0)").
		Scan(&grossOrderValue).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung gross revenue"})
		return
	}

	if err := config.DB.Model(&models.Order{}).
		Where("status IN ?", successStatuses).
		Select("COALESCE(SUM(total_base_price), 0)").
		Scan(&baseOrderValue).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung nilai order"})
		return
	}

	if err := config.DB.Model(&models.Order{}).
		Where("status IN ?", successStatuses).
		Select("COALESCE(SUM(system_fee), 0)").
		Scan(&systemFeeRevenue).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung fee layanan"})
		return
	}

	supplierNetRevenue = baseOrderValue - systemFeeRevenue

	if err := config.DB.Model(&models.Payment{}).Where("status = ?", models.PaymentSuccess).Count(&successPayments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung pembayaran sukses"})
		return
	}
	if err := config.DB.Model(&models.Payment{}).Where("status = ?", models.PaymentPending).Count(&pendingPayments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung pembayaran pending"})
		return
	}
	if err := config.DB.Model(&models.Payment{}).Where("status = ?", models.PaymentFailed).Count(&failedPayments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung pembayaran gagal"})
		return
	}
	if err := config.DB.Model(&models.Payment{}).
		Where("status = ?", models.PaymentPending).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&pendingPaymentValue).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung nilai pembayaran pending"})
		return
	}

	if err := config.DB.Preload("Product").Preload("Supplier").Preload("Umkm").
		Where("status IN ?", successStatuses).
		Order("updated_at DESC").
		Limit(10).
		Find(&latestOrders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil transaksi terbaru"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":                "success",
		"transaction_count":     transactionCount,
		"gross_order_value":     grossOrderValue,
		"base_order_value":      baseOrderValue,
		"system_fee_revenue":    systemFeeRevenue,
		"supplier_net_revenue":  supplierNetRevenue,
		"pending_payment_value": pendingPaymentValue,
		"success_payments":      successPayments,
		"pending_payments":      pendingPayments,
		"failed_payments":       failedPayments,
		"fee_rate":              0.03,
		"latest_orders":         latestOrders,
	})
}

func GetAdminStockSummary(c *gin.Context) {
	var totalProducts int64
	var activeProducts int64
	var outOfStockProducts int64
	var lowStockProducts int64
	var totalStock int64
	var products []models.Product
	var pendingStockReminders int64

	if err := config.DB.Model(&models.Product{}).Count(&totalProducts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung produk"})
		return
	}

	if err := config.DB.Model(&models.Product{}).Where("stock > 0").Count(&activeProducts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung produk aktif"})
		return
	}

	if err := config.DB.Model(&models.Product{}).Where("stock <= 0").Count(&outOfStockProducts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung stok kosong"})
		return
	}

	if err := config.DB.Model(&models.Product{}).Where("stock > 0 AND stock <= 10").Count(&lowStockProducts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung stok menipis"})
		return
	}

	if err := config.DB.Model(&models.Product{}).
		Select("COALESCE(SUM(stock), 0)").
		Scan(&totalStock).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung total stok"})
		return
	}

	if err := config.DB.Preload("Supplier").Order("stock ASC").Limit(100).Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data stok produk"})
		return
	}

	response := make([]adminStockProductResponse, 0, len(products))
	for _, product := range products {
		item := adminStockProductResponse{Product: product}

		alertCount, err := countProductStockAlerts(product.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung peringatan stok"})
			return
		}
		item.StockAlertCount = alertCount

		latestAlert, hasAlert, err := latestProductStockAlert(product.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil status peringatan stok"})
			return
		}
		if hasAlert {
			lastAlertAt := latestAlert.CreatedAt
			item.LastStockAlertAt = &lastAlertAt
			item.LastStockAlertID = latestAlert.ID
			item.UpdatedAfterLastAlert = product.UpdatedAt.After(latestAlert.CreatedAt)
			item.NeedsReminder = productNeedsStockReminder(product, latestAlert)
			if item.NeedsReminder {
				pendingStockReminders++
			}
		}

		response = append(response, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":                  "success",
		"total_products":          totalProducts,
		"active_products":         activeProducts,
		"out_of_stock_products":   outOfStockProducts,
		"low_stock_products":      lowStockProducts,
		"pending_stock_reminders": pendingStockReminders,
		"total_stock":             totalStock,
		"data":                    response,
	})
}

func GetAdminNotifications(c *gin.Context) {
	adminID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	query := config.DB.Where("user_id = ? AND role = ?", adminID, string(models.RoleAdmin))
	if c.Query("unread_only") == "true" {
		query = query.Where("is_read = ?", false)
	}

	var notifications []models.Notification
	if err := query.Order("created_at DESC").Limit(50).Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil notifikasi admin"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   notifications,
	})
}

func MarkAdminNotificationRead(c *gin.Context) {
	adminID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	now := time.Now()
	result := config.DB.Model(&models.Notification{}).
		Where("id = ? AND user_id = ? AND role = ?", c.Param("id"), adminID, string(models.RoleAdmin)).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menandai notifikasi admin"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notifikasi admin tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Notifikasi admin sudah dibaca",
	})
}

func UpdateAdminProductStock(c *gin.Context) {
	productID := c.Param("id")

	var input updateAdminProductStockInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stok tidak valid"})
		return
	}

	var product models.Product
	if err := config.DB.Preload("Supplier").Where("id = ?", productID).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil produk"})
		return
	}

	oldStock := product.Stock
	if err := config.DB.Model(&product).Update("stock", input.Stock).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui stok produk"})
		return
	}

	adminID, _ := c.Get("user_id")
	note := strings.TrimSpace(input.Note)
	if note == "" {
		note = "Penyesuaian stok oleh admin"
	}
	description := "Produk " + product.Name + " diubah stoknya dari " +
		strconv.Itoa(oldStock) + " menjadi " + strconv.Itoa(input.Stock) +
		" oleh admin " + toString(adminID) + ". " + note
	_ = services.CreateActivityLog(nil, toString(adminID), "UPDATE_PRODUCT_STOCK", description)
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     product.SupplierID,
		Role:       string(models.RoleSupplier),
		Title:      "Stok Produk Disesuaikan Admin",
		Message:    description,
		Type:       "stock_adjusted_by_admin",
		SourceType: "product",
		SourceID:   product.ID,
	})
	_, _ = services.CreateRoleActivityNotifications(nil, models.RoleAdmin, models.Notification{
		Title:      "Stok Produk Diperbarui",
		Message:    description,
		Type:       "stock_adjusted_by_admin",
		SourceType: "product",
		SourceID:   product.ID,
	})

	product.Stock = input.Stock
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Stok produk berhasil diperbarui",
		"data":    product,
	})
}

func SendLowStockAlert(c *gin.Context) {
	productID := c.Param("id")

	var input stockAlertInput
	_ = c.ShouldBindJSON(&input)

	var product models.Product
	if err := config.DB.Preload("Supplier").Where("id = ?", productID).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil produk"})
		return
	}

	if product.SupplierID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Produk belum memiliki supplier"})
		return
	}

	latestAlert, hasAlert, err := latestProductStockAlert(product.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil status peringatan stok"})
		return
	}
	isReminder := hasAlert && productNeedsStockReminder(product, latestAlert)
	title := "Peringatan Stok Menipis"
	if isReminder {
		title = "Pengingat Stok Menipis"
	}

	message := strings.TrimSpace(input.Message)
	if message == "" {
		if isReminder {
			message = "Pengingat: stok produk " + product.Name + " masih tersisa " + strconv.Itoa(product.Stock) + " unit dan belum berubah sejak peringatan terakhir. Mohon segera perbarui stok dari dashboard supplier."
		} else {
			message = "Stok produk " + product.Name + " tersisa " + strconv.Itoa(product.Stock) + " unit. Mohon segera perbarui stok dari dashboard supplier."
		}
	}

	notification, err := services.CreateActivityNotification(nil, models.Notification{
		UserID:     product.SupplierID,
		Role:       string(models.RoleSupplier),
		Title:      title,
		Message:    message,
		Type:       "low_stock",
		SourceType: "product",
		SourceID:   product.ID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengirim peringatan stok"})
		return
	}

	adminID, _ := c.Get("user_id")
	action := "SEND_LOW_STOCK_ALERT"
	description := "Admin mengirim peringatan stok untuk produk " + product.Name + " kepada supplier " + product.Supplier.BusinessName
	if isReminder {
		action = "RESEND_LOW_STOCK_ALERT"
		description = "Admin mengirim ulang peringatan stok untuk produk " + product.Name + " kepada supplier " + product.Supplier.BusinessName
	}
	_ = services.CreateActivityLog(nil, toString(adminID), action, description)

	c.JSON(http.StatusCreated, gin.H{
		"status":      "success",
		"message":     "Peringatan stok berhasil dikirim ke supplier",
		"is_reminder": isReminder,
		"data":        notification,
	})
}

func toString(value interface{}) string {
	if str, ok := value.(string); ok {
		return str
	}

	return ""
}
