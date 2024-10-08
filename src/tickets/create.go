package tickets

import (
	"fmt"
	"telegram-tickets-bot/src/database"
	"time"

	"gorm.io/gorm"
)

type Ticket struct {
	TicketID    int       `gorm:"primaryKey;column:ticket_id"`
	Title       string    `gorm:"column:title"`
	Description string    `gorm:"column:description"`
	Status      string    `gorm:"column:status"`
	Priority    string    `gorm:"column:priority"`
	CreatedBy   int       `gorm:"column:created_by"`
	AssignedTo  *int      `gorm:"column:assigned_to"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

type TicketCreationData struct {
	Title       string
	Description string
	TicketID    int
}

func (Ticket) TableName() string {
	return "tickets"
}

func CreateTicket(db *gorm.DB, telegramID int64, title string, description string, priority string) (*Ticket, error) {
	// 检查用户是否已注册，如果未注册则自动注册
	user, err := database.CheckAndRegisterUser(db, telegramID)
	if err != nil {
		return nil, fmt.Errorf("检查和注册用户失败: %v", err)
	}

	// 获取最大的 ticket_id
	var maxTicketID int
	err = db.Model(&Ticket{}).Select("COALESCE(MAX(ticket_id), 0)").Scan(&maxTicketID).Error
	if err != nil {
		return nil, fmt.Errorf("获取最大 ticket_id 失败: %v", err)
	}

	// 创建新工单
	ticket := Ticket{
		TicketID:    maxTicketID + 1,
		Title:       title,
		Description: description,
		Status:      "open",
		Priority:    priority,
		CreatedBy:   user.UserID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	result := db.Create(&ticket)
	if result.Error != nil {
		return nil, fmt.Errorf("创建工单失败: %v", result.Error)
	}

	return &ticket, nil
}
