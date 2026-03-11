package chat

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"gitlab.com/siffka/chat-message-mgz/internal/cache/chatcache"
	"gitlab.com/siffka/chat-message-mgz/internal/domain"
	"gitlab.com/siffka/chat-message-mgz/internal/repository"
)

const (
	defaultMessagesLimit = 50
	maxMessagesLimit     = 100

	defaultChatsLimit = 15
	maxChatsLimit     = 100

	maxMessageTextLen = 4000
)

var (
	ErrInvalidArgument = domain.InvalidArgumentError("invalid argument", nil)
	ErrSelfChat        = domain.InvalidArgumentError("direct chat with self is not allowed", nil)
	ErrEmptyMessage    = domain.InvalidArgumentError("message text is empty", nil)
	ErrMessageTooLong  = domain.InvalidArgumentError("message text is too long", nil)
)

type UseCase interface {
	CreateDirectChat(ctx context.Context, user1ID, user2ID uuid.UUID) (domain.Chat, error)
	DeleteChat(ctx context.Context, chatID uuid.UUID) error

	CreateMessage(ctx context.Context, chatID, userID uuid.UUID, text string) (domain.Message, error)
	DeleteMessage(ctx context.Context, chatID uuid.UUID, messageID int64) error
	UpdateMessageStatus(ctx context.Context, chatID uuid.UUID, messageID int64, status domain.MessageStatus) (domain.Message, error)

	GetChatPreview(ctx context.Context, chatID, userID uuid.UUID) (domain.ChatPreview, error)
	GetLastMessages(ctx context.Context, chatID uuid.UUID, limit int32, beforeMessageID *int64) ([]domain.Message, bool, error)
	ListUserChats(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]domain.ChatPreview, error)
	MarkChatRead(ctx context.Context, chatID, userID uuid.UUID, upToMessageID *int64) (int64, error)
}

type Service struct {
	chatRepo    repository.ChatRepository
	messageRepo repository.MessageRepository
	cache       *chatcache.Cache
}

func NewService(chatRepo repository.ChatRepository, messageRepo repository.MessageRepository) *Service {
	return &Service{
		chatRepo:    chatRepo,
		messageRepo: messageRepo,
	}
}

func NewServiceWithCache(chatRepo repository.ChatRepository, messageRepo repository.MessageRepository, cache *chatcache.Cache) *Service {
	return &Service{
		chatRepo:    chatRepo,
		messageRepo: messageRepo,
		cache:       cache,
	}
}

func (s *Service) CreateDirectChat(ctx context.Context, user1ID, user2ID uuid.UUID) (domain.Chat, error) {
	if user1ID == uuid.Nil || user2ID == uuid.Nil {
		return domain.Chat{}, ErrInvalidArgument
	}
	if user1ID == user2ID {
		return domain.Chat{}, ErrSelfChat
	}
	chat, err := s.chatRepo.CreateDirectChat(ctx, user1ID, user2ID)
	if err != nil {
		return domain.Chat{}, mapRepoError(err, "chat not found")
	}
	s.invalidateUsers(ctx, user1ID, user2ID)
	s.touchUsers(ctx, user1ID, user2ID)
	return chat, nil
}

func (s *Service) DeleteChat(ctx context.Context, chatID uuid.UUID) error {
	if chatID == uuid.Nil {
		return ErrInvalidArgument
	}
	return mapRepoError(s.chatRepo.DeleteChat(ctx, chatID), "chat not found")
}

func (s *Service) CreateMessage(ctx context.Context, chatID, userID uuid.UUID, text string) (domain.Message, error) {
	if chatID == uuid.Nil || userID == uuid.Nil {
		return domain.Message{}, ErrInvalidArgument
	}

	normalizedText := strings.TrimSpace(text)
	if normalizedText == "" {
		return domain.Message{}, ErrEmptyMessage
	}
	if len(normalizedText) > maxMessageTextLen {
		return domain.Message{}, ErrMessageTooLong
	}

	msg, err := s.messageRepo.CreateMessage(ctx, chatID, userID, normalizedText)
	if err != nil {
		return domain.Message{}, mapRepoError(err, "chat not found")
	}
	s.invalidateUsers(ctx, userID)
	s.touchUsers(ctx, userID)
	return msg, nil
}

func (s *Service) DeleteMessage(ctx context.Context, chatID uuid.UUID, messageID int64) error {
	if chatID == uuid.Nil || messageID <= 0 {
		return ErrInvalidArgument
	}
	return mapRepoError(s.messageRepo.DeleteMessage(ctx, chatID, messageID), "message not found")
}

func (s *Service) UpdateMessageStatus(ctx context.Context, chatID uuid.UUID, messageID int64, status domain.MessageStatus) (domain.Message, error) {
	if chatID == uuid.Nil || messageID <= 0 {
		return domain.Message{}, ErrInvalidArgument
	}
	if status != domain.MessageStatusSent && status != domain.MessageStatusDelivered && status != domain.MessageStatusRead {
		return domain.Message{}, ErrInvalidArgument
	}
	msg, err := s.messageRepo.UpdateMessageStatus(ctx, chatID, messageID, status)
	if err != nil {
		return domain.Message{}, mapRepoError(err, "message not found")
	}
	return msg, nil
}

func (s *Service) GetChatPreview(ctx context.Context, chatID, userID uuid.UUID) (domain.ChatPreview, error) {
	if chatID == uuid.Nil || userID == uuid.Nil {
		return domain.ChatPreview{}, ErrInvalidArgument
	}
	s.touchUsers(ctx, userID)
	if s.cache != nil {
		cached, found, err := s.cache.GetChatPreview(ctx, userID, chatID)
		if err == nil && found {
			return cached, nil
		}
	}
	preview, err := s.chatRepo.GetChatPreview(ctx, chatID, userID)
	if err != nil {
		return domain.ChatPreview{}, mapRepoError(err, "chat not found")
	}
	if s.cache != nil {
		_ = s.cache.SetChatPreview(ctx, userID, chatID, preview)
	}
	return preview, nil
}

func (s *Service) GetLastMessages(ctx context.Context, chatID uuid.UUID, limit int32, beforeMessageID *int64) ([]domain.Message, bool, error) {
	if chatID == uuid.Nil {
		return nil, false, ErrInvalidArgument
	}
	if beforeMessageID != nil && *beforeMessageID <= 0 {
		return nil, false, ErrInvalidArgument
	}
	if limit <= 0 {
		limit = defaultMessagesLimit
	}
	if limit > maxMessagesLimit {
		limit = maxMessagesLimit
	}
	msgs, hasMore, err := s.messageRepo.GetLastMessages(ctx, chatID, limit, beforeMessageID)
	if err != nil {
		return nil, false, mapRepoError(err, "chat not found")
	}
	return msgs, hasMore, nil
}

func (s *Service) ListUserChats(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]domain.ChatPreview, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidArgument
	}
	if limit <= 0 {
		limit = defaultChatsLimit
	}
	if limit > maxChatsLimit {
		limit = maxChatsLimit
	}
	if offset < 0 {
		offset = 0
	}
	s.touchUsers(ctx, userID)
	if s.cache != nil {
		cached, found, err := s.cache.GetUserChats(ctx, userID, limit, offset)
		if err == nil && found {
			return cached, nil
		}
	}
	chats, err := s.chatRepo.ListUserChats(ctx, userID, limit, offset)
	if err != nil {
		return nil, mapRepoError(err, "user chats not found")
	}
	if s.cache != nil {
		_ = s.cache.SetUserChats(ctx, userID, limit, offset, chats)
	}
	return chats, nil
}

func (s *Service) MarkChatRead(ctx context.Context, chatID, userID uuid.UUID, upToMessageID *int64) (int64, error) {
	if chatID == uuid.Nil || userID == uuid.Nil {
		return 0, ErrInvalidArgument
	}
	if upToMessageID != nil && *upToMessageID <= 0 {
		return 0, ErrInvalidArgument
	}
	updated, err := s.messageRepo.MarkChatRead(ctx, chatID, userID, upToMessageID)
	if err != nil {
		return 0, mapRepoError(err, "chat not found")
	}
	s.invalidateUsers(ctx, userID)
	s.touchUsers(ctx, userID)
	return updated, nil
}

func (s *Service) touchUsers(ctx context.Context, userIDs ...uuid.UUID) {
	if s.cache == nil {
		return
	}
	for _, userID := range userIDs {
		if userID == uuid.Nil {
			continue
		}
		_ = s.cache.TouchUser(ctx, userID)
	}
}

func (s *Service) invalidateUsers(ctx context.Context, userIDs ...uuid.UUID) {
	if s.cache == nil {
		return
	}
	for _, userID := range userIDs {
		if userID == uuid.Nil {
			continue
		}
		_ = s.cache.InvalidateUser(ctx, userID)
	}
}

func mapRepoError(err error, notFoundMessage string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.NotFoundError(notFoundMessage, err)
	}
	if domain.IsErrorCode(err, domain.ErrorCodeInvalidArgument) ||
		domain.IsErrorCode(err, domain.ErrorCodeNotFound) ||
		domain.IsErrorCode(err, domain.ErrorCodeConflict) ||
		domain.IsErrorCode(err, domain.ErrorCodeUnauthorized) ||
		domain.IsErrorCode(err, domain.ErrorCodeForbidden) ||
		domain.IsErrorCode(err, domain.ErrorCodeService) ||
		domain.IsErrorCode(err, domain.ErrorCodeInternal) {
		return err
	}
	return domain.InternalError(err)
}
