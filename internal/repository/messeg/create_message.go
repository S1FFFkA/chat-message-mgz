package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gitlab.com/siffka/chat-message-mgz/internal/domain"
)

func (r *Repository) CreateMessage(ctx context.Context, chatID, userID uuid.UUID, text string) (domain.Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Message{}, fmt.Errorf("begin tx create message: %w", err)
	}
	defer tx.Rollback(ctx)

	var messageID int64
	if err = tx.QueryRow(ctx, `
UPDATE chats
SET last_message_id = last_message_id + 1
WHERE id = $1
RETURNING last_message_id`, chatID).Scan(&messageID); err != nil {
		return domain.Message{}, fmt.Errorf("allocate message_id for chat: %w", err)
	}

	const query = `
INSERT INTO messages (id, chat_id, sender_id, content)
VALUES ($1, $2, $3, $4)
RETURNING id, chat_id, sender_id, content, status, created_at, updated_at`

	var msg domain.Message
	var rawStatus string
	err = tx.QueryRow(ctx, query, messageID, chatID, userID, text).
		Scan(&msg.ID, &msg.ChatID, &msg.SenderUserID, &msg.Text, &rawStatus, &msg.CreatedAt, &msg.UpdatedAt)
	if err != nil {
		return domain.Message{}, fmt.Errorf("create message: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Message{}, fmt.Errorf("commit create message: %w", err)
	}
	msg.Status = domain.MessageStatus(rawStatus)
	return msg, nil
}
