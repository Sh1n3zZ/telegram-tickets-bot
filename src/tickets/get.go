package tickets

import (
	"gorm.io/gorm"
)

func GetUserTickets(db *gorm.DB, telegramID int64) ([]Ticket, error) {
	var tickets []Ticket
	err := db.Joins("JOIN regular_users ON tickets.created_by = regular_users.user_id").
		Where("regular_users.telegram_id = ?", telegramID).
		Find(&tickets).Error
	return tickets, err
}

func GetTicketByID(db *gorm.DB, ticketID int) (*Ticket, error) {
	var ticket Ticket
	err := db.First(&ticket, ticketID).Error
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

func GetAllTickets(db *gorm.DB) ([]Ticket, error) {
	var tickets []Ticket
	if err := db.Order("created_at desc").Find(&tickets).Error; err != nil {
		return nil, err
	}
	return tickets, nil
}
