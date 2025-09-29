package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/caio-ishikawa/scopewarden/shared/models"
)

const baseURL = "https://api.telegram.org"

type TelegramClient struct {
	chatID int
	apiKey string
}

func NewTelegramClient() (TelegramClient, error) {
	chatIDStr := os.Getenv("SCOPEWARDEN_TELEGRAM_CHAT_ID")
	if chatIDStr == "" {
		return TelegramClient{}, fmt.Errorf("Failed to create telegram client: No client ID")
	}

	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil {
		return TelegramClient{}, fmt.Errorf("Invalid telegram chat ID: %s", chatIDStr)
	}

	apiKey := os.Getenv("SCOPEWARDEN_TELEGRAM_API_KEY")
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

	apiURL := fmt.Sprintf("%s/bot%s/sendMessage", baseURL, t.apiKey)

	res, err := http.Post(apiURL, "application/json", bytes.NewReader(req))
	if err != nil {
		return fmt.Errorf("Failed to send request to telegram API: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected status code from telegram message API: %v", res.StatusCode)
	}

	return nil
}
