package controllers

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"supplierhub-backend/config"
	"supplierhub-backend/models"
	"supplierhub-backend/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type supplierHubPaymentCallbackInput struct {
	PaymentReference string `json:"payment_reference"`
	ExternalOrderID  string `json:"external_order_id"`
	OrderID          string `json:"order_id"`
	PaymentStatus    string `json:"payment_status"`
	Status           string `json:"status"`
	PaidAt           string `json:"paid_at"`
}

func HandleSupplierHubPaymentCallback(c *gin.Context) {
	if !validateSupplierHubCallbackKey(c) {
		return
	}

	var input supplierHubPaymentCallbackInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload callback pembayaran tidak valid"})
		return
	}

	callbackStatus := strings.ToLower(strings.TrimSpace(input.Status))
	if callbackStatus == "" {
		callbackStatus = strings.ToLower(strings.TrimSpace(input.PaymentStatus))
	}
	if callbackStatus == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status callback wajib diisi"})
		return
	}

	isSuccess := callbackStatus == "success" || callbackStatus == "paid"
	isFailed := callbackStatus == "failed" || callbackStatus == "fail" || callbackStatus == "cancelled" || callbackStatus == "canceled"
	if !isSuccess && !isFailed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status callback harus success atau failed"})
		return
	}

	var payment models.Payment
	var order models.Order
	var shipmentLog *models.ShipmentLog
	var shipmentWarning string

	callbackPayload := services.JSONString(input)
	paidAt := parseCallbackPaidAt(input.PaidAt, isSuccess)

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Order.Product").Preload("Order.Supplier").Preload("Order.Umkm")
		switch {
		case strings.TrimSpace(input.PaymentReference) != "":
			query = query.Where("payment_reference = ?", strings.TrimSpace(input.PaymentReference))
		case strings.TrimSpace(input.ExternalOrderID) != "":
			query = query.Where("external_order_id = ? OR order_id = ?", strings.TrimSpace(input.ExternalOrderID), strings.TrimSpace(input.ExternalOrderID))
		case strings.TrimSpace(input.OrderID) != "":
			query = query.Where("order_id = ?", strings.TrimSpace(input.OrderID))
		default:
			return supplierHubHTTPError{Status: http.StatusBadRequest, Message: "payment_reference atau external_order_id wajib diisi"}
		}

		if err := query.First(&payment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return supplierHubHTTPError{Status: http.StatusNotFound, Message: "Payment tidak ditemukan"}
			}
			return err
		}

		order = payment.Order
		if order.ID == "" {
			return supplierHubHTTPError{Status: http.StatusNotFound, Message: "Order payment tidak ditemukan"}
		}

		if isFailed {
			payment.Status = models.PaymentFailed
			payment.CallbackStatus = "failed"
			payment.GatewayStatus = "failed"
			payment.GatewayResponse = callbackPayload
			order.Status = models.OrderPaymentFailed

			if err := tx.Model(&payment).Updates(map[string]interface{}{
				"status":           payment.Status,
				"callback_status":  payment.CallbackStatus,
				"gateway_status":   payment.GatewayStatus,
				"gateway_response": payment.GatewayResponse,
			}).Error; err != nil {
				return err
			}
			if err := tx.Model(&order).Update("status", order.Status).Error; err != nil {
				return err
			}
			return services.CreateFinanceLog(tx, order, &payment, "payment_failed", "Callback SmartBank menyatakan pembayaran gagal")
		}

		payment.Status = models.PaymentSuccess
		payment.CallbackStatus = "success"
		payment.GatewayStatus = "success"
		payment.GatewayResponse = callbackPayload
		payment.PaidAt = paidAt
		order.Status = models.OrderPaid

		if err := tx.Model(&payment).Updates(map[string]interface{}{
			"status":           payment.Status,
			"callback_status":  payment.CallbackStatus,
			"gateway_status":   payment.GatewayStatus,
			"gateway_response": payment.GatewayResponse,
			"paid_at":          payment.PaidAt,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&order).Update("status", order.Status).Error; err != nil {
			return err
		}
		if err := services.CreateFinanceLog(tx, order, &payment, "payment_success", "Callback SmartBank sukses"); err != nil {
			return err
		}

		shipmentResult, shipmentErr := services.CreateLogistiKitaShipment(order)
		if shipmentErr != nil {
			shipmentWarning = shipmentErr.Error()
			if err := services.CreateShipmentLog(tx, order, shipmentResult, shipmentErr); err != nil {
				return err
			}
			return nil
		}

		order.Status = models.OrderShipmentCreated
		if err := tx.Model(&order).Update("status", order.Status).Error; err != nil {
			return err
		}
		if err := services.CreateFinanceLog(tx, order, &payment, "shipment_created", "Shipment berhasil dibuat di LogistiKita"); err != nil {
			return err
		}
		if err := services.CreateShipmentLog(tx, order, shipmentResult, nil); err != nil {
			return err
		}

		var savedShipment models.ShipmentLog
		if err := tx.Where("order_id = ?", order.ID).Order("created_at DESC").First(&savedShipment).Error; err == nil {
			shipmentLog = &savedShipment
		}
		return nil
	})

	if err != nil {
		var httpErr supplierHubHTTPError
		if errors.As(err, &httpErr) {
			c.JSON(httpErr.Status, gin.H{"error": httpErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses callback pembayaran"})
		return
	}

	response := gin.H{
		"status":  "success",
		"message": "Callback pembayaran berhasil diproses",
		"data": gin.H{
			"payment": payment,
			"order":   order,
		},
	}
	if shipmentLog != nil {
		response["data"].(gin.H)["shipment"] = shipmentLog
	}
	if shipmentWarning != "" {
		response["status"] = "warning"
		response["message"] = "Pembayaran sukses, tetapi integrasi LogistiKita gagal"
		response["data"].(gin.H)["shipment_error"] = shipmentWarning
	}

	c.JSON(http.StatusOK, response)
}

func validateSupplierHubCallbackKey(c *gin.Context) bool {
	expected := strings.TrimSpace(os.Getenv("SUPPLIERHUB_CALLBACK_API_KEY"))
	if expected == "" {
		return true
	}

	provided := strings.TrimSpace(c.GetHeader("X-Internal-Token"))
	if provided == "" {
		provided = strings.TrimSpace(c.GetHeader("X-Callback-Token"))
	}
	if provided != expected {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Callback token tidak valid"})
		return false
	}
	return true
}

func parseCallbackPaidAt(raw string, success bool) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if !success {
			return nil
		}
		now := time.Now()
		return &now
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return &parsed
		}
	}

	now := time.Now()
	return &now
}
