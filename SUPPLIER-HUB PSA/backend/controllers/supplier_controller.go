package controllers

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"supplierhub-backend/config"
	"supplierhub-backend/models"
	"supplierhub-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type updateSupplierOrderStatusInput struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status" binding:"required"`
}

type supplierProfileInput struct {
	BusinessName string `json:"business_name" binding:"required"`
	Address      string `json:"address" binding:"required"`
	Category     string `json:"category" binding:"required"`
	Region       string `json:"region" binding:"required"`
}

func supplierProfilePayload(supplier models.User) gin.H {
	return gin.H{
		"id":            supplier.ID,
		"business_name": supplier.BusinessName,
		"email":         supplier.Email,
		"role":          supplier.Role,
		"address":       supplier.Address,
		"category":      supplier.Category,
		"region":        supplier.Region,
		"document_url":  supplier.DocumentURL,
		"status":        supplier.Status,
		"created_at":    supplier.CreatedAt,
		"updated_at":    supplier.UpdatedAt,
	}
}

func getCurrentSupplier(c *gin.Context) (models.User, bool) {
	var supplier models.User

	supplierID, ok := getAuthenticatedUserID(c)
	if !ok {
		return supplier, false
	}

	if err := config.DB.Where("id = ? AND role = ?", supplierID, models.RoleSupplier).First(&supplier).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profil supplier tidak ditemukan"})
			return supplier, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil profil supplier"})
		return supplier, false
	}

	return supplier, true
}

func GetSupplierProfile(c *gin.Context) {
	supplier, ok := getCurrentSupplier(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   supplierProfilePayload(supplier),
	})
}

func UpdateSupplierProfile(c *gin.Context) {
	supplier, ok := getCurrentSupplier(c)
	if !ok {
		return
	}

	var input supplierProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama toko, kategori, wilayah, dan alamat wajib diisi"})
		return
	}

	businessName := strings.TrimSpace(input.BusinessName)
	category := strings.TrimSpace(input.Category)
	region := strings.TrimSpace(input.Region)
	address := strings.TrimSpace(input.Address)
	if businessName == "" || category == "" || region == "" || address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama toko, kategori, wilayah, dan alamat wajib diisi"})
		return
	}

	updates := map[string]interface{}{
		"business_name": businessName,
		"address":       address,
		"category":      category,
		"region":        region,
	}

	if err := config.DB.Model(&supplier).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui profil supplier"})
		return
	}

	if err := config.DB.Where("id = ?", supplier.ID).First(&supplier).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat ulang profil supplier"})
		return
	}

	description := "Profil toko " + supplier.BusinessName + " diperbarui"
	_ = services.CreateActivityLog(nil, supplier.ID, "UPDATE_SUPPLIER_PROFILE", description)
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     supplier.ID,
		Role:       string(models.RoleSupplier),
		Title:      "Profil Toko Diperbarui",
		Message:    description,
		Type:       "supplier_profile_updated",
		SourceType: "supplier",
		SourceID:   supplier.ID,
	})
	_, _ = services.CreateRoleActivityNotifications(nil, models.RoleAdmin, models.Notification{
		Title:      "Profil Supplier Diperbarui",
		Message:    description,
		Type:       "supplier_profile_updated",
		SourceType: "supplier",
		SourceID:   supplier.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Profil toko berhasil diperbarui",
		"data":    supplierProfilePayload(supplier),
	})
}

func GetSupplierStats(c *gin.Context) {
	supplierID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	var completedOrders int64
	var pendingOrders int64
	var activeProducts int64
	var totalStock int64
	var supplierRevenue float64
	pendingStatuses := []models.OrderStatus{
		models.OrderPending,
		models.OrderPendingSupplierConfirmation,
		models.OrderSupplierConfirmed,
		models.OrderPaymentPending,
	}
	revenueStatuses := []models.OrderStatus{
		models.OrderPaid,
		models.OrderShipmentCreated,
		models.OrderCompleted,
	}

	if err := config.DB.Model(&models.Order{}).
		Where("supplier_id = ? AND status = ?", supplierID, models.OrderCompleted).
		Count(&completedOrders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung pesanan selesai"})
		return
	}

	if err := config.DB.Model(&models.Order{}).
		Where("supplier_id = ? AND status IN ?", supplierID, pendingStatuses).
		Count(&pendingOrders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung pesanan pending"})
		return
	}

	if err := config.DB.Model(&models.Product{}).
		Where("supplier_id = ? AND stock > 0", supplierID).
		Count(&activeProducts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung produk aktif"})
		return
	}

	if err := config.DB.Model(&models.Product{}).
		Where("supplier_id = ?", supplierID).
		Select("COALESCE(SUM(stock), 0)").
		Scan(&totalStock).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung total stok"})
		return
	}

	if err := config.DB.Model(&models.Order{}).
		Where("supplier_id = ? AND status IN ?", supplierID, revenueStatuses).
		Select("COALESCE(SUM(total_base_price - system_fee), 0)").
		Scan(&supplierRevenue).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung pendapatan supplier"})
		return
	}

	// Hitung rata-rata rating nyata milik supplier ini
	var avgRating float64
	config.DB.Model(&models.Review{}).
		Joins("JOIN products ON reviews.product_id = products.id").
		Where("products.supplier_id = ?", supplierID).
		Select("COALESCE(AVG(reviews.rating), 0)").
		Scan(&avgRating)

	c.JSON(http.StatusOK, gin.H{
		"status":           "success",
		"total_revenue":    supplierRevenue,
		"completed_orders": completedOrders,
		"active_products":  activeProducts,
		"pending_orders":   pendingOrders,
		"total_stock":      totalStock,
		"stock":            activeProducts,  // kompatibel dengan dashboard lama
		"new_orders":       pendingOrders,   // kompatibel dengan dashboard lama
		"revenue_rp":       supplierRevenue, // kompatibel dengan dashboard lama
		"rating":           avgRating,
	})
}

func GetSupplierProducts(c *gin.Context) {
	supplierID, _ := c.Get("user_id")

	var products []models.Product
	if err := config.DB.Where("supplier_id = ?", supplierID).Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data produk"})
		return
	}

	c.JSON(http.StatusOK, products)
}

func CreateProduct(c *gin.Context) {
	supplier, ok := getCurrentSupplier(c)
	if !ok {
		return
	}

	if supplier.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akun supplier Anda belum aktif atau belum diverifikasi oleh admin."})
		return
	}

	supplierRegion := strings.TrimSpace(supplier.Region)
	if supplierRegion == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lengkapi wilayah supply toko terlebih dahulu sebelum menambahkan produk."})
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	category := strings.TrimSpace(c.PostForm("category"))
	priceStr := strings.TrimSpace(c.PostForm("price"))
	stockStr := strings.TrimSpace(c.PostForm("stock"))
	description := strings.TrimSpace(c.PostForm("description"))

	if name == "" || category == "" || priceStr == "" || stockStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama, kategori, harga, dan stok produk wajib diisi"})
		return
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil || price < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Harga produk tidak valid"})
		return
	}

	stock, err := strconv.Atoi(stockStr)
	if err != nil || stock < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stok produk tidak valid"})
		return
	}

	// File Upload Handling
	file, err := c.FormFile("image")
	var imageURL string
	if err == nil {
		filename := uuid.New().String() + filepath.Ext(file.Filename)
		uploadPath := "uploads/" + filename
		if err := c.SaveUploadedFile(file, uploadPath); err == nil {
			imageURL = config.PublicURL(uploadPath)
		}
	}

	input := models.Product{
		SupplierID:  supplier.ID,
		Name:        name,
		Category:    category,
		Price:       price,
		Stock:       stock,
		Description: description,
		Location:    supplierRegion,
		ImageURL:    imageURL,
	}

	if err := config.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan produk: " + err.Error()})
		return
	}

	activityDescription := "Supplier " + supplier.BusinessName + " menambahkan produk " + input.Name + " dengan stok " + strconv.Itoa(input.Stock) + " unit"
	_ = services.CreateActivityLog(nil, supplier.ID, "CREATE_PRODUCT", activityDescription)
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     supplier.ID,
		Role:       string(models.RoleSupplier),
		Title:      "Produk Ditambahkan",
		Message:    activityDescription,
		Type:       "product_created",
		SourceType: "product",
		SourceID:   input.ID,
	})
	_, _ = services.CreateRoleActivityNotifications(nil, models.RoleAdmin, models.Notification{
		Title:      "Produk Supplier Ditambahkan",
		Message:    activityDescription,
		Type:       "product_created",
		SourceType: "product",
		SourceID:   input.ID,
	})

	c.JSON(http.StatusCreated, gin.H{
		"message": "Produk berhasil ditambahkan",
		"data":    input,
	})
}

func UpdateProduct(c *gin.Context) {
	supplier, ok := getCurrentSupplier(c)
	if !ok {
		return
	}

	if supplier.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akun supplier Anda belum aktif atau belum diverifikasi oleh admin."})
		return
	}

	supplierRegion := strings.TrimSpace(supplier.Region)
	if supplierRegion == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lengkapi wilayah supply toko terlebih dahulu sebelum mengubah produk."})
		return
	}

	productID := c.Param("id")

	var product models.Product
	if err := config.DB.Where("id = ? AND supplier_id = ?", productID, supplier.ID).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan atau Anda tidak berwenang"})
		return
	}
	oldStock := product.Stock
	oldName := product.Name

	name := c.PostForm("name")
	category := c.PostForm("category")
	priceStr := c.PostForm("price")
	stockStr := c.PostForm("stock")
	description := c.PostForm("description")
	stockChanged := false

	if name != "" {
		product.Name = strings.TrimSpace(name)
	}
	if category != "" {
		product.Category = strings.TrimSpace(category)
	}
	if priceStr != "" {
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			product.Price = price
		}
	}
	if stockStr != "" {
		stock, err := strconv.Atoi(stockStr)
		if err != nil || stock < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Stok produk tidak valid"})
			return
		}
		stockChanged = stock != product.Stock
		product.Stock = stock
	}
	if description != "" {
		product.Description = strings.TrimSpace(description)
	}
	product.Location = supplierRegion

	// File Upload Handling
	file, err := c.FormFile("image")
	if err == nil {
		filename := uuid.New().String() + filepath.Ext(file.Filename)
		uploadPath := "uploads/" + filename
		if err := c.SaveUploadedFile(file, uploadPath); err == nil {
			product.ImageURL = config.PublicURL(uploadPath)
		}
	}

	if err := config.DB.Save(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui produk: " + err.Error()})
		return
	}

	activityDescription := "Supplier " + supplier.BusinessName + " memperbarui produk " + oldName + " menjadi " + product.Name
	if stockChanged {
		activityDescription += ". Stok berubah dari " + strconv.Itoa(oldStock) + " menjadi " + strconv.Itoa(product.Stock) + " unit"
	}
	activityType := "product_updated"
	if stockChanged {
		activityType = "stock_updated"
	}
	_ = services.CreateActivityLog(nil, supplier.ID, strings.ToUpper(activityType), activityDescription)
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     supplier.ID,
		Role:       string(models.RoleSupplier),
		Title:      "Produk Diperbarui",
		Message:    activityDescription,
		Type:       activityType,
		SourceType: "product",
		SourceID:   product.ID,
	})
	_, _ = services.CreateRoleActivityNotifications(nil, models.RoleAdmin, models.Notification{
		Title:      "Produk Supplier Diperbarui",
		Message:    activityDescription,
		Type:       activityType,
		SourceType: "product",
		SourceID:   product.ID,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Produk berhasil diupdate", "data": product})
}

func DeleteProduct(c *gin.Context) {
	supplier, ok := getCurrentSupplier(c)
	if !ok {
		return
	}

	productID := c.Param("id")
	var product models.Product
	if err := config.DB.Where("id = ? AND supplier_id = ?", productID, supplier.ID).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan atau Anda tidak berwenang"})
		return
	}

	if err := config.DB.Delete(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus produk: " + err.Error()})
		return
	}

	description := "Supplier " + supplier.BusinessName + " menghapus produk " + product.Name
	_ = services.CreateActivityLog(nil, supplier.ID, "DELETE_PRODUCT", description)
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     supplier.ID,
		Role:       string(models.RoleSupplier),
		Title:      "Produk Dihapus",
		Message:    description,
		Type:       "product_deleted",
		SourceType: "product",
		SourceID:   product.ID,
	})
	_, _ = services.CreateRoleActivityNotifications(nil, models.RoleAdmin, models.Notification{
		Title:      "Produk Supplier Dihapus",
		Message:    description,
		Type:       "product_deleted",
		SourceType: "product",
		SourceID:   product.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Produk berhasil dihapus",
	})
}

func GetSupplierNotifications(c *gin.Context) {
	supplierID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	query := config.DB.Where("user_id = ? AND role = ?", supplierID, string(models.RoleSupplier))
	if c.Query("unread_only") == "true" {
		query = query.Where("is_read = ?", false)
	}
	switch strings.ToLower(strings.TrimSpace(c.Query("read_status"))) {
	case "read":
		query = query.Where("is_read = ?", true)
	case "unread":
		query = query.Where("is_read = ?", false)
	}

	limit := 100
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit < 1 || parsedLimit > 200 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Limit notifikasi harus di antara 1 sampai 200"})
			return
		}
		limit = parsedLimit
	}

	var notifications []models.Notification
	if err := query.Order("created_at DESC").Limit(limit).Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil notifikasi supplier"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   notifications,
	})
}

func MarkSupplierNotificationRead(c *gin.Context) {
	supplierID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	notificationID := c.Param("id")
	now := time.Now()

	result := config.DB.Model(&models.Notification{}).
		Where("id = ? AND user_id = ? AND role = ?", notificationID, supplierID, string(models.RoleSupplier)).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menandai notifikasi"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notifikasi tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Notifikasi sudah dibaca",
	})
}

func GetSupplierOrders(c *gin.Context) {
	supplierID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	var orders []models.Order
	query := config.DB.
		Preload("Product").
		Preload("Umkm").
		Where("supplier_id = ?", supplierID)

	if status := strings.TrimSpace(c.Query("status")); status != "" && status != "all" {
		normalizedStatus, valid := normalizeSupplierOrderStatus(status)
		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Status pesanan tidak valid"})
			return
		}
		query = query.Where("status = ?", normalizedStatus)
	}

	if err := query.Order("created_at DESC").Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar pesanan supplier"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   orders,
	})
}

func UpdateOrderStatus(c *gin.Context) {
	supplierID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	var input updateSupplierOrderStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID dan status wajib diisi"})
		return
	}

	orderID := c.Param("id")
	if orderID == "" {
		orderID = input.OrderID
	}
	if strings.TrimSpace(orderID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID wajib diisi"})
		return
	}

	nextStatus, valid := normalizeSupplierOrderStatus(input.Status)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status pesanan tidak valid"})
		return
	}

	var order models.Order
	if err := config.DB.Preload("Product").Where("id = ? AND supplier_id = ?", orderID, supplierID).First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pesanan tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil pesanan"})
		return
	}

	if !isValidSupplierStatusTransition(order.Status, nextStatus) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Transisi status tidak valid",
			"current_status": order.Status,
			"next_status":    nextStatus,
		})
		return
	}

	if err := config.DB.Model(&order).Update("status", nextStatus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status pesanan"})
		return
	}

	previousStatus := order.Status
	order.Status = nextStatus
	shortOrderID := order.ID
	if len(shortOrderID) > 8 {
		shortOrderID = shortOrderID[:8]
	}
	orderLabel := "ORD-" + strings.ToUpper(shortOrderID)
	description := "Status pesanan " + orderLabel + " untuk produk " + order.Product.Name + " diubah dari " + string(previousStatus) + " menjadi " + string(nextStatus)
	_ = services.CreateActivityLog(nil, supplierID, "UPDATE_ORDER_STATUS", description)
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     supplierID,
		Role:       string(models.RoleSupplier),
		Title:      "Status Pesanan Diperbarui",
		Message:    description,
		Type:       "order_status_updated",
		SourceType: "order",
		SourceID:   order.ID,
	})
	_, _ = services.CreateActivityNotification(nil, models.Notification{
		UserID:     order.UmkmID,
		Role:       string(models.RoleUser),
		Title:      "Status Pesanan Diperbarui",
		Message:    description,
		Type:       "order_status_updated",
		SourceType: "order",
		SourceID:   order.ID,
	})
	_, _ = services.CreateRoleActivityNotifications(nil, models.RoleAdmin, models.Notification{
		Title:      "Status Pesanan Supplier Diperbarui",
		Message:    description,
		Type:       "order_status_updated",
		SourceType: "order",
		SourceID:   order.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Status pesanan berhasil diperbarui",
		"data":    order,
	})
}

func normalizeSupplierOrderStatus(status string) (models.OrderStatus, bool) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return models.OrderPending, true
	case "pending_supplier_confirmation", "menunggu_konfirmasi":
		return models.OrderPendingSupplierConfirmation, true
	case "rejected_by_supplier", "ditolak_supplier":
		return models.OrderRejectedBySupplier, true
	case "stock_unavailable", "stok_habis":
		return models.OrderStockUnavailable, true
	case "supplier_confirmed", "dikonfirmasi_supplier":
		return models.OrderSupplierConfirmed, true
	case "payment_pending", "menunggu_pembayaran":
		return models.OrderPaymentPending, true
	case "payment_failed", "pembayaran_gagal":
		return models.OrderPaymentFailed, true
	case "shipment_created", "pengiriman_dibuat":
		return models.OrderShipmentCreated, true
	case "paid":
		return models.OrderPaid, true
	case "processed", "processing", "diproses":
		return models.OrderProcessing, true
	case "sent", "shipped", "dikirim":
		return models.OrderShipped, true
	case "completed", "selesai":
		return models.OrderCompleted, true
	case "cancelled", "canceled", "batal":
		return models.OrderCancelled, true
	default:
		return "", false
	}
}

func isValidSupplierStatusTransition(current, next models.OrderStatus) bool {
	if current == next {
		return true
	}

	switch current {
	case models.OrderPending:
		return next == models.OrderProcessing || next == models.OrderCancelled
	case models.OrderPaid:
		return next == models.OrderProcessing
	case models.OrderProcessing:
		return next == models.OrderShipped
	case models.OrderShipped:
		return next == models.OrderCompleted
	default:
		return false
	}
}

type PublicSupplierResponse struct {
	ID            string  `json:"id"`
	BusinessName  string  `json:"business_name"`
	Category      string  `json:"category"`
	Region        string  `json:"region"`
	RatingAverage float64 `json:"rating_average"`
	ReviewCount   int64   `json:"review_count"`
	ProductCount  int64   `json:"product_count"`
}

type SupplierDetailResponse struct {
	ID            string  `json:"id"`
	BusinessName  string  `json:"business_name"`
	Email         string  `json:"email"`
	Address       string  `json:"address"`
	Category      string  `json:"category"`
	Region        string  `json:"region"`
	PICName       string  `json:"pic_name"`
	Phone         string  `json:"phone"`
	Status        string  `json:"status"`
	RatingAverage float64 `json:"rating_average"`
	ReviewCount   int64   `json:"review_count"`
	ProductCount  int64   `json:"product_count"`
}

// GetPublicSuppliers mengambil semua supplier terverifikasi/aktif
func GetPublicSuppliers(c *gin.Context) {
	var suppliers []models.User
	if err := config.DB.Where("role = ? AND status = ?", models.RoleSupplier, "active").Order("created_at DESC").Find(&suppliers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar supplier"})
		return
	}

	response := make([]PublicSupplierResponse, 0, len(suppliers))
	for _, supplier := range suppliers {
		var productCount int64
		config.DB.Model(&models.Product{}).Where("supplier_id = ?", supplier.ID).Count(&productCount)

		var stats struct {
			Average float64
			Count   int64
		}
		config.DB.Model(&models.Review{}).
			Joins("JOIN products ON reviews.product_id = products.id").
			Where("products.supplier_id = ?", supplier.ID).
			Select("COALESCE(AVG(reviews.rating), 0) as average, COUNT(reviews.id) as count").
			Scan(&stats)

		response = append(response, PublicSupplierResponse{
			ID:            supplier.ID,
			BusinessName:  supplier.BusinessName,
			Category:      supplier.Category,
			Region:        supplier.Region,
			RatingAverage: stats.Average,
			ReviewCount:   stats.Count,
			ProductCount:  productCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   response,
	})
}

// GetSupplierDetail mengambil data detail lengkap supplier (wajib login)
func GetSupplierDetail(c *gin.Context) {
	supplierID := c.Param("id")
	var supplier models.User
	if err := config.DB.Where("id = ? AND role = ?", supplierID, models.RoleSupplier).First(&supplier).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Supplier tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data supplier"})
		return
	}

	var productCount int64
	config.DB.Model(&models.Product{}).Where("supplier_id = ?", supplier.ID).Count(&productCount)

	var stats struct {
		Average float64
		Count   int64
	}
	config.DB.Model(&models.Review{}).
		Joins("JOIN products ON reviews.product_id = products.id").
		Where("products.supplier_id = ?", supplier.ID).
		Select("COALESCE(AVG(reviews.rating), 0) as average, COUNT(reviews.id) as count").
		Scan(&stats)

	response := SupplierDetailResponse{
		ID:            supplier.ID,
		BusinessName:  supplier.BusinessName,
		Email:         supplier.Email,
		Address:       supplier.Address,
		Category:      supplier.Category,
		Region:        supplier.Region,
		PICName:       supplier.PICName,
		Phone:         supplier.Phone,
		Status:        supplier.Status,
		RatingAverage: stats.Average,
		ReviewCount:   stats.Count,
		ProductCount:  productCount,
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   response,
	})
}
