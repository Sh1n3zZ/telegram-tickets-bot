package telegram

import (
	"fmt"
	"log"
	"strings"
	"telegram-tickets-bot/src/database"
	"telegram-tickets-bot/src/tickets"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Store user's current conversation state
var userStates = make(map[int64]string)
var ticketData = make(map[int64]*tickets.TicketCreationData)

// Add new user states
const (
	StateNone              = ""
	StateWaitingForTitle   = "waiting_for_title"
	StateWaitingForDesc    = "waiting_for_description"
	StateWaitingForComment = "waiting_for_comment"
)

func (b *Bot) HandleGetMeCommand(message *tgbotapi.Message) error {
	user := message.From
	fullName := user.FirstName
	if user.LastName != "" {
		fullName += " " + user.LastName
	}

	username := "未设置"
	if user.UserName != "" {
		username = "@" + user.UserName
	}

	// Get database connection
	db, err := database.InitializeDB()
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get database connection: %v", err)
	}

	// Check if user is registered, if not, register automatically
	regularUser, err := database.CheckAndRegisterUser(db, int64(user.ID))
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to check and register user: %v", err)
	}

	infoText := fmt.Sprintf("您的信息:\n"+
		"用户ID: %d\n"+
		"全名: %s\n"+
		"用户名: %s\n"+
		"Telegram ID: %d\n"+
		"消息时间: %s\n"+
		"用户组: %s\n"+
		"注册时间: %s",
		regularUser.UserID,
		fullName, username, user.ID,
		message.Time().Format("2006-01-02 15:04:05"),
		regularUser.UserGroup,
		regularUser.CreatedAt.Format("2006-01-02 15:04:05"))

	// Get user's profile photo
	photos, err := b.api.GetUserProfilePhotos(tgbotapi.UserProfilePhotosConfig{UserID: user.ID, Limit: 1})
	if err != nil {
		return err
	}

	if photos.TotalCount > 0 {
		// User has a profile photo, send message with photo
		fileID := photos.Photos[0][0].FileID
		photoMsg := tgbotapi.NewPhoto(message.Chat.ID, tgbotapi.FileID(fileID))
		photoMsg.Caption = infoText
		_, err = b.api.Send(photoMsg)
	} else {
		// User has no profile photo, send text message only
		err = b.SendMessage(message.Chat.ID, infoText)
	}

	if err != nil {
		return err
	}

	return nil
}

func (b *Bot) HandleHelpCommand(message *tgbotapi.Message) error {
	isAdmin, err := database.IsUserAdmin(message.From.ID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to check admin status: %v", err)
	}

	var keyboard tgbotapi.InlineKeyboardMarkup
	if isAdmin {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("创建工单", "create_ticket"),
				tgbotapi.NewInlineKeyboardButtonData("查看我的工单", "view_tickets"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("获取个人信息", "get_info"),
				tgbotapi.NewInlineKeyboardButtonData("查看所有工单", "view_all_tickets"),
			),
		)
	} else {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("创建工单", "create_ticket"),
				tgbotapi.NewInlineKeyboardButtonData("查看我的工单", "view_tickets"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("获取个人信息", "get_info"),
			),
		)
	}

	helpText := "欢迎使用帮助菜单,请选择以下选项:"
	return b.SendMessageWithInlineKeyboard(message.Chat.ID, helpText, keyboard)
}

func (b *Bot) HandleCallbackQuery(callbackQuery *tgbotapi.CallbackQuery) error {
	chatID := callbackQuery.Message.Chat.ID
	data := callbackQuery.Data

	switch {
	case data == "create_ticket":
		userStates[chatID] = "waiting_for_title"
		ticketData[chatID] = &tickets.TicketCreationData{}
		return b.SendMessage(chatID, "请输入工单标题：")
	case data == "view_tickets":
		return b.HandleViewTickets(&tgbotapi.Message{
			From: callbackQuery.From,
			Chat: callbackQuery.Message.Chat,
		})
	case data == "get_info":
		userMessage := &tgbotapi.Message{
			From: callbackQuery.From,
			Chat: callbackQuery.Message.Chat,
			Date: int(callbackQuery.Message.Date),
		}
		return b.HandleGetMeCommand(userMessage)
	case data == "confirm_ticket":
		return b.CreateTicket(chatID)
	case data == "cancel_ticket":
		delete(userStates, chatID)
		delete(ticketData, chatID)
		return b.SendMessage(chatID, "工单创建已取消。")
	case data[:11] == "view_ticket":
		return b.HandleTicketView(callbackQuery)
	case data[:12] == "close_ticket":
		return b.HandleCloseTicket(callbackQuery)
	case data[:11] == "add_comment":
		return b.HandleAddComment(callbackQuery)
	case strings.HasPrefix(data, "assign_ticket_"):
		return b.HandleAssignTicket(callbackQuery)
	case strings.HasPrefix(data, "assign_to_"):
		var ticketID, adminID int
		_, err := fmt.Sscanf(data, "assign_to_%d_%d", &ticketID, &adminID)
		if err != nil {
			return fmt.Errorf("[ERROR] Failed to parse assign data: %v", err)
		}
		return b.AssignTicketToAdmin(ticketID, adminID)
	case strings.HasPrefix(data, "reply_ticket_"):
		var ticketID int
		_, err := fmt.Sscanf(data, "reply_ticket_%d", &ticketID)
		if err != nil {
			return fmt.Errorf("[ERROR] Failed to parse ticket ID: %v", err)
		}
		userStates[chatID] = StateWaitingForComment
		ticketData[chatID] = &tickets.TicketCreationData{TicketID: ticketID}
		return b.SendMessage(chatID, "请输入您的回复：")
	case data == "view_all_tickets":
		return b.HandleAdminViewTickets(&tgbotapi.Message{
			From: callbackQuery.From,
			Chat: callbackQuery.Message.Chat,
		})
	default:
		return b.SendMessage(chatID, "Unknown option.")
	}
}

func (b *Bot) HandleMessage(message *tgbotapi.Message) error {
	chatID := message.Chat.ID
	text := message.Text

	// 检查用户是否为管理员
	isAdmin, err := database.IsUserAdmin(message.From.ID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to check admin status: %v", err)
	}

	switch userStates[chatID] {
	case "waiting_for_title":
		ticketData[chatID].Title = text
		userStates[chatID] = "waiting_for_description"
		return b.SendMessage(chatID, "请输入工单描述：")
	case "waiting_for_description":
		ticketData[chatID].Description = text
		return b.ConfirmTicketCreation(chatID)
	case StateWaitingForComment:
		if isAdmin {
			return b.AddAdminCommentToTicket(chatID, message.From.ID, text, ticketData[chatID].TicketID)
		}
		return b.AddCommentToTicket(chatID, message.From.ID, text)
	default:
		return b.SendMessage(chatID, "我不明白您的意思。请使用 /help 查看可用命令。")
	}
}

func (b *Bot) AddAdminCommentToTicket(chatID int64, telegramUserID int64, content string, ticketID int) error {
	db, err := database.InitializeDB()
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get database connection: %v", err)
	}

	// Get admin ID
	adminID, err := database.GetAdminIDByTelegramID(db, telegramUserID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get admin ID: %v", err)
	}

	// Add admin comment
	err = tickets.AddAdminComment(db, ticketID, adminID, content)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to add admin comment: %v", err)
	}

	// Get ticket information
	ticket, err := tickets.GetTicketByID(db, ticketID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get ticket: %v", err)
	}

	log.Printf("[DEBUG] Fetched ticket: %+v", ticket)

	// Notify the user
	userMessage := fmt.Sprintf("工单 #%d 有来自 Staff 的新回复：\n%s", ticketID, content)

	// Directly get the user's Telegram ID
	userTelegramID, err := database.GetTelegramIDByUserID(db, ticket.CreatedBy)
	if err != nil {
		log.Printf("[ERROR] Failed to get Telegram ID for user %d: %v", ticket.CreatedBy, err)
		return fmt.Errorf("failed to get user Telegram ID: %v", err)
	}

	log.Printf("[DEBUG] User %d Telegram ID: %d", ticket.CreatedBy, userTelegramID)

	// Send message using the obtained Telegram ID
	err = b.SendMessage(userTelegramID, userMessage)
	if err != nil {
		log.Printf("[ERROR] Failed to notify user using Telegram ID %d for ticket #%d: %v", userTelegramID, ticketID, err)
		return fmt.Errorf("failed to notify user: %v", err)
	}

	log.Printf("[INFO] Successfully notified user %d for ticket #%d using Telegram ID", ticket.CreatedBy, ticketID)

	// Display ticket information
	log.Printf("[DEBUG] Calling HandleTicketView from AddAdminCommentToTicket with chatID: %d, ticketID: %d, telegramUserID: %d", chatID, ticketID, telegramUserID)
	err = b.HandleTicketView(&tgbotapi.CallbackQuery{
		Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}},
		Data:    fmt.Sprintf("view_ticket_%d", ticketID),
		From:    &tgbotapi.User{ID: telegramUserID},
	})
	if err != nil {
		log.Printf("[ERROR] HandleTicketView failed: %v", err)
		return err
	}
	log.Printf("[DEBUG] HandleTicketView completed successfully")
	return nil
}

func (b *Bot) ConfirmTicketCreation(chatID int64) error {
	data := ticketData[chatID]
	confirmationText := fmt.Sprintf("请确认工单信息：\n标题：%s\n描述：%s\n\n是否创建工单？", data.Title, data.Description)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("确认", "confirm_ticket"),
			tgbotapi.NewInlineKeyboardButtonData("取消", "cancel_ticket"),
		),
	)

	return b.SendMessageWithInlineKeyboard(chatID, confirmationText, keyboard)
}

func (b *Bot) HandleTicketConfirmation(callbackQuery *tgbotapi.CallbackQuery) error {
	chatID := callbackQuery.Message.Chat.ID
	data := callbackQuery.Data

	switch data {
	case "confirm_ticket":
		return b.CreateTicket(chatID)
	case "cancel_ticket":
		delete(userStates, chatID)
		delete(ticketData, chatID)
		return b.SendMessage(chatID, "工单创建已取消。")
	default:
		return b.SendMessage(chatID, "未知的选项。")
	}
}

func (b *Bot) CreateTicket(chatID int64) error {
	data := ticketData[chatID]

	db, err := database.InitializeDB()
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get database connection: %v", err)
	}

	ticket, err := tickets.CreateTicket(db, chatID, data.Title, data.Description, "normal")
	if err != nil {
		return b.SendMessage(chatID, fmt.Sprintf("[ERROR] Failed to create ticket: %v", err))
	}

	// Notify all administrators
	if err := b.NotifyAllAdmins(ticket); err != nil {
		log.Printf("[ERROR] Failed to notify admins: %v", err)
	}

	delete(userStates, chatID)
	delete(ticketData, chatID)

	successMsg := fmt.Sprintf("工单创建成功。工单ID: %d", ticket.TicketID)
	err = b.SendMessage(chatID, successMsg)
	if err != nil {
		return err
	}

	// Display details of the newly created ticket
	return b.HandleTicketView(&tgbotapi.CallbackQuery{
		Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}},
		Data:    fmt.Sprintf("view_ticket_%d", ticket.TicketID),
	})
}

func (b *Bot) HandleViewTickets(message *tgbotapi.Message) error {
	chatID := message.Chat.ID
	telegramID := message.From.ID

	db, err := database.InitializeDB()
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get database connection: %v", err)
	}

	userTickets, err := tickets.GetUserTickets(db, int64(telegramID))
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get user tickets: %v", err)
	}

	if len(userTickets) == 0 {
		return b.SendMessage(chatID, "您目前没有任何工单。")
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup()
	for _, ticket := range userTickets {
		buttonText := fmt.Sprintf("#%d: %s (%s)", ticket.TicketID, ticket.Title, ticket.Status)
		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("view_ticket_%d", ticket.TicketID))
		row := tgbotapi.NewInlineKeyboardRow(button)
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
	}

	return b.SendMessageWithInlineKeyboard(chatID, "您的工单列表：", keyboard)
}

func (b *Bot) HandleTicketView(callbackQuery *tgbotapi.CallbackQuery) error {
	log.Printf("[DEBUG] Entering HandleTicketView")

	if callbackQuery == nil {
		return fmt.Errorf("[ERROR] Callback query is nil")
	}
	if callbackQuery.Message == nil {
		return fmt.Errorf("[ERROR] Callback query message is nil")
	}
	if callbackQuery.Message.Chat == nil {
		return fmt.Errorf("[ERROR] Callback query chat is nil")
	}
	if callbackQuery.From == nil {
		return fmt.Errorf("[ERROR] Callback query From is nil")
	}

	chatID := callbackQuery.Message.Chat.ID
	data := callbackQuery.Data

	log.Printf("[DEBUG] HandleTicketView called with chatID: %d, data: %s, From.ID: %d", chatID, data, callbackQuery.From.ID)

	// Extract ticket ID from callback data
	var ticketID int
	_, err := fmt.Sscanf(data, "view_ticket_%d", &ticketID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to parse ticket ID: %v", err)
	}

	log.Printf("[DEBUG] Parsed ticketID: %d", ticketID)

	db, err := database.InitializeDB()
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get database connection: %v", err)
	}

	ticket, err := tickets.GetTicketByID(db, ticketID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get ticket information: %v", err)
	}

	log.Printf("[DEBUG] Retrieved ticket: %+v", ticket)

	ticketInfo := fmt.Sprintf("工单 #%d\n标题: %s\n描述: %s\n状态: %s\n优先级: %s\n创建时间: %s",
		ticket.TicketID, ticket.Title, ticket.Description, ticket.Status, ticket.Priority, ticket.CreatedAt.Format("2006-01-02 15:04:05"))

	log.Printf("[DEBUG] Constructed ticketInfo: %s", ticketInfo)

	// 检查用户是否为管理员
	isAdmin, err := database.IsUserAdmin(callbackQuery.From.ID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to check admin status: %v", err)
	}

	log.Printf("[DEBUG] User admin status: %v", isAdmin)

	var keyboard tgbotapi.InlineKeyboardMarkup
	if ticket.Status == "closed" {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("返回列表", "view_tickets"),
			),
		)
	} else {
		if isAdmin {
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("回复", fmt.Sprintf("reply_ticket_%d", ticket.TicketID)),
					tgbotapi.NewInlineKeyboardButtonData("关闭工单", fmt.Sprintf("close_ticket_%d", ticket.TicketID)),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("返回列表", "view_tickets"),
				),
			)
		} else {
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("添加评论", fmt.Sprintf("add_comment_%d", ticket.TicketID)),
					tgbotapi.NewInlineKeyboardButtonData("关闭工单", fmt.Sprintf("close_ticket_%d", ticket.TicketID)),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("返回列表", "view_tickets"),
				),
			)
		}
	}

	log.Printf("[DEBUG] Constructed keyboard: %+v", keyboard)

	// Fetch ticket comments
	comments, err := tickets.GetTicketComments(db, ticketID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to fetch ticket comments: %v", err)
	}

	log.Printf("[DEBUG] Retrieved %d comments", len(comments))

	// Add comments to ticket information
	for _, comment := range comments {
		if comment.AdminID != nil {
			// Fetch admin information
			admin, err := database.GetAdminByID(db, *comment.AdminID)
			if err != nil {
				log.Printf("[ERROR] Failed to fetch admin information: %v", err)
				continue
			}
			ticketInfo += fmt.Sprintf("\n\n[Staff] %s (Global Comment ID: %d):\n%s\n\nRegards,\n%s\n%s\nTime: %s",
				admin.FullName,
				comment.CommentID,
				comment.Content,
				admin.FullName,
				admin.Position,
				comment.CreatedAt.Format("2006-01-02 15:04:05"))
		} else {
			ticketInfo += fmt.Sprintf("\n\nUser Comment (ID: %d):\n%s\nTime: %s",
				comment.CommentID,
				comment.Content,
				comment.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	}

	log.Printf("[DEBUG] Sending message with inline keyboard")
	err = b.SendMessageWithInlineKeyboard(chatID, ticketInfo, keyboard)
	if err != nil {
		log.Printf("[ERROR] Failed to send message with inline keyboard: %v", err)
		return fmt.Errorf("[ERROR] Failed to send message with inline keyboard: %v", err)
	}

	log.Printf("[INFO] Successfully sent ticket view for ticket #%d", ticketID)
	log.Printf("[DEBUG] Exiting HandleTicketView")
	return nil
}

func (b *Bot) HandleCloseTicket(callbackQuery *tgbotapi.CallbackQuery) error {
	chatID := callbackQuery.Message.Chat.ID
	data := callbackQuery.Data

	var ticketID int
	_, err := fmt.Sscanf(data, "close_ticket_%d", &ticketID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to parse ticket ID: %v", err)
	}

	db, err := database.InitializeDB()
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get database connection: %v", err)
	}

	err = tickets.CloseTicket(db, ticketID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to close ticket: %v", err)
	}

	// Update inline keyboard, remove "Close ticket" button
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("返回列表", "view_tickets"),
		),
	)

	// Update message text, show ticket is closed
	updatedText := fmt.Sprintf("%s\n\n工单已关闭", callbackQuery.Message.Text)

	editMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, callbackQuery.Message.MessageID, updatedText, keyboard)
	_, err = b.api.Send(editMsg)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to update message: %v", err)
	}

	return nil
}

// Add new function to handle adding comments
func (b *Bot) HandleAddComment(callbackQuery *tgbotapi.CallbackQuery) error {
	chatID := callbackQuery.Message.Chat.ID
	data := callbackQuery.Data

	var ticketID int
	_, err := fmt.Sscanf(data, "add_comment_%d", &ticketID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to parse ticket ID: %v", err)
	}

	userStates[chatID] = StateWaitingForComment
	ticketData[chatID] = &tickets.TicketCreationData{TicketID: ticketID}

	return b.SendMessage(chatID, "请输入您的评论：")
}

// AddCommentToTicket adds a comment to the ticket
func (b *Bot) AddCommentToTicket(chatID int64, telegramUserID int64, content string) error {
	data := ticketData[chatID]

	db, err := database.InitializeDB()
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get database connection: %v", err)
	}

	// Get user ID
	userID, err := database.GetUserIDByTelegramID(db, telegramUserID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get user ID: %v", err)
	}

	// Add comment
	err = tickets.AddComment(db, data.TicketID, userID, content)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to add comment: %v", err)
	}

	// Get ticket information
	ticket, err := tickets.GetTicketByID(db, data.TicketID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get ticket: %v", err)
	}

	// Notify assigned admin
	if ticket.AssignedTo != nil {
		comment := &tickets.TicketComment{
			TicketID: data.TicketID,
			UserID:   &userID,
			Content:  content,
		}
		if err := b.NotifyAssignedAdmin(ticket, comment); err != nil {
			log.Printf("[ERROR] Failed to notify assigned admin: %v", err)
		}
	}

	delete(userStates, chatID)
	delete(ticketData, chatID)

	// Display ticket information
	return b.HandleTicketView(&tgbotapi.CallbackQuery{
		Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}},
		Data:    fmt.Sprintf("view_ticket_%d", data.TicketID),
	})
}

func (b *Bot) HandleAssignTicket(callbackQuery *tgbotapi.CallbackQuery) error {
	chatID := callbackQuery.Message.Chat.ID
	data := callbackQuery.Data

	var ticketID int
	_, err := fmt.Sscanf(data, "assign_ticket_%d", &ticketID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to parse ticket ID: %v", err)
	}

	db, err := database.InitializeDB()
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get database connection: %v", err)
	}

	var admins []database.AdminUser
	if err := db.Find(&admins).Error; err != nil {
		return fmt.Errorf("[ERROR] Failed to fetch admin users: %v", err)
	}

	var keyboard tgbotapi.InlineKeyboardMarkup
	for _, admin := range admins {
		button := tgbotapi.NewInlineKeyboardButtonData(
			admin.FullName,
			fmt.Sprintf("assign_to_%d_%d", ticketID, admin.AdminID),
		)
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, []tgbotapi.InlineKeyboardButton{button})
	}

	return b.SendMessageWithInlineKeyboard(chatID, "请选择要分配给的管理员:", keyboard)
}

// AssignTicketToAdmin assigns the ticket to the specified admin
func (b *Bot) AssignTicketToAdmin(ticketID int, adminID int) error {
	db, err := database.InitializeDB()
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get database connection: %v", err)
	}

	if err := db.Model(&tickets.Ticket{}).Where("ticket_id = ?", ticketID).Update("assigned_to", adminID).Error; err != nil {
		return fmt.Errorf("[ERROR] Failed to assign ticket: %v", err)
	}

	// Notify the assigned admin
	admin, err := database.GetAdminByID(db, adminID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get admin info: %v", err)
	}

	ticket, err := tickets.GetTicketByID(db, ticketID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get ticket info: %v", err)
	}

	message := fmt.Sprintf("工单已分配给您:\n工单ID: %d\n标题: %s\n描述: %s",
		ticket.TicketID, ticket.Title, ticket.Description)

	return b.SendMessage(admin.TelegramID, message)
}

// NotifyAssignedAdmin notifies the assigned admin about a new comment
func (b *Bot) NotifyAssignedAdmin(ticket *tickets.Ticket, comment *tickets.TicketComment) error {
	if ticket.AssignedTo == nil {
		return fmt.Errorf("[ERROR] Ticket not assigned to any admin")
	}

	db, err := database.InitializeDB()
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get database connection: %v", err)
	}

	admin, err := database.GetAdminByID(db, *ticket.AssignedTo)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get admin info: %v", err)
	}

	message := fmt.Sprintf("工单 #%d 有新回复:\n%s", ticket.TicketID, comment.Content)
	return b.SendMessage(admin.TelegramID, message)
}

func (b *Bot) HandleAdminViewTickets(message *tgbotapi.Message) error {
	chatID := message.Chat.ID

	// Check if the user is an admin
	isAdmin, err := database.IsUserAdmin(message.From.ID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to check admin status: %v", err)
	}
	if !isAdmin {
		return b.SendMessage(chatID, "对不起，只有管理员可以使用此命令。")
	}

	db, err := database.InitializeDB()
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get database connection: %v", err)
	}

	allTickets, err := tickets.GetAllTickets(db)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get tickets: %v", err)
	}

	if len(allTickets) == 0 {
		return b.SendMessage(chatID, "目前没有任何工单。")
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup()
	for _, ticket := range allTickets {
		buttonText := fmt.Sprintf("#%d: %s (%s)", ticket.TicketID, ticket.Title, ticket.Status)
		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("view_ticket_%d", ticket.TicketID))
		row := tgbotapi.NewInlineKeyboardRow(button)
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
	}

	return b.SendMessageWithInlineKeyboard(chatID, "所有工单列表：", keyboard)
}
