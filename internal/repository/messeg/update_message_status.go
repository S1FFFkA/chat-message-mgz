package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gitlab.com/siffka/chat-message-mgz/internal/domain"
)

func (r *Repository) UpdateMessageStatus(ctx context.Context, chatID uuid.UUID, messageID int64, status domain.MessageStatus) (domain.Message, error) {
	const query = `
UPDATE messages
SET status = $2
WHERE chat_id = $1
  AND id = $3
RETURNING id, chat_id, sender_id, content, status, created_at, updated_at`

	var msg domain.Message
	var rawStatus string
	err := r.pool.QueryRow(ctx, query, chatID, string(status), messageID).
		Scan(&msg.ID, &msg.ChatID, &msg.SenderUserID, &msg.Text, &rawStatus, &msg.CreatedAt, &msg.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && strings.Contains(pgErr.Message, "invalid status transition") {
			return domain.Message{}, fmt.Errorf("update message status: %w", ErrInvalidStatusTransition)
		}
		return domain.Message{}, fmt.Errorf("update message status: %w", err)
	}
	msg.Status = domain.MessageStatus(rawStatus)
	return msg, nil
}
