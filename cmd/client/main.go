package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	chatv1 "gitlab.com/siffka/chat-message-mgz/pkg/api/chat/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	addr := os.Getenv("CHAT_GRPC_ADDR")
	if addr == "" {
		addr = "localhost:50051"
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		failf("connect grpc: %v", err)
	}
	defer conn.Close()

	client := chatv1.NewChatMessageServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch os.Args[1] {
	case "create-chat":
		createChat(ctx, client, os.Args[2:])
	case "send-message":
		sendMessage(ctx, client, os.Args[2:])
	case "list-chats":
		listChats(ctx, client, os.Args[2:])
	case "get-messages":
		getMessages(ctx, client, os.Args[2:])
	case "preview":
		preview(ctx, client, os.Args[2:])
	case "mark-read":
		markRead(ctx, client, os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func createChat(ctx context.Context, c chatv1.ChatMessageServiceClient, args []string) {
	fs := flag.NewFlagSet("create-chat", flag.ExitOnError)
	user1 := fs.String("user1", "", "UUID user1")
	user2 := fs.String("user2", "", "UUID user2")
	_ = fs.Parse(args)
	require(*user1 != "" && *user2 != "", "flags --user1 and --user2 are required")

	resp, err := c.CreateDirectChat(ctx, &chatv1.CreateDirectChatRequest{
		User1Id: *user1,
		User2Id: *user2,
	})
	if err != nil {
		failf("CreateDirectChat: %v", err)
	}
	printProto(resp)
}

func sendMessage(ctx context.Context, c chatv1.ChatMessageServiceClient, args []string) {
	fs := flag.NewFlagSet("send-message", flag.ExitOnError)
	chatID := fs.String("chat", "", "UUID chat")
	sender := fs.String("sender", "", "UUID sender")
	text := fs.String("text", "", "message text")
	_ = fs.Parse(args)
	require(*chatID != "" && *sender != "" && *text != "", "flags --chat, --sender, --text are required")

	resp, err := c.SendMessage(ctx, &chatv1.SendMessageRequest{
		ChatId:          *chatID,
		SenderUserId:    *sender,
		Text:            *text,
		ClientMessageId: uuid.NewString(),
	})
	if err != nil {
		failf("SendMessage: %v", err)
	}
	printProto(resp)
}

func listChats(ctx context.Context, c chatv1.ChatMessageServiceClient, args []string) {
	fs := flag.NewFlagSet("list-chats", flag.ExitOnError)
	user := fs.String("user", "", "UUID user")
	limit := fs.Int("limit", 15, "limit")
	offset := fs.Int("offset", 0, "offset")
	_ = fs.Parse(args)
	require(*user != "", "flag --user is required")

	resp, err := c.ListUserChats(ctx, &chatv1.ListUserChatsRequest{
		UserId: *user,
		Limit:  int32(*limit),
		Offset: int32(*offset),
	})
	if err != nil {
		failf("ListUserChats: %v", err)
	}
	printProto(resp)
}

func getMessages(ctx context.Context, c chatv1.ChatMessageServiceClient, args []string) {
	fs := flag.NewFlagSet("get-messages", flag.ExitOnError)
	chatID := fs.String("chat", "", "UUID chat")
	limit := fs.Int("limit", 50, "limit")
	before := fs.Int64("before", 0, "before_message_id")
	_ = fs.Parse(args)
	require(*chatID != "", "flag --chat is required")

	resp, err := c.GetLastMessages(ctx, &chatv1.GetLastMessagesRequest{
		ChatId:          *chatID,
		Limit:           int32(*limit),
		BeforeMessageId: *before,
	})
	if err != nil {
		failf("GetLastMessages: %v", err)
	}
	printProto(resp)
}

func preview(ctx context.Context, c chatv1.ChatMessageServiceClient, args []string) {
	fs := flag.NewFlagSet("preview", flag.ExitOnError)
	chatID := fs.String("chat", "", "UUID chat")
	user := fs.String("user", "", "UUID viewer user")
	_ = fs.Parse(args)
	require(*chatID != "" && *user != "", "flags --chat and --user are required")

	ctx = metadata.AppendToOutgoingContext(ctx, "x-user-id", *user, "x-trace-id", uuid.NewString())
	resp, err := c.GetChatPreview(ctx, &chatv1.GetChatPreviewRequest{ChatId: *chatID})
	if err != nil {
		failf("GetChatPreview: %v", err)
	}
	printProto(resp)
}

func markRead(ctx context.Context, c chatv1.ChatMessageServiceClient, args []string) {
	fs := flag.NewFlagSet("mark-read", flag.ExitOnError)
	chatID := fs.String("chat", "", "UUID chat")
	user := fs.String("user", "", "UUID user")
	upTo := fs.Int64("up-to", 0, "up_to_message_id")
	_ = fs.Parse(args)
	require(*chatID != "" && *user != "", "flags --chat and --user are required")

	resp, err := c.MarkChatRead(ctx, &chatv1.MarkChatReadRequest{
		ChatId:        *chatID,
		UserId:        *user,
		UpToMessageId: *upTo,
	})
	if err != nil {
		failf("MarkChatRead: %v", err)
	}
	printProto(resp)
}

func printProto(v proto.Message) {
	out, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(v)
	if err != nil {
		failf("marshal response: %v", err)
	}
	fmt.Println(string(out))
}

func require(ok bool, msg string) {
	if ok {
		return
	}
	failf(msg)
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func usage() {
	fmt.Print(`Usage:
  go run ./cmd/client <command> [flags]

Commands:
  create-chat  --user1 <uuid> --user2 <uuid>
  send-message --chat <uuid> --sender <uuid> --text "hello"
  list-chats   --user <uuid> [--limit 15 --offset 0]
  get-messages --chat <uuid> [--limit 50 --before 0]
  preview      --chat <uuid> --user <uuid>
  mark-read    --chat <uuid> --user <uuid> [--up-to 0]
`)
}
