package tickets

import (
	"fmt"

	"gorm.io/gorm"
)

func CloseTicket(db *gorm.DB, ticketID int) error {
	result := db.Model(&Ticket{}).Where("ticket_id = ?", ticketID).Update("status", "closed")
	if result.Error != nil {
		return fmt.Errorf("关闭工单失败: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到指定的工单")
	}
	return nil
}
