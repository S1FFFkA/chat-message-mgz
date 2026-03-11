package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultMessagesLimit = 50

var ErrInvalidStatusTransition = errors.New("invalid status transition")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}
