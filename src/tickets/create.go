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
	// Check if the user is registered, if not, automatically register them
	user, err := database.CheckAndRegisterUser(db, telegramID)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Failed to check and register user: %v", err)
	}

	// Get the maximum ticket_id
	var maxTicketID int
	err = db.Model(&Ticket{}).Select("COALESCE(MAX(ticket_id), 0)").Scan(&maxTicketID).Error
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Failed to get maximum ticket_id: %v", err)
	}

	// Create a new ticket
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
		return nil, fmt.Errorf("[ERROR] Failed to create ticket: %v", result.Error)
	}

	return &ticket, nil
}
