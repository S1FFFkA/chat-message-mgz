package main

import (
	"context"
	"net"
	"net/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	
	"github.com/S1FFFkA/chat-message-mgz/internal/cache/chatcache"
	"github.com/S1FFFkA/chat-message-mgz/internal/config"
	"github.com/S1FFFkA/chat-message-mgz/internal/events"
	chatrepo "github.com/S1FFFkA/chat-message-mgz/internal/repository/chat"
	msgrepo "github.com/S1FFFkA/chat-message-mgz/internal/repository/message"
	pgstorage "github.com/S1FFFkA/chat-message-mgz/internal/storage/postgres"
	chattransport "github.com/S1FFFkA/chat-message-mgz/internal/transport/grpc/chat"
	grpcmw "github.com/S1FFFkA/chat-message-mgz/internal/transport/grpc/middleware"
	chatsvc "github.com/S1FFFkA/chat-message-mgz/internal/usecase/chat"
	"github.com/S1FFFkA/chat-message-mgz/pkg/logger"

	chatv1 "github.com/S1FFFkA/chat-message-mgz/pkg/api/chat/v1"
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
	messageRepo := msgrepo.NewRepository(pool)
	publisher := events.Publisher(events.NewNoopPublisher())
	if len(cfg.KafkaBrokers) > 0 {
		kafkaPublisher, kafkaErr := events.NewKafkaPublisher(cfg.KafkaBrokers, cfg.KafkaTopicChatMessages)
		if kafkaErr != nil {
			log.Warn("kafka publisher init failed, events are disabled", zap.Error(kafkaErr))
		} else {
			publisher = kafkaPublisher
			defer func() {
				_ = kafkaPublisher.Close()
			}()
			log.Info("kafka publisher enabled",
				zap.Strings("brokers", cfg.KafkaBrokers),
				zap.String("topic", cfg.KafkaTopicChatMessages),
			)
		}
	}
	chatService := chatsvc.NewServiceWithDeps(chatRepo, messageRepo, nil, publisher)
	if cfg.RedisAddr != "" {
		redisClient := chatcache.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
		defer func() {
			_ = redisClient.Close()
		}()
		cache := chatcache.New(redisClient, cfg.CacheTTL)
		if err = cache.Ping(ctx); err != nil {
			log.Warn("redis is configured but not reachable, cache is disabled", zap.Error(err))
		} else {
			chatService = chatsvc.NewServiceWithDeps(chatRepo, messageRepo, cache, publisher)
			log.Info("redis cache enabled",
				zap.String("redis_addr", cfg.RedisAddr),
				zap.Duration("cache_ttl", cfg.CacheTTL),
			)
		}
	}

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal("failed to listen",
			zap.String("port", cfg.GRPCPort),
			zap.Error(err),
		)
	}

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	go func() {
		log.Info("metrics server started", zap.String("port", "9101"))

		if err := http.ListenAndServe(":9101", metricsMux); err != nil {
			log.Warn("metrics server stopped", zap.Error(err))
		}
	}()

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
