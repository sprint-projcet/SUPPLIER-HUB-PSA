package services

import (
	"supplierhub-backend/models"

	"gorm.io/gorm"
)

func CreateFinanceLog(tx *gorm.DB, order models.Order, payment *models.Payment, transactionType, note string) error {
	var paymentID *string
	paymentStatus := ""
	if payment != nil {
		paymentID = &payment.ID
		paymentStatus = string(payment.Status)
	}

	logEntry := models.FinanceLog{
		OrderID:         order.ID,
		PaymentID:       paymentID,
		UmkmID:          order.UmkmID,
		SupplierID:      order.SupplierID,
		ProductID:       order.ProductID,
		Subtotal:        order.TotalBasePrice,
		SupplierFee:     order.SystemFee,
		GrandTotal:      order.GrandTotal,
		PaymentStatus:   paymentStatus,
		OrderStatus:     string(order.Status),
		TransactionType: transactionType,
		Note:            note,
	}

	return tx.Create(&logEntry).Error
}
