package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) MarkChatRead(ctx context.Context, chatID, userID uuid.UUID, upToMessageID *int64) (int64, error) {
	const query = `
WITH current_state AS (
	SELECT
		cus.last_read_message_id AS last_read_id
	FROM chat_user_state cus
	WHERE cus.chat_id = $1
	  AND cus.user_id = $2
),
target AS (
	SELECT id
	FROM messages
	WHERE chat_id = $1
	  AND ($3::bigint IS NULL OR id = $3::bigint)
	ORDER BY id DESC
	LIMIT 1
),
to_read AS (
	SELECT COUNT(*)::bigint AS cnt
	FROM messages m
	LEFT JOIN current_state s ON TRUE
	JOIN target t ON TRUE
	WHERE m.chat_id = $1
	  AND m.sender_id <> $2
	  AND m.id <= t.id
	  AND (s.last_read_id IS NULL OR m.id > s.last_read_id)
),
upsert_state AS (
	INSERT INTO chat_user_state (chat_id, user_id, last_read_message_id, updated_at)
	SELECT $1, $2, t.id::bigint, NOW()
	FROM target t
	ON CONFLICT (chat_id, user_id)
	DO UPDATE
	   SET last_read_message_id = EXCLUDED.last_read_message_id,
	       updated_at = NOW()
	RETURNING 1
)
SELECT COALESCE((SELECT cnt FROM to_read), 0)`

	var boundary any
	if upToMessageID != nil {
		boundary = *upToMessageID
	}

	var updated int64
	if err := r.pool.QueryRow(ctx, query, chatID, userID, boundary).Scan(&updated); err != nil {
		return 0, fmt.Errorf("mark chat read: %w", err)
	}
	return updated, nil
}
