package main

import (
	"log"
	"telegram-tickets-bot/src/config"
	"telegram-tickets-bot/src/database"
	"telegram-tickets-bot/src/telegram"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// 初始化配置
	cfg, err := config.InitializationConfig()
	if err != nil {
		log.Fatalf("初始化配置失败: %v", err)
	}

	// 初始化数据库并打印连接信息
	err = database.InitializeAndPrintDBInfo(&cfg)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 创建 Bot 实例
	bot, err := telegram.NewBot(&cfg)
	if err != nil {
		log.Fatalf("创建 Bot 失败: %v", err)
	}

	// 设置更新配置
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// 获取更新通道
	updates := bot.GetUpdatesChan(u)

	// 处理更新
	bot.HandleUpdates(updates)
}
