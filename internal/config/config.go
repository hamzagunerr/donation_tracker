package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken   string
	AdminChatID     int64
	CalendarAdminID int64
	DatabaseURL     string
}

func Load() (*Config, error) {
	// .env dosyası varsa yükle, yoksa environment variable'lardan oku (Docker için)
	_ = godotenv.Load()

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN tanımlanmamış")
	}

	adminChatIDStr := os.Getenv("ADMIN_CHAT_ID")
	if adminChatIDStr == "" {
		return nil, fmt.Errorf("ADMIN_CHAT_ID tanımlanmamış")
	}

	adminChatID, err := strconv.ParseInt(adminChatIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("ADMIN_CHAT_ID geçersiz: %w", err)
	}

	// Takvim admin'i opsiyonel
	var calendarAdminID int64
	calendarAdminIDStr := os.Getenv("CALENDAR_ADMIN_ID")
	if calendarAdminIDStr != "" {
		calendarAdminID, err = strconv.ParseInt(calendarAdminIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("CALENDAR_ADMIN_ID geçersiz: %w", err)
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL tanımlanmamış")
	}

	return &Config{
		TelegramToken:   token,
		AdminChatID:     adminChatID,
		CalendarAdminID: calendarAdminID,
		DatabaseURL:     dbURL,
	}, nil
}
