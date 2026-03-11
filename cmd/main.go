package main

import (
	"context"
	"net"

	"gitlab.com/siffka/chat-message-mgz/internal/config"
	chatrepo "gitlab.com/siffka/chat-message-mgz/internal/repository/chat"
	messagerepo "gitlab.com/siffka/chat-message-mgz/internal/repository/messeg"
	pgstorage "gitlab.com/siffka/chat-message-mgz/internal/storage/postgres"
	chattransport "gitlab.com/siffka/chat-message-mgz/internal/transport/grpc/chat"
	grpcmw "gitlab.com/siffka/chat-message-mgz/internal/transport/grpc/middleware"
	chatsvc "gitlab.com/siffka/chat-message-mgz/internal/usecase/chat"
	"gitlab.com/siffka/chat-message-mgz/pkg/logger"

	chatv1 "gitlab.com/siffka/chat-message-mgz/pkg/api/chat/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log, err := logger.NewJSON()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config", zap.Error(err))
	}

	ctx := context.Background()
	pool, err := pgstorage.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("failed to initialize postgres pool", zap.Error(err))
	}
	defer pool.Close()

	chatRepo := chatrepo.NewRepository(pool)
	messageRepo := messagerepo.NewRepository(pool)
	chatService := chatsvc.NewService(chatRepo, messageRepo)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal("failed to listen",
			zap.String("port", cfg.GRPCPort),
			zap.Error(err),
		)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcmw.UnaryTraceInterceptor()),
	)
	chatv1.RegisterChatMessageServiceServer(grpcServer, chattransport.NewServer(chatService, log))

	// Reflection is useful for local development and testing with grpcurl.
	reflection.Register(grpcServer)

	log.Info("chat-message gRPC server started", zap.String("port", cfg.GRPCPort))
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal("failed to serve gRPC", zap.Error(err))
	}
}
