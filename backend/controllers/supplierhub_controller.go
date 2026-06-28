package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"supplierhub-backend/config"
	"supplierhub-backend/models"
	"supplierhub-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const supplierHubServiceFeeRate = 0.03

type supplierHubMaterialInput struct {
	Name        *string  `json:"name"`
	Category    *string  `json:"category"`
	Price       *float64 `json:"price"`
	Stock       *int     `json:"stock"`
	Description *string  `json:"description"`
	ImageURL    *string  `json:"image_url"`
}

type supplierHubOrderInput struct {
	ProductID string `json:"product_id"`
	ItemID    string `json:"item_id"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
}

type supplierStockConfirmationInput struct {
	Action string `json:"action" binding:"required"`
	Note   string `json:"note"`
}

type supplierHubHTTPError struct {
	Status  int
	Message string
}

func (e supplierHubHTTPError) Error() string {
	return e.Message
}

func GetSupplierHubMaterials(c *gin.Context) {
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}
	role, ok := getAuthenticatedRole(c)
	if !ok {
		return
	}

	query := config.DB.Order("created_at DESC")
	switch role {
	case string(models.RoleSupplier):
		query = query.Where("supplier_id = ?", userID)
	case string(models.RoleAdmin):
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya supplier atau admin yang dapat melihat manajemen bahan baku"})
		return
	}

	var products []models.Product
	if err := query.Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar bahan baku"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   products,
	})
}

func CreateSupplierHubMaterial(c *gin.Context) {
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	var supplier models.User
	if err := config.DB.Where("id = ? AND role = ?", userID, models.RoleSupplier).First(&supplier).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya supplier yang dapat menambahkan bahan baku"})
		return
	}

	region := strings.TrimSpace(supplier.Region)
	if region == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lengkapi wilayah supply toko terlebih dahulu sebelum menambahkan produk."})
		return
	}

	input, err := bindSupplierHubMaterialInput(c, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	imageURL := derefString(input.ImageURL)
	if uploadedImageURL, uploaded := saveSupplierHubProductImage(c); uploaded {
		imageURL = uploadedImageURL
	}

	product := models.Product{
		SupplierID:  supplier.ID,
		Name:        strings.TrimSpace(derefString(input.Name)),
		Category:    strings.TrimSpace(derefString(input.Category)),
		Price:       derefFloat(input.Price),
		Stock:       derefInt(input.Stock),
		Description: strings.TrimSpace(derefString(input.Description)),
		Location:    region,
		ImageURL:    strings.TrimSpace(imageURL),
	}

	if err := config.DB.Create(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan bahan baku"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Bahan baku berhasil ditambahkan",
		"data":    product,
	})
}

func UpdateSupplierHubMaterial(c *gin.Context) {
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}
	role, ok := getAuthenticatedRole(c)
	if !ok {
		return
	}

	productID := strings.TrimSpace(c.Param("id"))
	var product models.Product
	if err := config.DB.Where("id = ?", productID).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
		return
	}

	if role == string(models.RoleSupplier) && product.SupplierID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Supplier tidak boleh mengubah produk supplier lain"})
		return
	}
	if role != string(models.RoleSupplier) && role != string(models.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya supplier atau admin yang dapat mengubah bahan baku"})
		return
	}

	var supplier models.User
	if err := config.DB.Where("id = ? AND role = ?", product.SupplierID, models.RoleSupplier).First(&supplier).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Supplier produk tidak ditemukan"})
		return
	}
	region := strings.TrimSpace(supplier.Region)
	if region == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lengkapi wilayah supply toko terlebih dahulu sebelum mengubah produk."})
		return
	}

	input, err := bindSupplierHubMaterialInput(c, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Name != nil {
		product.Name = strings.TrimSpace(*input.Name)
	}
	if input.Category != nil {
		product.Category = strings.TrimSpace(*input.Category)
	}
	if input.Price != nil {
		product.Price = *input.Price
	}
	if input.Stock != nil {
		product.Stock = *input.Stock
	}
	if input.Description != nil {
		product.Description = strings.TrimSpace(*input.Description)
	}
	if input.ImageURL != nil {
		product.ImageURL = strings.TrimSpace(*input.ImageURL)
	}
	if uploadedImageURL, uploaded := saveSupplierHubProductImage(c); uploaded {
		product.ImageURL = uploadedImageURL
	}
	product.Location = region

	if err := config.DB.Save(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengubah bahan baku"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Bahan baku berhasil diperbarui",
		"data":    product,
	})
}

func DeleteSupplierHubMaterial(c *gin.Context) {
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}
	role, ok := getAuthenticatedRole(c)
	if !ok {
		return
	}

	productID := strings.TrimSpace(c.Param("id"))
	query := config.DB.Where("id = ?", productID)
	if role == string(models.RoleSupplier) {
		query = query.Where("supplier_id = ?", userID)
	} else if role != string(models.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya supplier atau admin yang dapat menghapus bahan baku"})
		return
	}

	result := query.Delete(&models.Product{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus bahan baku"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan atau bukan milik supplier ini"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Bahan baku berhasil dihapus",
	})
}

func CreateSupplierHubOrder(c *gin.Context) {
	umkmID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	var input supplierHubOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id dan quantity wajib diisi"})
		return
	}

	productID := strings.TrimSpace(input.ProductID)
	if productID == "" {
		productID = strings.TrimSpace(input.ItemID)
	}
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id wajib diisi"})
		return
	}
	if input.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quantity harus lebih dari 0"})
		return
	}

	var product models.Product
	if err := config.DB.Where("id = ?", productID).First(&product).Error; err != nil {
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

	subtotal := product.Price * float64(input.Quantity)
	serviceFee := subtotal * supplierHubServiceFeeRate
	grandTotal := subtotal + serviceFee

	order := models.Order{
		UmkmID:         umkmID,
		SupplierID:     product.SupplierID,
		ProductID:      product.ID,
		Quantity:       input.Quantity,
		TotalBasePrice: subtotal,
		SystemFee:      serviceFee,
		GrandTotal:     grandTotal,
		Status:         models.OrderPendingSupplierConfirmation,
		StockDeducted:  false,
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		return services.CreateFinanceLog(tx, order, nil, "order_created", "Order bahan dibuat dan menunggu konfirmasi supplier")
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat order bahan"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Order bahan berhasil dibuat dan menunggu konfirmasi supplier",
		"data":    order,
	})
}

func ConfirmSupplierHubStock(c *gin.Context) {
	supplierID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	orderID := strings.TrimSpace(c.Param("order_id"))
	var input supplierStockConfirmationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Action confirm atau reject wajib diisi"})
		return
	}

	action := strings.ToLower(strings.TrimSpace(input.Action))
	note := strings.TrimSpace(input.Note)
	if action != "confirm" && action != "reject" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Action harus bernilai confirm atau reject"})
		return
	}

	responseStatus := http.StatusOK
	responseMessage := ""
	var responseOrder models.Order
	var responsePayment *models.Payment
	var gatewayPayload interface{}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var order models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", orderID).
			First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return supplierHubHTTPError{Status: http.StatusNotFound, Message: "Order tidak ditemukan"}
			}
			return err
		}

		if order.SupplierID != supplierID {
			return supplierHubHTTPError{Status: http.StatusForbidden, Message: "Supplier tidak boleh mengonfirmasi order supplier lain"}
		}

		if action == "reject" {
			order.Status = models.OrderRejectedBySupplier
			if err := tx.Model(&order).Update("status", order.Status).Error; err != nil {
				return err
			}
			if err := services.CreateFinanceLog(tx, order, nil, "supplier_rejected", note); err != nil {
				return err
			}
			responseOrder = order
			responseMessage = "Order ditolak supplier"
			return nil
		}

		if order.StockDeducted {
			return supplierHubHTTPError{Status: http.StatusBadRequest, Message: "Stok untuk order ini sudah pernah dikurangi."}
		}

		var product models.Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", order.ProductID).
			First(&product).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return supplierHubHTTPError{Status: http.StatusNotFound, Message: "Produk tidak ditemukan"}
			}
			return err
		}

		if product.Stock < order.Quantity {
			order.Status = models.OrderStockUnavailable
			if err := tx.Model(&order).Update("status", order.Status).Error; err != nil {
				return err
			}
			if err := services.CreateFinanceLog(tx, order, nil, "stock_not_enough", "Stok tidak mencukupi. "+note); err != nil {
				return err
			}
			responseStatus = http.StatusBadRequest
			responseMessage = "Stok tidak mencukupi."
			responseOrder = order
			return nil
		}

		product.Stock -= order.Quantity
		if err := tx.Save(&product).Error; err != nil {
			return err
		}

		order.StockDeducted = true
		order.Status = models.OrderSupplierConfirmed
		if err := tx.Model(&order).Updates(map[string]interface{}{
			"stock_deducted": true,
			"status":         order.Status,
		}).Error; err != nil {
			return err
		}
		if err := services.CreateFinanceLog(tx, order, nil, "supplier_confirmed", note); err != nil {
			return err
		}

		payment := models.Payment{
			OrderID:         order.ID,
			UserID:          order.UmkmID,
			Amount:          order.GrandTotal,
			SupplierFee:     order.SystemFee,
			Status:          models.PaymentPending,
			PaymentMethod:   "smartbank",
			ExternalOrderID: order.ID,
		}
		if err := tx.Create(&payment).Error; err != nil {
			return err
		}

		gatewayResult, gatewayErr := services.CreateSmartBankPaymentRequest(order, payment)
		gatewayPayload = gatewayResult
		if gatewayErr != nil {
			responseStatus = http.StatusBadGateway
			responseMessage = "Stok berhasil dikonfirmasi, tetapi payment request gagal dibuat"
			responseText := gatewayResult.RawResponse
			if responseText == "" {
				responseText = gatewayErr.Error()
			}

			payment.Status = models.PaymentFailed
			payment.GatewayStatus = "request_failed"
			payment.GatewayResponse = responseText
			if err := tx.Model(&payment).Updates(map[string]interface{}{
				"status":           payment.Status,
				"gateway_status":   payment.GatewayStatus,
				"gateway_response": payment.GatewayResponse,
			}).Error; err != nil {
				return err
			}

			order.Status = models.OrderPaymentRequestFailed
			if err := tx.Model(&order).Update("status", order.Status).Error; err != nil {
				return err
			}
			if err := services.CreateFinanceLog(tx, order, &payment, "payment_request_failed", gatewayErr.Error()); err != nil {
				return err
			}
			responseOrder = order
			responsePayment = &payment
			return nil
		}

		payment.PaymentReference = gatewayResult.PaymentReference
		payment.VirtualAccount = gatewayResult.VirtualAccount
		payment.GatewayStatus = gatewayResult.Status
		payment.GatewayResponse = gatewayResult.RawResponse
		if err := tx.Model(&payment).Updates(map[string]interface{}{
			"payment_reference": payment.PaymentReference,
			"virtual_account":   payment.VirtualAccount,
			"gateway_status":    payment.GatewayStatus,
			"gateway_response":  payment.GatewayResponse,
		}).Error; err != nil {
			return err
		}

		order.Status = models.OrderPaymentPending
		if err := tx.Model(&order).Update("status", order.Status).Error; err != nil {
			return err
		}
		if err := services.CreateFinanceLog(tx, order, &payment, "payment_request_created", "Payment request berhasil dibuat melalui API Gateway"); err != nil {
			return err
		}

		responseMessage = "Stok dikonfirmasi dan payment request berhasil dibuat"
		responseOrder = order
		responsePayment = &payment
		return nil
	})

	if err != nil {
		var httpErr supplierHubHTTPError
		if errors.As(err, &httpErr) {
			c.JSON(httpErr.Status, gin.H{"error": httpErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses konfirmasi stok"})
		return
	}

	payload := gin.H{
		"status":  "success",
		"message": responseMessage,
		"data": gin.H{
			"order": responseOrder,
		},
	}
	if responsePayment != nil {
		payload["data"].(gin.H)["payment"] = responsePayment
	}
	if gatewayPayload != nil {
		payload["data"].(gin.H)["gateway_response"] = gatewayPayload
	}
	if responseStatus >= http.StatusBadRequest {
		payload["status"] = "warning"
	}

	c.JSON(responseStatus, payload)
}

func GetSupplierHubServiceFeeSummary(c *gin.Context) {
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}
	role, ok := getAuthenticatedRole(c)
	if !ok {
		return
	}
	if role != string(models.RoleSupplier) && role != string(models.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya supplier atau admin yang dapat melihat biaya layanan supplier"})
		return
	}

	orderQuery := func() *gorm.DB {
		query := config.DB.Model(&models.Order{})
		if role == string(models.RoleSupplier) {
			query = query.Where("supplier_id = ?", userID)
		}
		return query
	}
	paymentQuery := func() *gorm.DB {
		query := config.DB.Model(&models.Payment{}).Joins("JOIN orders ON orders.id = payments.order_id")
		if role == string(models.RoleSupplier) {
			query = query.Where("orders.supplier_id = ?", userID)
		}
		return query
	}

	var totalTransactions int64
	var totalSubtotal float64
	var totalFee float64
	var totalGrandTotal float64
	var successPayments int64
	var pendingPayments int64
	var failedPayments int64
	var logs []models.FinanceLog

	if err := orderQuery().Count(&totalTransactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung total transaksi"})
		return
	}
	if err := orderQuery().Select("COALESCE(SUM(total_base_price), 0)").Scan(&totalSubtotal).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung subtotal"})
		return
	}
	if err := orderQuery().Select("COALESCE(SUM(system_fee), 0)").Scan(&totalFee).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung fee supplier"})
		return
	}
	if err := orderQuery().Select("COALESCE(SUM(grand_total), 0)").Scan(&totalGrandTotal).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung grand total"})
		return
	}
	if err := paymentQuery().Where("payments.status = ?", models.PaymentSuccess).Count(&successPayments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung pembayaran sukses"})
		return
	}
	if err := paymentQuery().Where("payments.status = ?", models.PaymentPending).Count(&pendingPayments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung pembayaran pending"})
		return
	}
	if err := paymentQuery().Where("payments.status = ?", models.PaymentFailed).Count(&failedPayments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung pembayaran gagal"})
		return
	}

	logQuery := config.DB.Order("created_at DESC").Limit(20)
	if role == string(models.RoleSupplier) {
		logQuery = logQuery.Where("supplier_id = ?", userID)
	}
	_ = logQuery.Find(&logs).Error

	c.JSON(http.StatusOK, gin.H{
		"status":                 "success",
		"fee_rate":               supplierHubServiceFeeRate,
		"total_transaksi":        totalTransactions,
		"total_subtotal":         totalSubtotal,
		"total_fee":              totalFee,
		"total_grand_total":      totalGrandTotal,
		"pembayaran_sukses":      successPayments,
		"pembayaran_pending":     pendingPayments,
		"pembayaran_gagal":       failedPayments,
		"latest_finance_logs":    logs,
		"scope":                  role,
		"service_fee_percentage": 3,
	})
}

func bindSupplierHubMaterialInput(c *gin.Context, requireAll bool) (supplierHubMaterialInput, error) {
	if strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		var input supplierHubMaterialInput
		if err := c.ShouldBindJSON(&input); err != nil {
			return input, fmt.Errorf("Format JSON tidak valid")
		}
		return validateSupplierHubMaterialInput(input, requireAll)
	}

	input := supplierHubMaterialInput{}
	if raw, exists := c.GetPostForm("name"); exists {
		value := strings.TrimSpace(raw)
		input.Name = &value
	}
	if raw, exists := c.GetPostForm("category"); exists {
		value := strings.TrimSpace(raw)
		input.Category = &value
	}
	if raw, exists := c.GetPostForm("price"); exists {
		value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return input, fmt.Errorf("Harga produk tidak valid")
		}
		input.Price = &value
	}
	if raw, exists := c.GetPostForm("stock"); exists {
		value, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return input, fmt.Errorf("Stok produk tidak valid")
		}
		input.Stock = &value
	}
	if raw, exists := c.GetPostForm("description"); exists {
		value := strings.TrimSpace(raw)
		input.Description = &value
	}
	if raw, exists := c.GetPostForm("image_url"); exists {
		value := strings.TrimSpace(raw)
		input.ImageURL = &value
	}

	return validateSupplierHubMaterialInput(input, requireAll)
}

func validateSupplierHubMaterialInput(input supplierHubMaterialInput, requireAll bool) (supplierHubMaterialInput, error) {
	if requireAll {
		if input.Name == nil || strings.TrimSpace(*input.Name) == "" ||
			input.Category == nil || strings.TrimSpace(*input.Category) == "" ||
			input.Price == nil || input.Stock == nil {
			return input, fmt.Errorf("name, category, price, dan stock wajib diisi")
		}
	}

	if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
		return input, fmt.Errorf("Nama produk tidak boleh kosong")
	}
	if input.Category != nil && strings.TrimSpace(*input.Category) == "" {
		return input, fmt.Errorf("Kategori produk tidak boleh kosong")
	}
	if input.Price != nil && *input.Price < 0 {
		return input, fmt.Errorf("Harga produk tidak boleh negatif")
	}
	if input.Stock != nil && *input.Stock < 0 {
		return input, fmt.Errorf("Stok tidak boleh negatif")
	}

	return input, nil
}

func saveSupplierHubProductImage(c *gin.Context) (string, bool) {
	file, err := c.FormFile("image")
	if err != nil {
		return "", false
	}

	filename := uuid.New().String() + filepath.Ext(file.Filename)
	uploadPath := "uploads/" + filename
	if err := c.SaveUploadedFile(file, uploadPath); err != nil {
		return "", false
	}

	return config.PublicURL(uploadPath), true
}

func getAuthenticatedRole(c *gin.Context) (string, bool) {
	role, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Role pengguna tidak ditemukan"})
		return "", false
	}
	roleText, ok := role.(string)
	if !ok || strings.TrimSpace(roleText) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Role pengguna tidak valid"})
		return "", false
	}
	return roleText, true
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefFloat(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func derefInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
