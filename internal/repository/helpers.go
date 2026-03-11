package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gitlab.com/siffka/chat-message-mgz/internal/domain"
)

type RowScanner interface {
	Scan(dest ...any) error
}

func NormalizeUserPair(a, b uuid.UUID) (uuid.UUID, uuid.UUID) {
	if strings.Compare(a.String(), b.String()) < 0 {
		return a, b
	}
	return b, a
}

func NewUUIDv7() (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func ScanMessage(scanner RowScanner) (domain.Message, error) {
	var (
		msg       domain.Message
		rawStatus string
	)
	err := scanner.Scan(
		&msg.ID,
		&msg.ChatID,
		&msg.SenderUserID,
		&msg.Text,
		&rawStatus,
		&msg.CreatedAt,
		&msg.UpdatedAt,
	)
	if err != nil {
		return domain.Message{}, fmt.Errorf("scan message: %w", err)
	}
	msg.Status = domain.MessageStatus(rawStatus)
	return msg, nil
}

func ScanChatPreview(scanner RowScanner) (domain.ChatPreview, error) {
	var (
		preview     domain.ChatPreview
		lastMessage *string
		lastAt      *time.Time
		lastStatus  *string
		unreadCount int64
	)
	err := scanner.Scan(
		&preview.ChatID,
		&preview.OtherUserID,
		&lastMessage,
		&lastAt,
		&lastStatus,
		&unreadCount,
	)
	if err != nil {
		return domain.ChatPreview{}, fmt.Errorf("scan chat preview: %w", err)
	}
	if lastMessage != nil {
		preview.LastMessage = *lastMessage
	}
	if lastAt != nil {
		preview.LastMessageAt = lastAt
	}
	if lastStatus != nil {
		status := domain.MessageStatus(*lastStatus)
		preview.LastStatus = &status
	}
	preview.UnreadCount = unreadCount
	return preview, nil
}
