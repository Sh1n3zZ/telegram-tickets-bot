package main

import (
	"log"
	"telegram-tickets-bot/src/config"
	"telegram-tickets-bot/src/database"
	"telegram-tickets-bot/src/telegram"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// Initialize configuration
	cfg, err := config.InitializationConfig()
	if err != nil {
		log.Fatalf("[ERROR] Failed to initialize configuration: %v", err)
	}

	// Initialize database and print connection information
	err = database.InitializeAndPrintDBInfo(&cfg)
	if err != nil {
		log.Fatalf("[ERROR] Failed to initialize database: %v", err)
	}

	// Create Bot instance
	bot, err := telegram.NewBot(&cfg)
	if err != nil {
		log.Fatalf("[ERROR] Failed to create Bot: %v", err)
	}

	// Set update configuration
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Get update channel
	updates := bot.GetUpdatesChan(u)

	// Handle updates
	bot.HandleUpdates(updates)
}
