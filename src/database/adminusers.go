package database

import (
	"fmt"

	"gorm.io/gorm"
)

type AdminUser struct {
	AdminID    int    `gorm:"primaryKey;column:admin_id"`
	Username   string `gorm:"column:username"`
	FullName   string `gorm:"column:full_name"`
	Position   string `gorm:"column:position"`
	TelegramID int64  `gorm:"column:telegram_id"`
}

func (AdminUser) TableName() string {
	return "admin_users"
}

func GetAdminByID(db *gorm.DB, adminID int) (*AdminUser, error) {
	var admin AdminUser
	if err := db.Where("admin_id = ?", adminID).First(&admin).Error; err != nil {
		return nil, fmt.Errorf("[ERROR] Failed to fetch admin information: %v", err)
	}
	return &admin, nil
}

func IsUserAdmin(telegramID int64) (bool, error) {
	db, err := InitializeDB()
	if err != nil {
		return false, err
	}

	var count int64
	if err := db.Model(&AdminUser{}).Where("telegram_id = ?", telegramID).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func GetAdminIDByTelegramID(db *gorm.DB, telegramID int64) (int, error) {
	var admin AdminUser
	if err := db.Where("telegram_id = ?", telegramID).First(&admin).Error; err != nil {
		return 0, err
	}
	return admin.AdminID, nil
}

func GetTelegramIDByUserID(db *gorm.DB, userID int) (int64, error) {
	var user RegularUser
	result := db.Select("telegram_id").Where("user_id = ?", userID).First(&user)
	if result.Error != nil {
		return 0, fmt.Errorf("[ERROR] Failed to get Telegram ID for user %d: %v", userID, result.Error)
	}
	return user.TelegramID, nil
}
