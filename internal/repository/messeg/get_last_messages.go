package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"gitlab.com/siffka/chat-message-mgz/internal/domain"
	repocore "gitlab.com/siffka/chat-message-mgz/internal/repository"
)

func (r *Repository) GetLastMessages(ctx context.Context, chatID uuid.UUID, limit int32, beforeMessageID *int64) ([]domain.Message, bool, error) {
	if limit <= 0 {
		limit = defaultMessagesLimit
	}

	var (
		rows pgx.Rows
		err  error
	)

	if beforeMessageID == nil {
		const query = `
SELECT id, chat_id, sender_id, content, status, created_at, updated_at
FROM messages
WHERE chat_id = $1
ORDER BY id DESC
LIMIT $2`
		rows, err = r.pool.Query(ctx, query, chatID, limit+1)
	} else {
		const query = `
WITH boundary AS (
	SELECT id
	FROM messages
	WHERE chat_id = $1
	  AND id = $2
)
SELECT m.id, m.chat_id, m.sender_id, m.content, m.status, m.created_at, m.updated_at
FROM messages m
JOIN boundary b ON TRUE
WHERE m.chat_id = $1
  AND m.id < b.id
ORDER BY m.id DESC
LIMIT $3`
		rows, err = r.pool.Query(ctx, query, chatID, *beforeMessageID, limit+1)
	}
	if err != nil {
		return nil, false, fmt.Errorf("get last messages: %w", err)
	}
	defer rows.Close()

	messages := make([]domain.Message, 0, limit+1)
	for rows.Next() {
		msg, scanErr := repocore.ScanMessage(rows)
		if scanErr != nil {
			return nil, false, scanErr
		}
		messages = append(messages, msg)
	}
	if err = rows.Err(); err != nil {
		return nil, false, fmt.Errorf("get last messages rows: %w", err)
	}

	hasMore := false
	if int32(len(messages)) > limit {
		hasMore = true
		messages = messages[:len(messages)-1]
	}

	return messages, hasMore, nil
}
