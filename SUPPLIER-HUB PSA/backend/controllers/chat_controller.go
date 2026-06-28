package controllers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"supplierhub-backend/config"
	"supplierhub-backend/models"
	"supplierhub-backend/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type createChatConversationInput struct {
	SupplierID string `json:"supplier_id" binding:"required"`
}

type sendChatMessageInput struct {
	Message string `json:"message" binding:"required"`
}

func getChatActor(c *gin.Context) (string, string, bool) {
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		return "", "", false
	}

	roleValue, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Role pengguna tidak ditemukan"})
		return "", "", false
	}

	role, ok := roleValue.(string)
	if !ok || role == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Role pengguna tidak valid"})
		return "", "", false
	}

	return userID, role, true
}

func GetChatConversations(c *gin.Context) {
	userID, role, ok := getChatActor(c)
	if !ok {
		return
	}

	var conversations []models.ChatConversation
	query := config.DB.Preload("Umkm").Preload("Supplier")
	if role == string(models.RoleSupplier) {
		query = query.Where("supplier_id = ?", userID)
	} else {
		query = query.Where("umkm_id = ?", userID)
	}

	if err := query.Order("updated_at DESC").Find(&conversations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar chat"})
		return
	}

	response := make([]gin.H, 0, len(conversations))
	for _, conversation := range conversations {
		response = append(response, chatConversationPayload(conversation, userID, role))
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   response,
	})
}

func CreateChatConversation(c *gin.Context) {
	userID, role, ok := getChatActor(c)
	if !ok {
		return
	}

	if role != string(models.RoleUser) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya UMKM yang dapat memulai chat dari profil supplier"})
		return
	}

	var input createChatConversationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	supplierID := strings.TrimSpace(input.SupplierID)
	var supplier models.User
	if err := config.DB.Where("id = ? AND role = ?", supplierID, models.RoleSupplier).First(&supplier).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Supplier tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi supplier"})
		return
	}

	var conversation models.ChatConversation
	err := config.DB.Preload("Umkm").Preload("Supplier").
		Where("umkm_id = ? AND supplier_id = ?", userID, supplierID).
		First(&conversation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		conversation = models.ChatConversation{
			UmkmID:     userID,
			SupplierID: supplier.ID,
		}
		if err := config.DB.Create(&conversation).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat chat supplier"})
			return
		}
		if err := config.DB.Preload("Umkm").Preload("Supplier").First(&conversation, "id = ?", conversation.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat chat supplier"})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat chat supplier"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   chatConversationPayload(conversation, userID, role),
	})
}

func GetChatMessages(c *gin.Context) {
	userID, role, ok := getChatActor(c)
	if !ok {
		return
	}

	conversation, ok := getAccessibleConversation(c, userID, role)
	if !ok {
		return
	}

	now := time.Now()
	if err := config.DB.Model(&models.ChatMessage{}).
		Where("conversation_id = ? AND receiver_id = ? AND is_read = ?", conversation.ID, userID, false).
		Updates(map[string]interface{}{"is_read": true, "read_at": &now}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menandai chat terbaca"})
		return
	}

	var messages []models.ChatMessage
	if err := config.DB.Preload("Sender").
		Where("conversation_id = ?", conversation.ID).
		Order("created_at ASC").
		Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil pesan"})
		return
	}

	response := make([]gin.H, 0, len(messages))
	for _, message := range messages {
		response = append(response, chatMessagePayload(message))
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"conversation": chatConversationPayload(conversation, userID, role),
		"data":         response,
	})
}

func SendChatMessage(c *gin.Context) {
	userID, role, ok := getChatActor(c)
	if !ok {
		return
	}

	conversation, ok := getAccessibleConversation(c, userID, role)
	if !ok {
		return
	}

	var input sendChatMessageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	body := strings.TrimSpace(input.Message)
	if body == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pesan tidak boleh kosong"})
		return
	}
	if len(body) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pesan maksimal 2000 karakter"})
		return
	}

	receiverID := conversation.SupplierID
	receiverRole := string(models.RoleSupplier)
	if role == string(models.RoleSupplier) {
		receiverID = conversation.UmkmID
		receiverRole = string(models.RoleUser)
	}

	now := time.Now()
	message := models.ChatMessage{
		ConversationID: conversation.ID,
		SenderID:       userID,
		ReceiverID:     receiverID,
		Message:        body,
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&message).Error; err != nil {
			return err
		}

		return tx.Model(&models.ChatConversation{}).
			Where("id = ?", conversation.ID).
			Updates(map[string]interface{}{
				"last_message":    body,
				"last_sender_id":  userID,
				"last_message_at": &now,
			}).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengirim pesan"})
		return
	}

	var sender models.User
	_ = config.DB.First(&sender, "id = ?", userID).Error
	senderName := sender.BusinessName
	if senderName == "" {
		senderName = "Pengguna SupplierHub"
	}
	_, _ = services.CreateNotification(nil, models.Notification{
		UserID:     receiverID,
		Role:       receiverRole,
		Title:      "Pesan baru dari " + senderName,
		Message:    shortChatPreview(body, 120),
		Type:       "chat_message",
		SourceType: "chat",
		SourceID:   conversation.ID,
	})

	if err := config.DB.Preload("Sender").First(&message, "id = ?", message.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Pesan terkirim, tetapi gagal dimuat ulang"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Pesan berhasil dikirim",
		"data":    chatMessagePayload(message),
	})
}

func MarkChatConversationRead(c *gin.Context) {
	userID, role, ok := getChatActor(c)
	if !ok {
		return
	}

	conversation, ok := getAccessibleConversation(c, userID, role)
	if !ok {
		return
	}

	now := time.Now()
	if err := config.DB.Model(&models.ChatMessage{}).
		Where("conversation_id = ? AND receiver_id = ? AND is_read = ?", conversation.ID, userID, false).
		Updates(map[string]interface{}{"is_read": true, "read_at": &now}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menandai chat terbaca"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Chat sudah ditandai terbaca",
	})
}

func getAccessibleConversation(c *gin.Context, userID, role string) (models.ChatConversation, bool) {
	conversationID := c.Param("id")
	var conversation models.ChatConversation
	query := config.DB.Preload("Umkm").Preload("Supplier").Where("id = ?", conversationID)
	if role == string(models.RoleSupplier) {
		query = query.Where("supplier_id = ?", userID)
	} else {
		query = query.Where("umkm_id = ?", userID)
	}

	if err := query.First(&conversation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat tidak ditemukan"})
			return conversation, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat chat"})
		return conversation, false
	}

	return conversation, true
}

func chatConversationPayload(conversation models.ChatConversation, viewerID, viewerRole string) gin.H {
	var unreadCount int64
	config.DB.Model(&models.ChatMessage{}).
		Where("conversation_id = ? AND receiver_id = ? AND is_read = ?", conversation.ID, viewerID, false).
		Count(&unreadCount)

	counterpart := conversation.Supplier
	counterpartRole := string(models.RoleSupplier)
	if viewerRole == string(models.RoleSupplier) {
		counterpart = conversation.Umkm
		counterpartRole = string(models.RoleUser)
	}

	return gin.H{
		"id":              conversation.ID,
		"umkm_id":         conversation.UmkmID,
		"supplier_id":     conversation.SupplierID,
		"last_message":    conversation.LastMessage,
		"last_sender_id":  conversation.LastSenderID,
		"last_message_at": conversation.LastMessageAt,
		"unread_count":    unreadCount,
		"created_at":      conversation.CreatedAt,
		"updated_at":      conversation.UpdatedAt,
		"counterpart": gin.H{
			"id":            counterpart.ID,
			"business_name": counterpart.BusinessName,
			"email":         counterpart.Email,
			"role":          counterpartRole,
			"address":       counterpart.Address,
			"category":      counterpart.Category,
			"region":        counterpart.Region,
			"phone":         counterpart.Phone,
		},
		"umkm": gin.H{
			"id":            conversation.Umkm.ID,
			"business_name": conversation.Umkm.BusinessName,
			"email":         conversation.Umkm.Email,
		},
		"supplier": gin.H{
			"id":            conversation.Supplier.ID,
			"business_name": conversation.Supplier.BusinessName,
			"email":         conversation.Supplier.Email,
			"category":      conversation.Supplier.Category,
			"region":        conversation.Supplier.Region,
		},
	}
}

func chatMessagePayload(message models.ChatMessage) gin.H {
	return gin.H{
		"id":              message.ID,
		"conversation_id": message.ConversationID,
		"sender_id":       message.SenderID,
		"receiver_id":     message.ReceiverID,
		"message":         message.Message,
		"is_read":         message.IsRead,
		"read_at":         message.ReadAt,
		"created_at":      message.CreatedAt,
		"sender": gin.H{
			"id":            message.Sender.ID,
			"business_name": message.Sender.BusinessName,
			"email":         message.Sender.Email,
			"role":          message.Sender.Role,
		},
	}
}

func shortChatPreview(value string, maxLength int) string {
	text := strings.TrimSpace(value)
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength-3] + "..."
}
