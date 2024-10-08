package telegram

import (
	"fmt"
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
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("创建工单", "create_ticket"),
			tgbotapi.NewInlineKeyboardButtonData("查看我的工单", "view_tickets"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("获取个人信息", "get_info"),
		),
	)

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
		return b.SendMessage(chatID, "Please enter the ticket title:")
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
	default:
		return b.SendMessage(chatID, "Unknown option.")
	}
}

func (b *Bot) HandleMessage(message *tgbotapi.Message) error {
	chatID := message.Chat.ID
	text := message.Text

	switch userStates[chatID] {
	case "waiting_for_title":
		ticketData[chatID].Title = text
		userStates[chatID] = "waiting_for_description"
		return b.SendMessage(chatID, "请输入工单描述：")
	case "waiting_for_description":
		ticketData[chatID].Description = text
		return b.ConfirmTicketCreation(chatID)
	case StateWaitingForComment:
		return b.AddCommentToTicket(chatID, message.From.ID, text)
	default:
		return b.SendMessage(chatID, "我不明白您的意思。请使用 /help 查看可用命令。")
	}
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

	delete(userStates, chatID)
	delete(ticketData, chatID)

	successMsg := fmt.Sprintf("工单创建成功。工单ID: %d", ticket.TicketID)
	err = b.SendMessage(chatID, successMsg)
	if err != nil {
		return err
	}

	// 显示新创建的工单详情
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
	chatID := callbackQuery.Message.Chat.ID
	data := callbackQuery.Data

	// Extract ticket ID from callback data
	var ticketID int
	_, err := fmt.Sscanf(data, "view_ticket_%d", &ticketID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to parse ticket ID: %v", err)
	}

	db, err := database.InitializeDB()
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get database connection: %v", err)
	}

	ticket, err := tickets.GetTicketByID(db, ticketID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get ticket information: %v", err)
	}

	ticketInfo := fmt.Sprintf("工单 #%d\n标题: %s\n描述: %s\n状态: %s\n优先级: %s\n创建时间: %s",
		ticket.TicketID, ticket.Title, ticket.Description, ticket.Status, ticket.Priority, ticket.CreatedAt.Format("2006-01-02 15:04:05"))

	var keyboard tgbotapi.InlineKeyboardMarkup
	if ticket.Status == "closed" {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
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

	// Get ticket comments
	comments, err := tickets.GetTicketComments(db, ticketID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get ticket comments: %v", err)
	}

	// Add comments to ticket information
	for _, comment := range comments {
		ticketInfo += fmt.Sprintf("\n\n评论 (ID: %d):\n%s\n时间: %s", comment.CommentID, comment.Content, comment.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	return b.SendMessageWithInlineKeyboard(chatID, ticketInfo, keyboard)
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

// Add new function to handle adding comments
func (b *Bot) AddCommentToTicket(chatID int64, telegramUserID int64, content string) error {
	data := ticketData[chatID]

	db, err := database.InitializeDB()
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get database connection: %v", err)
	}

	// Get user_id from database
	userID, err := database.GetUserIDByTelegramID(db, telegramUserID)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get user ID: %v", err)
	}

	err = tickets.AddComment(db, data.TicketID, userID, content)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to add comment: %v", err)
	}

	delete(userStates, chatID)
	delete(ticketData, chatID)

	// Display ticket information again
	return b.HandleTicketView(&tgbotapi.CallbackQuery{
		Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}},
		Data:    fmt.Sprintf("view_ticket_%d", data.TicketID),
	})
}
