package events

import "context"

type NoopPublisher struct{}

func NewNoopPublisher() *NoopPublisher {
	return &NoopPublisher{}
}

func (p *NoopPublisher) PublishChatUpdated(context.Context, ChatUpdatedEvent) error {
	return nil
}
