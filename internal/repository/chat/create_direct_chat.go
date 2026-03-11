package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gitlab.com/siffka/chat-message-mgz/internal/domain"
	repocore "gitlab.com/siffka/chat-message-mgz/internal/repository"
)

func (r *Repository) CreateDirectChat(ctx context.Context, user1ID, user2ID uuid.UUID) (domain.Chat, error) {
	user1, user2 := repocore.NormalizeUserPair(user1ID, user2ID)
	chatID, err := repocore.NewUUIDv7()
	if err != nil {
		return domain.Chat{}, fmt.Errorf("generate uuidv7 for chat: %w", err)
	}
	const query = `
INSERT INTO chats (id, user1_id, user2_id)
VALUES ($1, $2, $3)
ON CONFLICT (user1_id, user2_id)
DO UPDATE SET user1_id = EXCLUDED.user1_id
RETURNING id, user1_id, user2_id, created_at`

	var chat domain.Chat
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Chat{}, fmt.Errorf("begin tx create direct chat: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, query, chatID, user1, user2).Scan(&chat.ID, &chat.User1ID, &chat.User2ID, &chat.CreatedAt)
	if err != nil {
		return domain.Chat{}, fmt.Errorf("create direct chat: %w", err)
	}

	const stateQuery = `
INSERT INTO chat_user_state (chat_id, user_id)
VALUES ($1, $2), ($1, $3)
ON CONFLICT (chat_id, user_id) DO NOTHING`

	if _, err = tx.Exec(ctx, stateQuery, chat.ID, chat.User1ID, chat.User2ID); err != nil {
		return domain.Chat{}, fmt.Errorf("init chat user state: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Chat{}, fmt.Errorf("commit create direct chat: %w", err)
	}

	return chat, nil
}
