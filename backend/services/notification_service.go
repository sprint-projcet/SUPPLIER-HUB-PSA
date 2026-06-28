package services

import (
	"strings"
	"time"

	"supplierhub-backend/config"
	"supplierhub-backend/models"

	"gorm.io/gorm"
)

func notificationDB(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return config.DB
}

func CreateNotification(tx *gorm.DB, notification models.Notification) (models.Notification, error) {
	db := notificationDB(tx)
	if err := db.Create(&notification).Error; err != nil {
		return notification, err
	}
	return notification, nil
}

func CreateRoleNotifications(tx *gorm.DB, role models.Role, notification models.Notification) ([]models.Notification, error) {
	db := notificationDB(tx)

	var users []models.User
	if err := db.Where("role = ?", role).Find(&users).Error; err != nil {
		return nil, err
	}

	notifications := make([]models.Notification, 0, len(users))
	for _, user := range users {
		item := notification
		item.ID = ""
		item.UserID = user.ID
		item.Role = string(role)

		created, err := CreateNotification(tx, item)
		if err != nil {
			return notifications, err
		}
		notifications = append(notifications, created)
	}

	return notifications, nil
}

func ActivityTimeLabel(value time.Time) string {
	if value.IsZero() {
		value = time.Now()
	}
	return value.Local().Format("02/01/2006 15:04")
}

func ActivityMessage(message string, value time.Time) string {
	message = strings.TrimSpace(message)
	timestamp := "Dicatat pada " + ActivityTimeLabel(value) + " WIB."
	if message == "" {
		return timestamp
	}
	return message + " " + timestamp
}

func CreateActivityLog(tx *gorm.DB, userID, action, description string) error {
	db := notificationDB(tx)
	return db.Create(&models.Log{
		UserID:      userID,
		Action:      strings.ToUpper(strings.TrimSpace(action)),
		Description: ActivityMessage(description, time.Now()),
	}).Error
}

func CreateActivityNotification(tx *gorm.DB, notification models.Notification) (models.Notification, error) {
	now := time.Now()
	notification.Message = ActivityMessage(notification.Message, now)
	return CreateNotification(tx, notification)
}

func CreateRoleActivityNotifications(tx *gorm.DB, role models.Role, notification models.Notification) ([]models.Notification, error) {
	now := time.Now()
	notification.Message = ActivityMessage(notification.Message, now)
	return CreateRoleNotifications(tx, role, notification)
}
