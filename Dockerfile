FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -o /out/chat-service ./cmd/main.go

FROM alpine:3.22

RUN adduser -D -g '' appuser
USER appuser
WORKDIR /app

COPY --from=builder /out/chat-service /app/chat-service

EXPOSE 50051

ENTRYPOINT ["/app/chat-service"]


