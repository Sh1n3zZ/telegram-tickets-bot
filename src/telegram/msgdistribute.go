package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func (b *Bot) HandleCommand(message *tgbotapi.Message) error {
	switch message.Command() {
	case "getme":
		return b.HandleGetMeCommand(message)
	case "help", "start":
		return b.HandleHelpCommand(message)
	default:
		return b.SendMessage(message.Chat.ID, "未知命令,请尝试 /help 获取帮助。")
	}
}
