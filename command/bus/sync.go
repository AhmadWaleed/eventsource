package sync

import (
	"context"
	"fmt"
	"reflect"

	"github.com/AhmadWaleed/eventsource/command"
)

// Bus is an interface for building bus to process both commands and evets.
type Bus interface {
	command.CommandSender
	command.EventPublisher
	// Bind binds handlers v type, v can be either command or event.
	Bind(v interface{}, handlers []command.Handler) error
}

func NewSyncBus(middlewares ...command.Middleware) Bus {
	return &synchronousBus{middlewares: middlewares}
}

// synchronousBus process commands/events handler synchronously
// this is an example how the command pkg can be used extend and
// build any type of command/event bus to process aggregate events.
type synchronousBus struct {
	middlewares []command.Middleware
	handlers    map[reflect.Type][]command.Handler
}

func (b *synchronousBus) Bind(v interface{}, handlers []command.Handler) error {
	t := typ(v)
	if len(b.getHandlers(t)) > 0 {
		return fmt.Errorf("handler already bind to %T", t)
	}

	b.handlers[t] = handlers

	return nil
}

func (b *synchronousBus) Send(ctx context.Context, cmd interface{}) error {
	for _, m := range b.middlewares {
		m.Before(ctx, cmd)
	}

	var err error
	for _, h := range b.getHandlers(cmd) {
		err = h.Handle(ctx, cmd)
	}

	return err
}

func (b *synchronousBus) Publish(ctx context.Context, event interface{}) error {
	for _, m := range b.middlewares {
		m.Before(ctx, event)
	}

	var err error
	for _, h := range b.getHandlers(event) {
		err = h.Handle(ctx, event)
	}

	return err
}

func (b *synchronousBus) getHandlers(v interface{}) []command.Handler {
	t := typ(v)
	if h, ok := b.handlers[t]; ok {
		return h
	}

	return make([]command.Handler, 0)
}

func typ(v interface{}) reflect.Type {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t
}
