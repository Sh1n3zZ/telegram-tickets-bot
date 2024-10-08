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
		return config, fmt.Errorf("[ERROR] Failed to get config file path: %w", err)
	}

	_, err = toml.DecodeFile(configPath, &config)
	if err != nil {
		return config, fmt.Errorf("[ERROR] Failed to parse config file: %w", err)
	}

	if config.Telegram.BotToken == "" {
		return config, fmt.Errorf("[ERROR] Telegram Bot Token not set in config file")
	}

	return config, nil
}
