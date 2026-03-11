package chat

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gitlab.com/siffka/chat-message-mgz/internal/domain"
	grpcmw "gitlab.com/siffka/chat-message-mgz/internal/transport/grpc/middleware"
	chatsvc "gitlab.com/siffka/chat-message-mgz/internal/usecase/chat"
	chatv1 "gitlab.com/siffka/chat-message-mgz/pkg/api/chat/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	chatv1.UnimplementedChatMessageServiceServer
	chatService chatsvc.UseCase
	logger      *zap.Logger
}

func NewServer(chatService chatsvc.UseCase, logger *zap.Logger) *Server {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Server{
		chatService: chatService,
		logger:      logger,
	}
}

func (s *Server) CreateDirectChat(ctx context.Context, req *chatv1.CreateDirectChatRequest) (*chatv1.CreateDirectChatResponse, error) {
	user1ID, err := parseUUID(req.GetUser1Id())
	if err != nil {
		return nil, domain.ToGRPCStatus(domain.InvalidArgumentError("invalid user1_id", err))
	}
	user2ID, err := parseUUID(req.GetUser2Id())
	if err != nil {
		return nil, domain.ToGRPCStatus(domain.InvalidArgumentError("invalid user2_id", err))
	}

	chat, err := s.chatService.CreateDirectChat(ctx, user1ID, user2ID)
	if err != nil {
		return nil, s.handleError(ctx, "CreateDirectChat", err,
			zap.String("user1_id", req.GetUser1Id()),
			zap.String("user2_id", req.GetUser2Id()),
		)
	}
	return &chatv1.CreateDirectChatResponse{Chat: toProtoChat(chat)}, nil
}

func (s *Server) DeleteChat(ctx context.Context, req *chatv1.DeleteChatRequest) (*chatv1.DeleteChatResponse, error) {
	chatID, err := parseUUID(req.GetChatId())
	if err != nil {
		return nil, domain.ToGRPCStatus(domain.InvalidArgumentError("invalid chat_id", err))
	}
	if err = s.chatService.DeleteChat(ctx, chatID); err != nil {
		return nil, s.handleError(ctx, "DeleteChat", err, zap.String("chat_id", req.GetChatId()))
	}
	return &chatv1.DeleteChatResponse{}, nil
}

func (s *Server) CreateMessage(ctx context.Context, req *chatv1.CreateMessageRequest) (*chatv1.CreateMessageResponse, error) {
	chatID, userID, err := parseChatAndUser(req.GetChatId(), req.GetSenderUserId())
	if err != nil {
		return nil, domain.ToGRPCStatus(err)
	}
	message, err := s.chatService.CreateMessage(ctx, chatID, userID, req.GetText())
	if err != nil {
		return nil, s.handleError(ctx, "CreateMessage", err,
			zap.String("chat_id", req.GetChatId()),
			zap.String("sender_user_id", req.GetSenderUserId()),
		)
	}
	return &chatv1.CreateMessageResponse{Message: toProtoMessage(message)}, nil
}

func (s *Server) SendMessage(ctx context.Context, req *chatv1.SendMessageRequest) (*chatv1.SendMessageResponse, error) {
	chatID, userID, err := parseChatAndUser(req.GetChatId(), req.GetSenderUserId())
	if err != nil {
		return nil, domain.ToGRPCStatus(err)
	}
	message, err := s.chatService.CreateMessage(ctx, chatID, userID, req.GetText())
	if err != nil {
		return nil, s.handleError(ctx, "SendMessage", err,
			zap.String("chat_id", req.GetChatId()),
			zap.String("sender_user_id", req.GetSenderUserId()),
		)
	}
	return &chatv1.SendMessageResponse{Message: toProtoMessage(message)}, nil
}

func (s *Server) UpdateMessageStatus(ctx context.Context, req *chatv1.UpdateMessageStatusRequest) (*chatv1.UpdateMessageStatusResponse, error) {
	chatID, err := parseUUID(req.GetChatId())
	if err != nil {
		return nil, domain.ToGRPCStatus(domain.InvalidArgumentError("invalid chat_id", err))
	}
	domainStatus, err := fromProtoStatus(req.GetStatus())
	if err != nil {
		return nil, domain.ToGRPCStatus(err)
	}
	message, err := s.chatService.UpdateMessageStatus(ctx, chatID, req.GetMessageId(), domainStatus)
	if err != nil {
		return nil, s.handleError(ctx, "UpdateMessageStatus", err,
			zap.String("chat_id", req.GetChatId()),
			zap.Int64("message_id", req.GetMessageId()),
			zap.String("status", req.GetStatus().String()),
		)
	}
	return &chatv1.UpdateMessageStatusResponse{Message: toProtoMessage(message)}, nil
}

func (s *Server) GetChatPreview(ctx context.Context, req *chatv1.GetChatPreviewRequest) (*chatv1.GetChatPreviewResponse, error) {
	chatID, err := parseUUID(req.GetChatId())
	if err != nil {
		return nil, domain.ToGRPCStatus(domain.InvalidArgumentError("invalid chat_id", err))
	}
	userID, err := userIDFromMetadata(ctx)
	if err != nil {
		return nil, domain.ToGRPCStatus(err)
	}
	preview, err := s.chatService.GetChatPreview(ctx, chatID, userID)
	if err != nil {
		return nil, s.handleError(ctx, "GetChatPreview", err, zap.String("chat_id", req.GetChatId()))
	}
	return &chatv1.GetChatPreviewResponse{Preview: toProtoPreview(preview)}, nil
}

func (s *Server) GetLastMessages(ctx context.Context, req *chatv1.GetLastMessagesRequest) (*chatv1.GetLastMessagesResponse, error) {
	chatID, err := parseUUID(req.GetChatId())
	if err != nil {
		return nil, domain.ToGRPCStatus(domain.InvalidArgumentError("invalid chat_id", err))
	}
	var before *int64
	if req.GetBeforeMessageId() > 0 {
		val := req.GetBeforeMessageId()
		before = &val
	}
	messages, hasMore, err := s.chatService.GetLastMessages(ctx, chatID, req.GetLimit(), before)
	if err != nil {
		return nil, s.handleError(ctx, "GetLastMessages", err,
			zap.String("chat_id", req.GetChatId()),
			zap.Int32("limit", req.GetLimit()),
			zap.Int64("before_message_id", req.GetBeforeMessageId()),
		)
	}
	result := make([]*chatv1.Message, 0, len(messages))
	for _, m := range messages {
		result = append(result, toProtoMessage(m))
	}
	return &chatv1.GetLastMessagesResponse{
		Messages: result,
		HasMore:  hasMore,
	}, nil
}

func (s *Server) ListUserChats(ctx context.Context, req *chatv1.ListUserChatsRequest) (*chatv1.ListUserChatsResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, domain.ToGRPCStatus(domain.InvalidArgumentError("invalid user_id", err))
	}
	chats, err := s.chatService.ListUserChats(ctx, userID, req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, s.handleError(ctx, "ListUserChats", err,
			zap.String("user_id", req.GetUserId()),
			zap.Int32("limit", req.GetLimit()),
			zap.Int32("offset", req.GetOffset()),
		)
	}
	result := make([]*chatv1.ChatPreview, 0, len(chats))
	for _, c := range chats {
		result = append(result, toProtoPreview(c))
	}
	return &chatv1.ListUserChatsResponse{Chats: result}, nil
}

func (s *Server) MarkChatRead(ctx context.Context, req *chatv1.MarkChatReadRequest) (*chatv1.MarkChatReadResponse, error) {
	chatID, userID, err := parseChatAndUser(req.GetChatId(), req.GetUserId())
	if err != nil {
		return nil, domain.ToGRPCStatus(err)
	}
	var upTo *int64
	if req.GetUpToMessageId() > 0 {
		val := req.GetUpToMessageId()
		upTo = &val
	}
	updated, err := s.chatService.MarkChatRead(ctx, chatID, userID, upTo)
	if err != nil {
		return nil, s.handleError(ctx, "MarkChatRead", err,
			zap.String("chat_id", req.GetChatId()),
			zap.String("user_id", req.GetUserId()),
			zap.Int64("up_to_message_id", req.GetUpToMessageId()),
		)
	}
	return &chatv1.MarkChatReadResponse{UpdatedMessages: updated}, nil
}

func parseChatAndUser(chatIDRaw, userIDRaw string) (uuid.UUID, uuid.UUID, error) {
	chatID, err := parseUUID(chatIDRaw)
	if err != nil {
		return uuid.Nil, uuid.Nil, domain.InvalidArgumentError("invalid chat_id", err)
	}
	userID, err := parseUUID(userIDRaw)
	if err != nil {
		return uuid.Nil, uuid.Nil, domain.InvalidArgumentError("invalid user_id", err)
	}
	return chatID, userID, nil
}

func parseUUID(raw string) (uuid.UUID, error) {
	return uuid.Parse(raw)
}

func userIDFromMetadata(ctx context.Context) (uuid.UUID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return uuid.Nil, domain.InvalidArgumentError("missing metadata x-user-id", errors.New("incoming metadata is not present"))
	}
	values := md.Get("x-user-id")
	if len(values) == 0 {
		return uuid.Nil, domain.InvalidArgumentError("missing metadata x-user-id", errors.New("x-user-id metadata key is empty"))
	}
	userID, err := uuid.Parse(values[0])
	if err != nil {
		return uuid.Nil, domain.InvalidArgumentError("invalid metadata x-user-id", err)
	}
	return userID, nil
}

func fromProtoStatus(status chatv1.MessageStatus) (domain.MessageStatus, error) {
	switch status {
	case chatv1.MessageStatus_MESSAGE_STATUS_SENT:
		return domain.MessageStatusSent, nil
	case chatv1.MessageStatus_MESSAGE_STATUS_DELIVERED:
		return domain.MessageStatusDelivered, nil
	case chatv1.MessageStatus_MESSAGE_STATUS_READ:
		return domain.MessageStatusRead, nil
	default:
		return "", domain.InvalidArgumentError("invalid message status", errors.New(status.String()))
	}
}

func toProtoStatus(status domain.MessageStatus) chatv1.MessageStatus {
	switch status {
	case domain.MessageStatusSent:
		return chatv1.MessageStatus_MESSAGE_STATUS_SENT
	case domain.MessageStatusDelivered:
		return chatv1.MessageStatus_MESSAGE_STATUS_DELIVERED
	case domain.MessageStatusRead:
		return chatv1.MessageStatus_MESSAGE_STATUS_READ
	default:
		return chatv1.MessageStatus_MESSAGE_STATUS_UNSPECIFIED
	}
}

func toProtoChat(chat domain.Chat) *chatv1.Chat {
	return &chatv1.Chat{
		Id:        chat.ID.String(),
		User1Id:   chat.User1ID.String(),
		User2Id:   chat.User2ID.String(),
		CreatedAt: timestamppb.New(chat.CreatedAt),
	}
}

func toProtoMessage(msg domain.Message) *chatv1.Message {
	return &chatv1.Message{
		Id:           msg.ID,
		ChatId:       msg.ChatID.String(),
		SenderUserId: msg.SenderUserID.String(),
		Text:         msg.Text,
		Status:       toProtoStatus(msg.Status),
		CreatedAt:    timestamppb.New(msg.CreatedAt),
		UpdatedAt:    timestamppb.New(msg.UpdatedAt),
	}
}

func toProtoPreview(preview domain.ChatPreview) *chatv1.ChatPreview {
	result := &chatv1.ChatPreview{
		ChatId:      preview.ChatID.String(),
		OtherUserId: preview.OtherUserID.String(),
		UnreadCount: preview.UnreadCount,
		LastMessage: &chatv1.Message{},
	}
	if preview.LastMessageAt != nil {
		result.LastMessageAt = timestamppb.New(*preview.LastMessageAt)
		result.LastMessage.CreatedAt = timestamppb.New(*preview.LastMessageAt)
	}
	if preview.LastStatus != nil {
		result.LastMessage.Status = toProtoStatus(*preview.LastStatus)
	}
	result.LastMessage.ChatId = preview.ChatID.String()
	result.LastMessage.Text = preview.LastMessage
	return result
}

func mapError(err error) error {
	if err == nil {
		return nil
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
	if errors.Is(err, chatsvc.ErrInvalidArgument) ||
		errors.Is(err, chatsvc.ErrSelfChat) ||
		errors.Is(err, chatsvc.ErrEmptyMessage) ||
		errors.Is(err, chatsvc.ErrMessageTooLong) {
		return domain.InvalidArgumentError("invalid request", err)
	}
	return domain.InternalError(err)
}

func (s *Server) handleError(ctx context.Context, method string, err error, fields ...zap.Field) error {
	appErr := mapError(err)
	st := status.Convert(domain.ToGRPCStatus(appErr))
	logFields := []zap.Field{
		zap.String("grpc_method", method),
		zap.String("trace_id", grpcmw.TraceIDFromContext(ctx)),
		zap.String("grpc_code", st.Code().String()),
		zap.String("app_error_code", appErrorCode(appErr)),
		zap.Error(err),
	}
	logFields = append(logFields, fields...)

	if domain.IsErrorCode(appErr, domain.ErrorCodeInvalidArgument) ||
		domain.IsErrorCode(appErr, domain.ErrorCodeNotFound) {
		s.logger.Warn("request failed validation", logFields...)
		return domain.ToGRPCStatus(appErr)
	}

	s.logger.Error("request failed with internal error", logFields...)
	return domain.ToGRPCStatus(appErr)
}

func appErrorCode(err error) string {
	switch {
	case domain.IsErrorCode(err, domain.ErrorCodeInvalidArgument):
		return string(domain.ErrorCodeInvalidArgument)
	case domain.IsErrorCode(err, domain.ErrorCodeNotFound):
		return string(domain.ErrorCodeNotFound)
	case domain.IsErrorCode(err, domain.ErrorCodeConflict):
		return string(domain.ErrorCodeConflict)
	case domain.IsErrorCode(err, domain.ErrorCodeUnauthorized):
		return string(domain.ErrorCodeUnauthorized)
	case domain.IsErrorCode(err, domain.ErrorCodeForbidden):
		return string(domain.ErrorCodeForbidden)
	case domain.IsErrorCode(err, domain.ErrorCodeService):
		return string(domain.ErrorCodeService)
	case domain.IsErrorCode(err, domain.ErrorCodeInternal):
		return string(domain.ErrorCodeInternal)
	default:
		return "unknown"
	}
}
