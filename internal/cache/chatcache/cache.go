package chatcache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gitlab.com/siffka/chat-message-mgz/internal/domain"
)

type Cache struct {
	rdb *redis.Client
	ttl time.Duration
}

func New(rdb *redis.Client, ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = 20 * time.Minute
	}
	return &Cache{rdb: rdb, ttl: ttl}
}

func NewRedisClient(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Cache) GetUserChats(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]domain.ChatPreview, bool, error) {
	key := userChatsKey(userID, limit, offset)
	raw, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var result []domain.ChatPreview
	if err = json.Unmarshal(raw, &result); err != nil {
		return nil, false, err
	}
	return result, true, nil
}

func (c *Cache) SetUserChats(ctx context.Context, userID uuid.UUID, limit, offset int32, chats []domain.ChatPreview) error {
	key := userChatsKey(userID, limit, offset)
	return c.setUserKey(ctx, userID, key, chats)
}

func (c *Cache) GetChatPreview(ctx context.Context, userID, chatID uuid.UUID) (domain.ChatPreview, bool, error) {
	key := previewKey(userID, chatID)
	raw, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return domain.ChatPreview{}, false, nil
	}
	if err != nil {
		return domain.ChatPreview{}, false, err
	}
	var result domain.ChatPreview
	if err = json.Unmarshal(raw, &result); err != nil {
		return domain.ChatPreview{}, false, err
	}
	return result, true, nil
}

func (c *Cache) SetChatPreview(ctx context.Context, userID, chatID uuid.UUID, preview domain.ChatPreview) error {
	key := previewKey(userID, chatID)
	return c.setUserKey(ctx, userID, key, preview)
}

func (c *Cache) TouchUser(ctx context.Context, userID uuid.UUID) error {
	setKey := userSetKey(userID)
	keys, err := c.rdb.SMembers(ctx, setKey).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	pipe := c.rdb.Pipeline()
	if len(keys) > 0 {
		for _, key := range keys {
			pipe.Expire(ctx, key, c.ttl)
		}
	}
	pipe.Expire(ctx, setKey, c.ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (c *Cache) InvalidateUser(ctx context.Context, userID uuid.UUID) error {
	setKey := userSetKey(userID)
	keys, err := c.rdb.SMembers(ctx, setKey).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	if len(keys) == 0 {
		return c.rdb.Del(ctx, setKey).Err()
	}
	all := append(keys, setKey)
	return c.rdb.Del(ctx, all...).Err()
}

func (c *Cache) setUserKey(ctx context.Context, userID uuid.UUID, key string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	setKey := userSetKey(userID)
	pipe := c.rdb.Pipeline()
	pipe.Set(ctx, key, raw, c.ttl)
	pipe.SAdd(ctx, setKey, key)
	pipe.Expire(ctx, setKey, c.ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func userSetKey(userID uuid.UUID) string {
	return fmt.Sprintf("chat:userkeys:%s", userID.String())
}

func userChatsKey(userID uuid.UUID, limit, offset int32) string {
	return fmt.Sprintf("chat:list:%s:%d:%d", userID.String(), limit, offset)
}

func previewKey(userID, chatID uuid.UUID) string {
	return fmt.Sprintf("chat:preview:%s:%s", userID.String(), chatID.String())
}
