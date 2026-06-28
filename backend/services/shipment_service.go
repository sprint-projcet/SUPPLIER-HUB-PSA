package services

import (
	"supplierhub-backend/models"

	"gorm.io/gorm"
)

func CreateShipmentLog(tx *gorm.DB, order models.Order, result LogistiKitaShipmentResult, err error) error {
	status := result.Status
	if status == "" {
		status = "failed"
	}

	shipmentLog := models.ShipmentLog{
		OrderID:         order.ID,
		SupplierID:      order.SupplierID,
		UmkmID:          order.UmkmID,
		ProductID:       order.ProductID,
		ShipmentID:      result.ShipmentID,
		Status:          status,
		GatewayResponse: result.RawResponse,
	}

	if err != nil {
		shipmentLog.Status = "failed"
		shipmentLog.ErrorMessage = err.Error()
	}

	return tx.Create(&shipmentLog).Error
}
