package repository

import (
	"context"

	"github.com/S1FFFkA/chat-message-mgz/internal/domain"
	"github.com/google/uuid"
)

type ChatRepository interface {
	CreateDirectChat(ctx context.Context, user1ID, user2ID uuid.UUID) (domain.Chat, error)
	GetChat(ctx context.Context, chatID uuid.UUID) (domain.Chat, error)
	DeleteChat(ctx context.Context, chatID uuid.UUID) error
	GetChatPreview(ctx context.Context, chatID, userID uuid.UUID) (domain.ChatPreview, error)
	ListUserChats(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]domain.ChatPreview, error)
}

type MessageRepository interface {
	CreateMessage(ctx context.Context, chatID, userID uuid.UUID, text string) (domain.Message, error)
	DeleteMessage(ctx context.Context, chatID uuid.UUID, messageID int64) error
	UpdateMessageStatus(ctx context.Context, chatID uuid.UUID, messageID int64, status domain.MessageStatus) (domain.Message, error)
	GetLastMessages(ctx context.Context, chatID uuid.UUID, limit int32, beforeMessageID *int64) ([]domain.Message, bool, error)
	MarkChatRead(ctx context.Context, chatID, userID uuid.UUID, upToMessageID *int64) (int64, error)
}
