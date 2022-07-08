package command

import (
	"context"
)

type Middleware interface {
	Before(ctx context.Context, v interface{}) error
}

type CommandSender interface {
	Send(ctx context.Context, cmd interface{}) error
}

type EventPublisher interface {
	Publish(ctx context.Context, v interface{}) error
}

type CommandEventBus interface {
	CommandSender
	EventPublisher
}

type Handler interface {
	Handle(ctx context.Context, v interface{}) error
}

type MessageMarshaler interface {
}
