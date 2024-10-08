package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type RegularUser struct {
	UserID     int       `gorm:"primaryKey;column:user_id"`
	UserGroup  string    `gorm:"column:user_group"`
	TelegramID int64     `gorm:"uniqueIndex;column:telegram_id"`
	CreatedAt  time.Time `gorm:"column:created_at;type:datetime"`
}

func (RegularUser) TableName() string {
	return "regular_users"
}

func CreateRegularUser(db *gorm.DB, telegramID int64) error {
	var maxUserID int
	err := db.Model(&RegularUser{}).Select("COALESCE(MAX(user_id), 0)").Scan(&maxUserID).Error
	if err != nil {
		return fmt.Errorf("获取最大 user_id 失败: %v", err)
	}

	// 创建新用户
	user := RegularUser{
		UserID:     maxUserID + 1,
		TelegramID: telegramID,
		UserGroup:  "Default",
		CreatedAt:  time.Now(),
	}

	result := db.Create(&user)
	return result.Error
}

func GetRegularUserByTelegramID(db *gorm.DB, telegramID int64) (*RegularUser, error) {
	var user RegularUser
	result := db.Where("telegram_id = ?", telegramID).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func CheckAndRegisterUser(db *gorm.DB, telegramID int64) (*RegularUser, error) {
	user, err := GetRegularUserByTelegramID(db, telegramID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 用户未注册，自动注册
			err = CreateRegularUser(db, telegramID)
			if err != nil {
				return nil, fmt.Errorf("创建用户失败: %v", err)
			}
			// 重新获取用户信息
			user, err = GetRegularUserByTelegramID(db, telegramID)
			if err != nil {
				return nil, fmt.Errorf("获取新创建的用户信息失败: %v", err)
			}
		} else {
			return nil, fmt.Errorf("查询用户信息失败: %v", err)
		}
	}

	return user, nil
}

func GetUserIDByTelegramID(db *gorm.DB, telegramID int64) (int, error) {
	var user RegularUser
	result := db.Select("user_id").Where("telegram_id = ?", telegramID).First(&user)
	if result.Error != nil {
		return 0, result.Error
	}
	return user.UserID, nil
}
