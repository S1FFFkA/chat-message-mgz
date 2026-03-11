package events

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ChatEventType string

const (
	ChatEventTypeMessageCreated ChatEventType = "message_created"
	ChatEventTypeChatRead       ChatEventType = "chat_read"
)

type ChatUpdatedEvent struct {
	ChatID          uuid.UUID
	MessageID       int64
	UserID          uuid.UUID
	Type            ChatEventType
	UpdatedMessages int64
	OccurredAt      time.Time
}

type Publisher interface {
	PublishChatUpdated(ctx context.Context, event ChatUpdatedEvent) error
}
