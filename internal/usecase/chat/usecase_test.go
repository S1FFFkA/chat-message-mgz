package chat

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"gitlab.com/siffka/chat-message-mgz/internal/cache/chatcache"
	"gitlab.com/siffka/chat-message-mgz/internal/domain"
)

type fakeChatRepo struct {
	createDirectChatFn func(ctx context.Context, user1ID, user2ID uuid.UUID) (domain.Chat, error)
	deleteChatFn       func(ctx context.Context, chatID uuid.UUID) error
	getChatPreviewFn   func(ctx context.Context, chatID, userID uuid.UUID) (domain.ChatPreview, error)
	listUserChatsFn    func(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]domain.ChatPreview, error)
}

func (f *fakeChatRepo) CreateDirectChat(ctx context.Context, user1ID, user2ID uuid.UUID) (domain.Chat, error) {
	return f.createDirectChatFn(ctx, user1ID, user2ID)
}

func (f *fakeChatRepo) DeleteChat(ctx context.Context, chatID uuid.UUID) error {
	return f.deleteChatFn(ctx, chatID)
}

func (f *fakeChatRepo) GetChatPreview(ctx context.Context, chatID, userID uuid.UUID) (domain.ChatPreview, error) {
	return f.getChatPreviewFn(ctx, chatID, userID)
}

func (f *fakeChatRepo) ListUserChats(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]domain.ChatPreview, error) {
	return f.listUserChatsFn(ctx, userID, limit, offset)
}

type fakeMessageRepo struct {
	createMessageFn   func(ctx context.Context, chatID, userID uuid.UUID, text string) (domain.Message, error)
	deleteMessageFn   func(ctx context.Context, chatID uuid.UUID, messageID int64) error
	updateStatusFn    func(ctx context.Context, chatID uuid.UUID, messageID int64, status domain.MessageStatus) (domain.Message, error)
	getLastMessagesFn func(ctx context.Context, chatID uuid.UUID, limit int32, beforeMessageID *int64) ([]domain.Message, bool, error)
	markChatReadFn    func(ctx context.Context, chatID, userID uuid.UUID, upToMessageID *int64) (int64, error)
}

func (f *fakeMessageRepo) CreateMessage(ctx context.Context, chatID, userID uuid.UUID, text string) (domain.Message, error) {
	return f.createMessageFn(ctx, chatID, userID, text)
}

func (f *fakeMessageRepo) DeleteMessage(ctx context.Context, chatID uuid.UUID, messageID int64) error {
	return f.deleteMessageFn(ctx, chatID, messageID)
}

func (f *fakeMessageRepo) UpdateMessageStatus(ctx context.Context, chatID uuid.UUID, messageID int64, status domain.MessageStatus) (domain.Message, error) {
	return f.updateStatusFn(ctx, chatID, messageID, status)
}

func (f *fakeMessageRepo) GetLastMessages(ctx context.Context, chatID uuid.UUID, limit int32, beforeMessageID *int64) ([]domain.Message, bool, error) {
	return f.getLastMessagesFn(ctx, chatID, limit, beforeMessageID)
}

func (f *fakeMessageRepo) MarkChatRead(ctx context.Context, chatID, userID uuid.UUID, upToMessageID *int64) (int64, error) {
	return f.markChatReadFn(ctx, chatID, userID, upToMessageID)
}

func newServiceForTest() *Service {
	return NewService(
		&fakeChatRepo{
			createDirectChatFn: func(context.Context, uuid.UUID, uuid.UUID) (domain.Chat, error) { return domain.Chat{}, nil },
			deleteChatFn:       func(context.Context, uuid.UUID) error { return nil },
			getChatPreviewFn: func(context.Context, uuid.UUID, uuid.UUID) (domain.ChatPreview, error) {
				return domain.ChatPreview{}, nil
			},
			listUserChatsFn: func(context.Context, uuid.UUID, int32, int32) ([]domain.ChatPreview, error) { return nil, nil },
		},
		&fakeMessageRepo{
			createMessageFn: func(context.Context, uuid.UUID, uuid.UUID, string) (domain.Message, error) {
				return domain.Message{}, nil
			},
			deleteMessageFn: func(context.Context, uuid.UUID, int64) error { return nil },
			updateStatusFn: func(context.Context, uuid.UUID, int64, domain.MessageStatus) (domain.Message, error) {
				return domain.Message{}, nil
			},
			getLastMessagesFn: func(context.Context, uuid.UUID, int32, *int64) ([]domain.Message, bool, error) {
				return nil, false, nil
			},
			markChatReadFn: func(context.Context, uuid.UUID, uuid.UUID, *int64) (int64, error) { return 0, nil },
		},
	)
}

func TestCreateDirectChatValidation(t *testing.T) {
	svc := newServiceForTest()
	_, err := svc.CreateDirectChat(context.Background(), uuid.Nil, uuid.New())
	if !domain.IsErrorCode(err, domain.ErrorCodeInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}

	id := uuid.New()
	_, err = svc.CreateDirectChat(context.Background(), id, id)
	if !domain.IsErrorCode(err, domain.ErrorCodeInvalidArgument) {
		t.Fatalf("expected invalid argument for self chat, got %v", err)
	}
}

func TestCreateMessageValidationAndTrim(t *testing.T) {
	var gotText string
	svc := NewService(
		&fakeChatRepo{
			createDirectChatFn: func(context.Context, uuid.UUID, uuid.UUID) (domain.Chat, error) { return domain.Chat{}, nil },
			deleteChatFn:       func(context.Context, uuid.UUID) error { return nil },
			getChatPreviewFn: func(context.Context, uuid.UUID, uuid.UUID) (domain.ChatPreview, error) {
				return domain.ChatPreview{}, nil
			},
			listUserChatsFn: func(context.Context, uuid.UUID, int32, int32) ([]domain.ChatPreview, error) { return nil, nil },
		},
		&fakeMessageRepo{
			createMessageFn: func(ctx context.Context, chatID, userID uuid.UUID, text string) (domain.Message, error) {
				gotText = text
				return domain.Message{Text: text}, nil
			},
			deleteMessageFn: func(context.Context, uuid.UUID, int64) error { return nil },
			updateStatusFn: func(context.Context, uuid.UUID, int64, domain.MessageStatus) (domain.Message, error) {
				return domain.Message{}, nil
			},
			getLastMessagesFn: func(context.Context, uuid.UUID, int32, *int64) ([]domain.Message, bool, error) {
				return nil, false, nil
			},
			markChatReadFn: func(context.Context, uuid.UUID, uuid.UUID, *int64) (int64, error) { return 0, nil },
		},
	)

	chatID := uuid.New()
	userID := uuid.New()
	_, err := svc.CreateMessage(context.Background(), chatID, userID, "   ")
	if !domain.IsErrorCode(err, domain.ErrorCodeInvalidArgument) {
		t.Fatalf("expected invalid argument for empty message, got %v", err)
	}

	_, err = svc.CreateMessage(context.Background(), chatID, userID, strings.Repeat("a", maxMessageTextLen+1))
	if !domain.IsErrorCode(err, domain.ErrorCodeInvalidArgument) {
		t.Fatalf("expected invalid argument for long message, got %v", err)
	}

	_, err = svc.CreateMessage(context.Background(), chatID, userID, "  hi  ")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gotText != "hi" {
		t.Fatalf("expected trimmed message, got %q", gotText)
	}
}

func TestRepoNoRowsMappedToNotFound(t *testing.T) {
	svc := NewService(
		&fakeChatRepo{
			createDirectChatFn: func(context.Context, uuid.UUID, uuid.UUID) (domain.Chat, error) { return domain.Chat{}, nil },
			deleteChatFn:       func(context.Context, uuid.UUID) error { return pgx.ErrNoRows },
			getChatPreviewFn: func(context.Context, uuid.UUID, uuid.UUID) (domain.ChatPreview, error) {
				return domain.ChatPreview{}, pgx.ErrNoRows
			},
			listUserChatsFn: func(context.Context, uuid.UUID, int32, int32) ([]domain.ChatPreview, error) { return nil, nil },
		},
		&fakeMessageRepo{
			createMessageFn: func(context.Context, uuid.UUID, uuid.UUID, string) (domain.Message, error) {
				return domain.Message{}, nil
			},
			deleteMessageFn: func(context.Context, uuid.UUID, int64) error { return pgx.ErrNoRows },
			updateStatusFn: func(context.Context, uuid.UUID, int64, domain.MessageStatus) (domain.Message, error) {
				return domain.Message{}, pgx.ErrNoRows
			},
			getLastMessagesFn: func(context.Context, uuid.UUID, int32, *int64) ([]domain.Message, bool, error) {
				return nil, false, pgx.ErrNoRows
			},
			markChatReadFn: func(context.Context, uuid.UUID, uuid.UUID, *int64) (int64, error) { return 0, pgx.ErrNoRows },
		},
	)

	chatID := uuid.New()
	userID := uuid.New()

	if err := svc.DeleteChat(context.Background(), chatID); !domain.IsErrorCode(err, domain.ErrorCodeNotFound) {
		t.Fatalf("DeleteChat expected not_found, got %v", err)
	}
	if _, err := svc.GetChatPreview(context.Background(), chatID, userID); !domain.IsErrorCode(err, domain.ErrorCodeNotFound) {
		t.Fatalf("GetChatPreview expected not_found, got %v", err)
	}
	if err := svc.DeleteMessage(context.Background(), chatID, 1); !domain.IsErrorCode(err, domain.ErrorCodeNotFound) {
		t.Fatalf("DeleteMessage expected not_found, got %v", err)
	}
	if _, err := svc.UpdateMessageStatus(context.Background(), chatID, 1, domain.MessageStatusSent); !domain.IsErrorCode(err, domain.ErrorCodeNotFound) {
		t.Fatalf("UpdateMessageStatus expected not_found, got %v", err)
	}
	if _, _, err := svc.GetLastMessages(context.Background(), chatID, 1, nil); !domain.IsErrorCode(err, domain.ErrorCodeNotFound) {
		t.Fatalf("GetLastMessages expected not_found, got %v", err)
	}
	if _, err := svc.MarkChatRead(context.Background(), chatID, userID, nil); !domain.IsErrorCode(err, domain.ErrorCodeNotFound) {
		t.Fatalf("MarkChatRead expected not_found, got %v", err)
	}
}

func TestPaginationDefaultsAndLimits(t *testing.T) {
	var gotLimit int32
	svc := NewService(
		&fakeChatRepo{
			createDirectChatFn: func(context.Context, uuid.UUID, uuid.UUID) (domain.Chat, error) { return domain.Chat{}, nil },
			deleteChatFn:       func(context.Context, uuid.UUID) error { return nil },
			getChatPreviewFn: func(context.Context, uuid.UUID, uuid.UUID) (domain.ChatPreview, error) {
				return domain.ChatPreview{}, nil
			},
			listUserChatsFn: func(_ context.Context, _ uuid.UUID, limit, _ int32) ([]domain.ChatPreview, error) {
				gotLimit = limit
				return []domain.ChatPreview{}, nil
			},
		},
		&fakeMessageRepo{
			createMessageFn: func(context.Context, uuid.UUID, uuid.UUID, string) (domain.Message, error) {
				return domain.Message{}, nil
			},
			deleteMessageFn: func(context.Context, uuid.UUID, int64) error { return nil },
			updateStatusFn: func(context.Context, uuid.UUID, int64, domain.MessageStatus) (domain.Message, error) {
				return domain.Message{}, nil
			},
			getLastMessagesFn: func(_ context.Context, _ uuid.UUID, limit int32, _ *int64) ([]domain.Message, bool, error) {
				gotLimit = limit
				return []domain.Message{}, false, nil
			},
			markChatReadFn: func(context.Context, uuid.UUID, uuid.UUID, *int64) (int64, error) { return 0, nil },
		},
	)

	_, _, err := svc.GetLastMessages(context.Background(), uuid.New(), 0, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gotLimit != defaultMessagesLimit {
		t.Fatalf("expected default messages limit %d got %d", defaultMessagesLimit, gotLimit)
	}

	_, _, err = svc.GetLastMessages(context.Background(), uuid.New(), 1000, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gotLimit != maxMessagesLimit {
		t.Fatalf("expected max messages limit %d got %d", maxMessagesLimit, gotLimit)
	}

	_, err = svc.ListUserChats(context.Background(), uuid.New(), 0, -10)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gotLimit != defaultChatsLimit {
		t.Fatalf("expected default chats limit %d got %d", defaultChatsLimit, gotLimit)
	}
}

func TestUsecaseInputValidation(t *testing.T) {
	svc := newServiceForTest()
	ctx := context.Background()

	if err := svc.DeleteChat(ctx, uuid.Nil); !domain.IsErrorCode(err, domain.ErrorCodeInvalidArgument) {
		t.Fatalf("DeleteChat expected invalid_argument, got %v", err)
	}
	if err := svc.DeleteMessage(ctx, uuid.New(), 0); !domain.IsErrorCode(err, domain.ErrorCodeInvalidArgument) {
		t.Fatalf("DeleteMessage expected invalid_argument, got %v", err)
	}
	if _, err := svc.UpdateMessageStatus(ctx, uuid.New(), 1, domain.MessageStatus("bad")); !domain.IsErrorCode(err, domain.ErrorCodeInvalidArgument) {
		t.Fatalf("UpdateMessageStatus expected invalid_argument, got %v", err)
	}
	if _, _, err := svc.GetLastMessages(ctx, uuid.Nil, 1, nil); !domain.IsErrorCode(err, domain.ErrorCodeInvalidArgument) {
		t.Fatalf("GetLastMessages expected invalid_argument for nil chat")
	}
	zero := int64(0)
	if _, _, err := svc.GetLastMessages(ctx, uuid.New(), 1, &zero); !domain.IsErrorCode(err, domain.ErrorCodeInvalidArgument) {
		t.Fatalf("GetLastMessages expected invalid_argument for bad before")
	}
	if _, err := svc.ListUserChats(ctx, uuid.Nil, 1, 0); !domain.IsErrorCode(err, domain.ErrorCodeInvalidArgument) {
		t.Fatalf("ListUserChats expected invalid_argument, got %v", err)
	}
	if _, err := svc.MarkChatRead(ctx, uuid.New(), uuid.New(), &zero); !domain.IsErrorCode(err, domain.ErrorCodeInvalidArgument) {
		t.Fatalf("MarkChatRead expected invalid_argument for up_to")
	}
}

func TestMapRepoErrorBranches(t *testing.T) {
	if mapRepoError(nil, "x") != nil {
		t.Fatalf("nil should stay nil")
	}
	if !domain.IsErrorCode(mapRepoError(pgx.ErrNoRows, "x"), domain.ErrorCodeNotFound) {
		t.Fatalf("ErrNoRows should map to not_found")
	}
	conflict := domain.ConflictError("c", errors.New("c"))
	if !errors.Is(mapRepoError(conflict, "x"), conflict) {
		t.Fatalf("domain app error should pass through")
	}
	if !domain.IsErrorCode(mapRepoError(errors.New("boom"), "x"), domain.ErrorCodeInternal) {
		t.Fatalf("unknown should map to internal")
	}
}

func TestGetChatPreviewCacheHit(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := chatcache.New(rdb, 20*time.Minute)

	userID := uuid.New()
	chatID := uuid.New()
	expected := domain.ChatPreview{ChatID: chatID, OtherUserID: uuid.New(), UnreadCount: 5}
	if err := cache.SetChatPreview(context.Background(), userID, chatID, expected); err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	repoCalled := false
	svc := NewServiceWithCache(
		&fakeChatRepo{
			createDirectChatFn: func(context.Context, uuid.UUID, uuid.UUID) (domain.Chat, error) { return domain.Chat{}, nil },
			deleteChatFn:       func(context.Context, uuid.UUID) error { return nil },
			getChatPreviewFn: func(context.Context, uuid.UUID, uuid.UUID) (domain.ChatPreview, error) {
				repoCalled = true
				return domain.ChatPreview{}, nil
			},
			listUserChatsFn: func(context.Context, uuid.UUID, int32, int32) ([]domain.ChatPreview, error) { return nil, nil },
		},
		&fakeMessageRepo{
			createMessageFn: func(context.Context, uuid.UUID, uuid.UUID, string) (domain.Message, error) {
				return domain.Message{}, nil
			},
			deleteMessageFn: func(context.Context, uuid.UUID, int64) error { return nil },
			updateStatusFn: func(context.Context, uuid.UUID, int64, domain.MessageStatus) (domain.Message, error) {
				return domain.Message{}, nil
			},
			getLastMessagesFn: func(context.Context, uuid.UUID, int32, *int64) ([]domain.Message, bool, error) {
				return nil, false, nil
			},
			markChatReadFn: func(context.Context, uuid.UUID, uuid.UUID, *int64) (int64, error) { return 0, nil },
		},
		cache,
	)

	got, err := svc.GetChatPreview(context.Background(), chatID, userID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.ChatID != expected.ChatID || repoCalled {
		t.Fatalf("expected cache hit without repo call")
	}
}

func TestListUserChatsCacheMissThenHit(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := chatcache.New(rdb, 20*time.Minute)

	calls := 0
	userID := uuid.New()
	expected := []domain.ChatPreview{{ChatID: uuid.New(), OtherUserID: uuid.New(), UnreadCount: 1}}

	svc := NewServiceWithCache(
		&fakeChatRepo{
			createDirectChatFn: func(context.Context, uuid.UUID, uuid.UUID) (domain.Chat, error) { return domain.Chat{}, nil },
			deleteChatFn:       func(context.Context, uuid.UUID) error { return nil },
			getChatPreviewFn: func(context.Context, uuid.UUID, uuid.UUID) (domain.ChatPreview, error) {
				return domain.ChatPreview{}, nil
			},
			listUserChatsFn: func(context.Context, uuid.UUID, int32, int32) ([]domain.ChatPreview, error) {
				calls++
				return expected, nil
			},
		},
		&fakeMessageRepo{
			createMessageFn: func(context.Context, uuid.UUID, uuid.UUID, string) (domain.Message, error) {
				return domain.Message{}, nil
			},
			deleteMessageFn: func(context.Context, uuid.UUID, int64) error { return nil },
			updateStatusFn: func(context.Context, uuid.UUID, int64, domain.MessageStatus) (domain.Message, error) {
				return domain.Message{}, nil
			},
			getLastMessagesFn: func(context.Context, uuid.UUID, int32, *int64) ([]domain.Message, bool, error) {
				return nil, false, nil
			},
			markChatReadFn: func(context.Context, uuid.UUID, uuid.UUID, *int64) (int64, error) { return 0, nil },
		},
		cache,
	)

	_, err := svc.ListUserChats(context.Background(), userID, 15, 0)
	if err != nil {
		t.Fatalf("first call err: %v", err)
	}
	_, err = svc.ListUserChats(context.Background(), userID, 15, 0)
	if err != nil {
		t.Fatalf("second call err: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected repo to be called once, got %d", calls)
	}
}
