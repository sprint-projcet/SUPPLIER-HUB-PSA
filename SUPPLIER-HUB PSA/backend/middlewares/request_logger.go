package middlewares

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"supplierhub-backend/config"
	"supplierhub-backend/models"

	"github.com/gin-gonic/gin"
)

const maxLoggedBodySize = 8192

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !shouldLogRequest(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()
		requestBody := readAndRestoreBody(c)
		responseBuffer := &bytes.Buffer{}
		c.Writer = bodyLogWriter{ResponseWriter: c.Writer, body: responseBuffer}

		c.Next()

		userID, _ := c.Get("user_id")
		role, _ := c.Get("user_role")
		responseMessage := truncate(responseBuffer.String(), maxLoggedBodySize)

		logEntry := models.RequestLog{
			UserID:          stringContextValue(userID),
			Role:            stringContextValue(role),
			Method:          c.Request.Method,
			Path:            c.Request.URL.Path,
			StatusCode:      c.Writer.Status(),
			IPAddress:       c.ClientIP(),
			UserAgent:       c.Request.UserAgent(),
			RequestBody:     requestBody,
			ResponseMessage: responseMessage,
			LatencyMS:       time.Since(start).Milliseconds(),
		}

		if config.DB != nil {
			_ = config.DB.Create(&logEntry).Error
		}
	}
}

func shouldLogRequest(path string) bool {
	prefixes := []string{
		"/supplierhub/order_bahan",
		"/supplierhub/konfirmasi_stok",
		"/supplierhub/payment",
		"/supplierhub/manajemen_bahan_baku",
		"/supplierhub/biaya_layanan_supplier",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func readAndRestoreBody(c *gin.Context) string {
	if c.Request.Body == nil || c.Request.Body == http.NoBody {
		return ""
	}

	contentType := strings.ToLower(c.GetHeader("Content-Type"))
	if strings.Contains(contentType, "multipart/form-data") {
		return "[multipart body omitted]"
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return sanitizeBody(bodyBytes, contentType)
}

func sanitizeBody(bodyBytes []byte, contentType string) string {
	if len(bodyBytes) == 0 {
		return ""
	}

	if strings.Contains(contentType, "application/json") {
		var payload interface{}
		if err := json.Unmarshal(bodyBytes, &payload); err == nil {
			redactSensitiveFields(payload)
			if sanitized, err := json.Marshal(payload); err == nil {
				return truncate(string(sanitized), maxLoggedBodySize)
			}
		}
	}

	body := string(bodyBytes)
	lowerBody := strings.ToLower(body)
	if strings.Contains(lowerBody, "password") || strings.Contains(lowerBody, "token") || strings.Contains(lowerBody, "authorization") {
		return "[sensitive body redacted]"
	}

	return truncate(body, maxLoggedBodySize)
}

func redactSensitiveFields(value interface{}) {
	switch data := value.(type) {
	case map[string]interface{}:
		for key, fieldValue := range data {
			lowerKey := strings.ToLower(key)
			if strings.Contains(lowerKey, "password") || strings.Contains(lowerKey, "token") || strings.Contains(lowerKey, "authorization") {
				data[key] = "[redacted]"
				continue
			}
			redactSensitiveFields(fieldValue)
		}
	case []interface{}:
		for _, item := range data {
			redactSensitiveFields(item)
		}
	}
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "...[truncated]"
}

func stringContextValue(value interface{}) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
