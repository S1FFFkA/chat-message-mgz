package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) DeleteChat(ctx context.Context, chatID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM chats WHERE id = $1`, chatID)
	if err != nil {
		return fmt.Errorf("delete chat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
