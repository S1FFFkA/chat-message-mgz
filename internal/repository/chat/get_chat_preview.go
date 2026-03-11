package repository

import (
	"context"

	"github.com/google/uuid"
	"gitlab.com/siffka/chat-message-mgz/internal/domain"
	repocore "gitlab.com/siffka/chat-message-mgz/internal/repository"
)

func (r *Repository) GetChatPreview(ctx context.Context, chatID, userID uuid.UUID) (domain.ChatPreview, error) {
	const query = `
SELECT
	c.id,
	CASE WHEN c.user1_id = $2 THEN c.user2_id ELSE c.user1_id END AS other_user_id,
	c.last_message,
	c.last_message_at,
	c.last_message_status,
	(
		SELECT COUNT(*)
		FROM messages m
		WHERE m.chat_id = c.id
		  AND m.sender_id != $2
		  AND (cus.last_read_message_id IS NULL OR m.id > rm.id) -- смотрим заходил ли человек в чат вообще или сравниваем последнее непрочитанное с послденим непрочитанным
	) AS unread_count
FROM chats c
LEFT JOIN chat_user_state cus
       ON cus.chat_id = c.id
      AND cus.user_id = $2
LEFT JOIN messages rm
       ON rm.chat_id = c.id
      AND rm.id = cus.last_read_message_id
WHERE c.id = $1
  AND ($2 = c.user1_id OR $2 = c.user2_id)`

	return repocore.ScanChatPreview(r.pool.QueryRow(ctx, query, chatID, userID))
}
