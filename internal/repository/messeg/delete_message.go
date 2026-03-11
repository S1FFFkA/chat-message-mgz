package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) DeleteMessage(ctx context.Context, chatID uuid.UUID, messageID int64) error {
	tag, err := r.pool.Exec(ctx, `
DELETE FROM messages
WHERE chat_id = $1
  AND id = $2`, chatID, messageID)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
