package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"supplierhub-backend/models"
)

const defaultGatewayTimeout = 10 * time.Second

type SmartBankPaymentResult struct {
	Success          bool
	PaymentReference string
	VirtualAccount   string
	Status           string
	RawResponse      string
}

type LogistiKitaShipmentResult struct {
	Success     bool
	ShipmentID  string
	Status      string
	RawResponse string
}

func CreateSmartBankPaymentRequest(order models.Order, payment models.Payment) (SmartBankPaymentResult, error) {
	payload := map[string]interface{}{
		"external_order_id": order.ID,
		"user_id":           order.UmkmID,
		"supplier_id":       order.SupplierID,
		"amount":            payment.Amount,
		"subtotal":          order.TotalBasePrice,
		"service_fee":       order.SystemFee,
		"callback_url":      supplierHubCallbackURL(),
	}

	response, statusCode, err := postJSON(gatewayURL()+smartBankPaymentPath(), payload)
	if err != nil {
		return SmartBankPaymentResult{}, err
	}

	result := SmartBankPaymentResult{
		Success:          boolValue(response["success"]),
		PaymentReference: stringValue(response["payment_reference"]),
		VirtualAccount:   stringValue(response["virtual_account"]),
		Status:           stringValue(response["status"]),
		RawResponse:      JSONString(response),
	}

	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return result, fmt.Errorf("API Gateway mengembalikan status %d", statusCode)
	}
	if !result.Success {
		return result, fmt.Errorf("SmartBank menolak payment request")
	}
	if result.Status == "" {
		result.Status = "pending"
	}

	return result, nil
}

func CreateLogistiKitaShipment(order models.Order) (LogistiKitaShipmentResult, error) {
	payload := map[string]interface{}{
		"external_order_id":   order.ID,
		"supplier_id":         order.SupplierID,
		"umkm_id":             order.UmkmID,
		"product_id":          order.ProductID,
		"quantity":            order.Quantity,
		"origin_region":       order.Supplier.Region,
		"destination_address": order.Umkm.Address,
		"status":              "waiting_pickup",
	}

	response, statusCode, err := postJSON(gatewayURL()+logistiKitaShipmentPath(), payload)
	if err != nil {
		return LogistiKitaShipmentResult{}, err
	}

	result := LogistiKitaShipmentResult{
		Success:     boolValue(response["success"]),
		ShipmentID:  stringValue(response["shipment_id"]),
		Status:      stringValue(response["status"]),
		RawResponse: JSONString(response),
	}

	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return result, fmt.Errorf("API Gateway mengembalikan status %d", statusCode)
	}
	if !result.Success {
		return result, fmt.Errorf("LogistiKita menolak shipment request")
	}
	if result.Status == "" {
		result.Status = "waiting_pickup"
	}

	return result, nil
}

func JSONString(value interface{}) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func postJSON(url string, payload interface{}) (map[string]interface{}, int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: defaultGatewayTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	var response map[string]interface{}
	if len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, resp.StatusCode, err
		}
	} else {
		response = map[string]interface{}{}
	}

	return response, resp.StatusCode, nil
}

func gatewayURL() string {
	url := strings.TrimRight(strings.TrimSpace(os.Getenv("API_GATEWAY_URL")), "/")
	if url == "" {
		url = "http://localhost:9000"
	}
	return url
}

func smartBankPaymentPath() string {
	path := strings.TrimSpace(os.Getenv("SMARTBANK_PAYMENT_PATH"))
	if path == "" {
		path = "/smartbank/payment/request"
	}
	return "/" + strings.TrimLeft(path, "/")
}

func logistiKitaShipmentPath() string {
	path := strings.TrimSpace(os.Getenv("LOGISTIKITA_SHIPMENT_PATH"))
	if path == "" {
		path = "/logistikita/shipment/create"
	}
	return "/" + strings.TrimLeft(path, "/")
}

func supplierHubCallbackURL() string {
	url := strings.TrimSpace(os.Getenv("SUPPLIERHUB_CALLBACK_URL"))
	if url == "" {
		url = "http://localhost:8080/supplierhub/payment/callback"
	}
	return url
}

func stringValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}

func boolValue(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true") || strings.EqualFold(v, "success")
	default:
		return false
	}
}
