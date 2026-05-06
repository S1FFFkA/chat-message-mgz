package events

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaPublisher struct {
	writer *kafka.Writer
}

type kafkaChatUpdatedPayload struct {
	Type            ChatEventType `json:"type"`
	ChatID          string        `json:"chat_id"`
	MessageID       int64         `json:"message_id,omitempty"`
	UserID          string        `json:"user_id,omitempty"`
	RecipientUserID string        `json:"recipient_user_id,omitempty"`
	MessageText     string        `json:"message_text,omitempty"`
	UnreadCount     int64         `json:"unread_count,omitempty"`
	UpdatedMessages int64         `json:"updated_messages,omitempty"`
	OccurredAt      time.Time     `json:"occurred_at"`
}

func NewKafkaPublisher(brokers []string, topic string) (*KafkaPublisher, error) {
	if len(brokers) == 0 {
		return nil, errors.New("kafka brokers are required")
	}
	cleanBrokers := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		v := strings.TrimSpace(broker)
		if v != "" {
			cleanBrokers = append(cleanBrokers, v)
		}
	}
	if len(cleanBrokers) == 0 {
		return nil, errors.New("kafka brokers are required")
	}
	if strings.TrimSpace(topic) == "" {
		return nil, errors.New("kafka topic is required")
	}
	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(cleanBrokers...),
			Topic:                  topic,
			AllowAutoTopicCreation: true,
			Balancer:               &kafka.LeastBytes{},
		},
	}, nil
}

func (p *KafkaPublisher) PublishChatUpdated(ctx context.Context, event ChatUpdatedEvent) error {
	payload := kafkaChatUpdatedPayload{
		Type:            event.Type,
		ChatID:          event.ChatID.String(),
		MessageID:       event.MessageID,
		UserID:          event.UserID.String(),
		RecipientUserID: event.RecipientUserID.String(),
		MessageText:     event.MessageText,
		UnreadCount:     event.UnreadCount,
		UpdatedMessages: event.UpdatedMessages,
		OccurredAt:      event.OccurredAt,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := kafka.Message{
		Key:   []byte(payload.ChatID),
		Value: raw,
		Time:  event.OccurredAt,
	}
	return p.writer.WriteMessages(ctx, msg)
}

func (p *KafkaPublisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
