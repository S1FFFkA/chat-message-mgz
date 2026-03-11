package middleware

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const TraceIDHeader = "x-trace-id"

type traceIDKey struct{}

func TraceIDFromContext(ctx context.Context) string {
	val, ok := ctx.Value(traceIDKey{}).(string)
	if !ok {
		return ""
	}
	return val
}

func UnaryTraceInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		traceID := traceIDFromIncomingMetadata(ctx)
		ctx = context.WithValue(ctx, traceIDKey{}, traceID)
		_ = grpc.SetHeader(ctx, metadata.Pairs(TraceIDHeader, traceID))
		return handler(ctx, req)
	}
}

func traceIDFromIncomingMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return uuid.NewString()
	}
	values := md.Get(TraceIDHeader)
	if len(values) == 0 || values[0] == "" {
		return uuid.NewString()
	}
	return values[0]
}
