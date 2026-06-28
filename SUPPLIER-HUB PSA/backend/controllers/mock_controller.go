package controllers

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type mockPaymentRequestInput struct {
	ExternalOrderID string  `json:"external_order_id"`
	UserID          string  `json:"user_id"`
	SupplierID      string  `json:"supplier_id"`
	Amount          float64 `json:"amount"`
	Subtotal        float64 `json:"subtotal"`
	ServiceFee      float64 `json:"service_fee"`
	CallbackURL     string  `json:"callback_url"`
}

type mockCallbackSimulationInput struct {
	PaymentReference string `json:"payment_reference"`
	ExternalOrderID  string `json:"external_order_id"`
	OrderID          string `json:"order_id"`
	Status           string `json:"status"`
	CallbackURL      string `json:"callback_url"`
}

func MockSmartBankPaymentRequest(c *gin.Context) {
	var input mockPaymentRequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Payload payment request tidak valid"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"payment_reference": "SB-" + randomNumericString(8),
		"virtual_account":   "8808" + randomNumericString(8),
		"status":            "pending",
		"external_order_id": input.ExternalOrderID,
	})
}

func MockSmartBankSimulateCallback(c *gin.Context) {
	var input mockCallbackSimulationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Payload simulasi callback tidak valid"})
		return
	}

	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status == "" {
		status = "success"
	}
	if status != "success" && status != "failed" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Status simulasi harus success atau failed"})
		return
	}

	callbackURL := strings.TrimSpace(input.CallbackURL)
	if callbackURL == "" {
		callbackURL = strings.TrimSpace(os.Getenv("SUPPLIERHUB_CALLBACK_URL"))
	}
	if callbackURL == "" {
		callbackURL = "http://localhost:8080/supplierhub/payment/callback"
	}

	paidAt := interface{}(nil)
	if status == "success" {
		paidAt = time.Now().Format(time.RFC3339)
	}

	payload := gin.H{
		"payment_reference": input.PaymentReference,
		"external_order_id": input.ExternalOrderID,
		"order_id":          input.OrderID,
		"status":            status,
		"paid_at":           paidAt,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Gagal menyiapkan payload callback"})
		return
	}

	req, err := http.NewRequest(http.MethodPost, callbackURL, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Gagal membuat request callback"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if token := strings.TrimSpace(os.Getenv("SUPPLIERHUB_CALLBACK_API_KEY")); token != "" {
		req.Header.Set("X-Internal-Token", token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": err.Error()})
		return
	}
	defer resp.Body.Close()

	var callbackResponse map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&callbackResponse)

	c.JSON(http.StatusOK, gin.H{
		"success":           resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices,
		"callback_url":      callbackURL,
		"callback_status":   resp.StatusCode,
		"callback_response": callbackResponse,
	})
}

func MockLogistiKitaShipmentCreate(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Payload shipment tidak valid"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"shipment_id": "LK-" + randomNumericString(8),
		"status":      "waiting_pickup",
	})
}

func randomNumericString(length int) string {
	rand.Seed(time.Now().UnixNano())
	digits := make([]byte, length)
	for i := range digits {
		digits[i] = byte('0' + rand.Intn(10))
	}
	return string(digits)
}
