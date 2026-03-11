package chatcache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gitlab.com/siffka/chat-message-mgz/internal/domain"
)

func TestCacheSetGetAndInvalidateUser(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := New(rdb, 20*time.Minute)
	ctx := context.Background()

	userID := uuid.New()
	chatID := uuid.New()
	previews := []domain.ChatPreview{{ChatID: chatID, OtherUserID: uuid.New(), UnreadCount: 3}}

	if err := cache.SetUserChats(ctx, userID, 15, 0, previews); err != nil {
		t.Fatalf("SetUserChats: %v", err)
	}
	if err := cache.SetChatPreview(ctx, userID, chatID, previews[0]); err != nil {
		t.Fatalf("SetChatPreview: %v", err)
	}

	gotList, found, err := cache.GetUserChats(ctx, userID, 15, 0)
	if err != nil {
		t.Fatalf("GetUserChats: %v", err)
	}
	if !found || len(gotList) != 1 {
		t.Fatalf("expected cached list")
	}

	gotPreview, found, err := cache.GetChatPreview(ctx, userID, chatID)
	if err != nil {
		t.Fatalf("GetChatPreview: %v", err)
	}
	if !found || gotPreview.ChatID != chatID {
		t.Fatalf("expected cached preview")
	}

	if err := cache.InvalidateUser(ctx, userID); err != nil {
		t.Fatalf("InvalidateUser: %v", err)
	}
	_, found, err = cache.GetUserChats(ctx, userID, 15, 0)
	if err != nil {
		t.Fatalf("GetUserChats after invalidate: %v", err)
	}
	if found {
		t.Fatalf("expected cache miss after invalidate")
	}
}

func TestTouchUserExtendsTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := New(rdb, 20*time.Minute)
	ctx := context.Background()

	userID := uuid.New()
	chatID := uuid.New()
	preview := domain.ChatPreview{ChatID: chatID, OtherUserID: uuid.New(), UnreadCount: 1}

	if err := cache.SetChatPreview(ctx, userID, chatID, preview); err != nil {
		t.Fatalf("SetChatPreview: %v", err)
	}

	key := previewKey(userID, chatID)
	initialTTL := mr.TTL(key)
	if initialTTL <= 0 {
		t.Fatalf("expected key ttl > 0")
	}

	mr.FastForward(10 * time.Minute)
	midTTL := mr.TTL(key)
	if midTTL >= initialTTL {
		t.Fatalf("expected ttl to decrease after time passes")
	}

	if err := cache.TouchUser(ctx, userID); err != nil {
		t.Fatalf("TouchUser: %v", err)
	}
	refreshedTTL := mr.TTL(key)
	if refreshedTTL < 19*time.Minute {
		t.Fatalf("expected ttl refresh close to cache ttl, got %s", refreshedTTL)
	}
}
