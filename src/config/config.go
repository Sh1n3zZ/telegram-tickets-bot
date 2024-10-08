package config

import (
	"fmt"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Telegram struct {
		BotToken string `toml:"bot_token"`
	} `toml:"Telegram"`
	Database struct {
		Host     string `toml:"host"`
		Port     int    `toml:"port"`
		User     string `toml:"user"`
		Password string `toml:"password"`
		DBName   string `toml:"dbname"`
	} `toml:"Database"`
}

func InitializationConfig() (Config, error) {
	var config Config
	configPath, err := filepath.Abs("config.toml")
	if err != nil {
		return config, fmt.Errorf("获取配置文件路径失败: %w", err)
	}

	_, err = toml.DecodeFile(configPath, &config)
	if err != nil {
		return config, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if config.Telegram.BotToken == "" {
		return config, fmt.Errorf("Telegram Bot Token 未在配置文件中设置")
	}

	return config, nil
}
