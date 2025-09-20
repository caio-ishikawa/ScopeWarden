package models

import "fmt"

type UpdateType string

const (
	PortUpdate UpdateType = "PORT"
	URLUpdate  UpdateType = "URL"
	TestUpdate UpdateType = "TEST"
)

type Notification struct {
	TargetName string
	Type       UpdateType
	Content    string
}

type GetMeResponse struct {
	Id       string `json:"id"`
	IsBot    bool   `json:"is_bot"`
	Username string `json:"username"`
}

type NotificationMessage struct {
	ChatID int    `json:"chat_id"`
	Text   string `json:"text"`
}

func (n Notification) CraftMessage() string {
	return fmt.Sprintf("🚨NEW %s FOR %s🚨\nA %s has become available to the public: %s", n.Type, n.TargetName, n.Type, n.Content)
}
