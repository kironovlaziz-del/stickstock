package delivery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TelegramConfig struct {
	BotToken string
}

const telegramMaxMessageLen = 4000 // Telegram's actual cap is 4096; leave headroom for the "(truncated)" suffix

// SendTelegramMessage posts text to a chat via the Bot API. chatID is
// whatever the report's delivery_target holds — typically a numeric chat
// ID (as a string) or "@channelusername".
func SendTelegramMessage(cfg TelegramConfig, chatID, text string) error {
	if cfg.BotToken == "" {
		return fmt.Errorf("telegram is not configured (TELEGRAM_BOT_TOKEN is empty)")
	}

	if len(text) > telegramMaxMessageLen {
		text = text[:telegramMaxMessageLen] + "\n... (truncated)"
	}

	payload, err := json.Marshal(map[string]string{
		"chat_id": chatID,
		"text":    text,
	})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)
	client := &http.Client{Timeout: 15 * time.Second}

	resp, err := client.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}
	return nil
}
