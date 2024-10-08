package tickets

import (
	"fmt"

	"gorm.io/gorm"
)

func CloseTicket(db *gorm.DB, ticketID int) error {
	result := db.Model(&Ticket{}).Where("ticket_id = ?", ticketID).Update("status", "closed")
	if result.Error != nil {
		return fmt.Errorf("[ERROR] Failed to close ticket: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("[WARNING] Ticket not found")
	}
	return nil
}
