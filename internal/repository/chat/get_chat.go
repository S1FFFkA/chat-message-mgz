package repository

import (
	"context"

	"github.com/S1FFFkA/chat-message-mgz/internal/domain"
	"github.com/google/uuid"
)

func (r *Repository) GetChat(ctx context.Context, chatID uuid.UUID) (domain.Chat, error) {
	const query = `
SELECT id, user1_id, user2_id, created_at
FROM chats
WHERE id = $1`
	var chat domain.Chat
	err := r.pool.QueryRow(ctx, query, chatID).Scan(
		&chat.ID,
		&chat.User1ID,
		&chat.User2ID,
		&chat.CreatedAt,
	)
	return chat, err
}
