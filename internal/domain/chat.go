package domain

import (
	"time"

	"github.com/google/uuid"
)

type MessageStatus string

const (
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
)

type Chat struct {
	ID        uuid.UUID
	User1ID   uuid.UUID
	User2ID   uuid.UUID
	CreatedAt time.Time
}

type Message struct {
	ID           int64
	ChatID       uuid.UUID
	SenderUserID uuid.UUID
	Text         string
	Status       MessageStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ChatPreview struct {
	ChatID        uuid.UUID
	OtherUserID   uuid.UUID
	LastMessage   string
	LastMessageAt *time.Time
	LastStatus    *MessageStatus
	UnreadCount   int64
}
