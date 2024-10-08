package tickets

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type TicketComment struct {
	CommentID int       `gorm:"primaryKey;column:comment_id"`
	TicketID  int       `gorm:"column:ticket_id"`
	UserID    int       `gorm:"column:user_id"`  // 修改为 int
	AdminID   *int      `gorm:"column:admin_id"` // 修改为 int
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
		return fmt.Errorf("获取最大 comment_id 失败: %v", err)
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
		return fmt.Errorf("添加评论失败: %v", result.Error)
	}

	return nil
}

func GetTicketComments(db *gorm.DB, ticketID int) ([]TicketComment, error) {
	var comments []TicketComment
	err := db.Where("ticket_id = ?", ticketID).Order("created_at ASC").Find(&comments).Error
	if err != nil {
		return nil, fmt.Errorf("获取工单评论失败: %v", err)
	}
	return comments, nil
}
