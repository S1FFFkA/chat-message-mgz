package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gitlab.com/siffka/chat-message-mgz/internal/domain"
	repocore "gitlab.com/siffka/chat-message-mgz/internal/repository"
)

func (r *Repository) ListUserChats(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]domain.ChatPreview, error) {
	if limit <= 0 {
		limit = 15
	}
	if offset < 0 {
		offset = 0
	}

	const query = `
SELECT
	c.id,
	CASE WHEN c.user1_id = $1 THEN c.user2_id ELSE c.user1_id END AS other_user_id,
	c.last_message,
	c.last_message_at,
	c.last_message_status,
	(
		SELECT COUNT(*)
		FROM messages m
		WHERE m.chat_id = c.id
		  AND m.sender_id != $1
		  AND (cus.last_read_message_id IS NULL OR m.id > rm.id)
	) AS unread_count
FROM chats c
LEFT JOIN chat_user_state cus
       ON cus.chat_id = c.id
      AND cus.user_id = $1
LEFT JOIN messages rm
       ON rm.chat_id = c.id
      AND rm.id = cus.last_read_message_id
WHERE c.user1_id = $1 OR c.user2_id = $1
ORDER BY c.last_message_at DESC NULLS LAST, c.id DESC
LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list user chats: %w", err)
	}
	defer rows.Close()

	result := make([]domain.ChatPreview, 0, limit)
	for rows.Next() {
		preview, scanErr := repocore.ScanChatPreview(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, preview)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("list user chats rows: %w", err)
	}
	return result, nil
}
