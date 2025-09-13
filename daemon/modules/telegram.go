package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/caio-ishikawa/target-tracker/shared/models"
)

const baseURL = "https://api.telegram.org"

type TelegramClient struct {
	chatID string
	apiKey string
}

func NewTelegramClient() (TelegramClient, error) {
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	if chatID == "" {
		return TelegramClient{}, fmt.Errorf("Failed to create telegram client: No client ID")
	}

	apiKey := os.Getenv("TELEGRAM_API_KEY")
	if apiKey == "" {
		return TelegramClient{}, fmt.Errorf("Failed to create telegram client: No API key")
	}

	return TelegramClient{
		chatID: chatID,
		apiKey: apiKey,
	}, nil
}

func (t TelegramClient) SendMessage(notification models.Notification) error {
	body := models.NotificationMessage{
		ChatID: t.chatID,
		Text:   notification.CraftMessage(),
	}

	req, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("Failed to marshal request body: %w", err)
	}

	res, err := http.Post(fmt.Sprintf("%s/bot%s/sendMessage", baseURL, t.apiKey), "application/json", bytes.NewReader(req))
	if err != nil {
		return fmt.Errorf("Failed to send request to telegram API: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected status code from telegram message API: %v", res.StatusCode)
	}

	return nil
}
