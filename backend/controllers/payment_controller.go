package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"supplierhub-backend/config"
	"supplierhub-backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const supplierHubFeeRate = 0.03

type paymentRequestInput struct {
	OrderID       string `json:"order_id" binding:"required"`
	PaymentMethod string `json:"payment_method"`
}

type smartBankCallbackInput struct {
	OrderID       string `json:"order_id" binding:"required"`
	PaymentStatus string `json:"payment_status"`
	Status        string `json:"status"`
}

func CreatePaymentRequest(c *gin.Context) {
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	var input paymentRequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID wajib diisi"})
		return
	}

	var order models.Order
	if err := config.DB.Preload("Product").
		Where("id = ? AND umkm_id = ?", input.OrderID, userID).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pesanan tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil pesanan"})
		return
	}

	if order.Status == models.OrderCancelled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pesanan yang dibatalkan tidak bisa dibayar"})
		return
	}
	if order.Status == models.OrderPaid || order.Status == models.OrderProcessing ||
		order.Status == models.OrderShipped || order.Status == models.OrderCompleted {
		c.JSON(http.StatusConflict, gin.H{"error": "Pesanan sudah diproses pembayaran"})
		return
	}
	if !order.StockDeducted && order.Product.Stock < order.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stok produk tidak mencukupi untuk pembayaran"})
		return
	}

	baseTotal := order.TotalBasePrice
	if baseTotal <= 0 {
		baseTotal = order.Product.Price * float64(order.Quantity)
	}
	systemFee := baseTotal * supplierHubFeeRate
	grandTotal := baseTotal + systemFee

	payment := models.Payment{
		OrderID:       order.ID,
		UserID:        userID,
		Amount:        grandTotal,
		SupplierFee:   systemFee,
		Status:        models.PaymentPending,
		PaymentMethod: strings.TrimSpace(input.PaymentMethod),
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&order).Updates(map[string]interface{}{
			"total_base_price": baseTotal,
			"system_fee":       systemFee,
			"grand_total":      grandTotal,
		}).Error; err != nil {
			return err
		}
		return tx.Create(&payment).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyiapkan pembayaran"})
		return
	}

	gatewayPayload := gin.H{
		"target_service": "SmartBank",
		"target_path":    "/mock/smartbank/pay",
		"order_id":       order.ID,
		"amount":         grandTotal,
		"supplier_fee":   systemFee,
		"callback_url":   paymentCallbackURL(),
		"requested_at":   time.Now().Format(time.RFC3339),
	}
	if payment.PaymentMethod != "" {
		gatewayPayload["payment_method"] = payment.PaymentMethod
	}

	gatewayResponse, statusCode, err := forwardPaymentToSmartBank(gatewayPayload)
	if err != nil {
		_ = config.DB.Model(&payment).Updates(map[string]interface{}{
			"status":           models.PaymentFailed,
			"gateway_status":   "forward_failed",
			"gateway_response": err.Error(),
		}).Error
		c.JSON(http.StatusBadGateway, gin.H{"error": "Gagal meneruskan pembayaran ke SmartBank"})
		return
	}

	paymentUpdate := map[string]interface{}{
		"gateway_status":   gatewayResponse["status"],
		"gateway_response": mustJSON(gatewayResponse),
	}
	if va, ok := gatewayResponse["virtual_account"].(string); ok {
		paymentUpdate["virtual_account"] = va
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		paymentUpdate["status"] = models.PaymentFailed
		_ = config.DB.Model(&payment).Updates(paymentUpdate).Error
		c.JSON(http.StatusBadGateway, gin.H{"error": "SmartBank menolak request pembayaran", "data": gatewayResponse})
		return
	}

	_ = config.DB.Model(&payment).Updates(paymentUpdate).Error
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Order Confirmed. Payment requested to SmartBank.",
		"data": gin.H{
			"order_id":         order.ID,
			"grand_total":      grandTotal,
			"fee_deducted":     systemFee,
			"payment":          payment,
			"gateway_response": gatewayResponse,
		},
	})
}

func HandleSmartBankCallback(c *gin.Context) {
	HandleSupplierHubPaymentCallback(c)
}

func forwardPaymentToSmartBank(payload gin.H) (map[string]interface{}, int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequest(http.MethodPost, smartBankGatewayURL(), bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Target-Service", "SmartBank")
	req.Header.Set("X-Target-Path", "/mock/smartbank/pay")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	var gatewayResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&gatewayResponse); err != nil {
		return nil, resp.StatusCode, err
	}
	return gatewayResponse, resp.StatusCode, nil
}

func smartBankGatewayURL() string {
	url := strings.TrimSpace(os.Getenv("SMARTBANK_GATEWAY_URL"))
	if url == "" {
		url = "http://localhost:3000/gateway/forward"
	}
	return url
}

func paymentCallbackURL() string {
	url := strings.TrimSpace(os.Getenv("SUPPLIERHUB_CALLBACK_URL"))
	if url == "" {
		url = strings.TrimSpace(os.Getenv("PAYMENT_CALLBACK_URL"))
	}
	if url == "" {
		url = config.AppBaseURL() + "/supplierhub/payment/callback"
	}
	return url
}

func mustJSON(value interface{}) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}
