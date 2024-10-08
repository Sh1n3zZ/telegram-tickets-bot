package tickets

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type TicketComment struct {
	CommentID int       `gorm:"primaryKey;column:comment_id"`
	TicketID  int       `gorm:"column:ticket_id"`
	UserID    int       `gorm:"column:user_id"`
	AdminID   *int      `gorm:"column:admin_id"`
	Content   string    `gorm:"column:content"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (TicketComment) TableName() string {
	return "ticket_comments"
}

func AddComment(db *gorm.DB, ticketID int, userID int, content string) error {
	var maxCommentID int
	err := db.Model(&TicketComment{}).Select("COALESCE(MAX(comment_id), 0)").Scan(&maxCommentID).Error
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get max comment_id: %v", err)
	}

	comment := TicketComment{
		CommentID: maxCommentID + 1,
		TicketID:  ticketID,
		UserID:    userID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	result := db.Create(&comment)
	if result.Error != nil {
		return fmt.Errorf("[ERROR] Failed to add comment: %v", result.Error)
	}

	return nil
}

func GetTicketComments(db *gorm.DB, ticketID int) ([]TicketComment, error) {
	var comments []TicketComment
	err := db.Where("ticket_id = ?", ticketID).Order("created_at ASC").Find(&comments).Error
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Failed to get ticket comments: %v", err)
	}
	return comments, nil
}
