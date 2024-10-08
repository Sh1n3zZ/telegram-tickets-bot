package telegram

import (
	"fmt"
	"log"
	"strings"

	"telegram-tickets-bot/src/config"
	"telegram-tickets-bot/src/database"
	"telegram-tickets-bot/src/tickets"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api *tgbotapi.BotAPI
}

// Initialize Telegram Bot
func NewBot(cfg *config.Config) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.Telegram.BotToken)
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Authorized on account %s", bot.Self.UserName)

	return &Bot{api: bot}, nil
}

// Send text message
func (b *Bot) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := b.api.Send(msg)
	return err
}

// Send photo message
func (b *Bot) SendPhoto(chatID int64, photoPath string, caption string) error {
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(photoPath))
	photo.Caption = caption
	_, err := b.api.Send(photo)
	return err
}

// Send message with inline keyboard
func (b *Bot) SendMessageWithInlineKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	_, err := b.api.Send(msg)
	return err
}

func (b *Bot) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return b.api.GetUpdatesChan(config)
}

func (b *Bot) NotifyAllAdmins(ticket *tickets.Ticket) error {
	db, err := database.InitializeDB()
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get database connection: %v", err)
	}

	var admins []database.AdminUser
	if err := db.Find(&admins).Error; err != nil {
		return fmt.Errorf("[ERROR] Failed to fetch admin users: %v", err)
	}

	for _, admin := range admins {
		message := fmt.Sprintf("新工单已创建:\n工单ID: %d\n标题: %s\n描述: %s",
			ticket.TicketID, ticket.Title, ticket.Description)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("分配工单", fmt.Sprintf("assign_ticket_%d", ticket.TicketID)),
			),
		)

		if err := b.SendMessageWithInlineKeyboard(admin.TelegramID, message, keyboard); err != nil {
			log.Printf("[ERROR] Failed to notify admin %d: %v", admin.AdminID, err)
		}
	}

	return nil
}

func (b *Bot) HandleUpdates(updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		var err error
		if update.Message != nil {
			if update.Message.IsCommand() {
				err = b.HandleCommand(update.Message)
			} else {
				err = b.HandleMessage(update.Message)
			}
		} else if update.CallbackQuery != nil {
			if strings.HasPrefix(update.CallbackQuery.Data, "confirm_") || strings.HasPrefix(update.CallbackQuery.Data, "cancel_") {
				err = b.HandleTicketConfirmation(update.CallbackQuery)
			} else {
				err = b.HandleCallbackQuery(update.CallbackQuery)
			}

			callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
			if _, err := b.api.Request(callback); err != nil {
				log.Printf("[ERROR] Error answering callback query: %v", err)
			}
		}

		if err != nil {
			log.Printf("[ERROR] Error handling update: %v", err)
		}
	}
}
