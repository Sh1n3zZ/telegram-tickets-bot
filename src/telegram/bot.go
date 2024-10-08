package telegram

import (
	"fmt"
	"log"
	"strings"

	"telegram-tickets-bot/src/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api *tgbotapi.BotAPI
}

// 初始化Telegram Bot
func NewBot(cfg *config.Config) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.Telegram.BotToken)
	if err != nil {
		return nil, err
	}

	log.Printf("已授权账号 %s", bot.Self.UserName)

	return &Bot{api: bot}, nil
}

// 发送文本消息
func (b *Bot) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := b.api.Send(msg)
	return err
}

// 发送图片消息
func (b *Bot) SendPhoto(chatID int64, photoPath string, caption string) error {
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(photoPath))
	photo.Caption = caption
	_, err := b.api.Send(photo)
	return err
}

// 发送带有内联键盘的消息
func (b *Bot) SendMessageWithInlineKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	_, err := b.api.Send(msg)
	return err
}

func (b *Bot) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return b.api.GetUpdatesChan(config)
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
				fmt.Printf("回答回调查询时出错: %v\n", err)
			}
		}

		if err != nil {
			fmt.Printf("处理更新时出错: %v\n", err)
		}
	}
}
