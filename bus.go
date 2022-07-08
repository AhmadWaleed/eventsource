package eventsource

import (
	"context"
	"errors"
	"fmt"

	"github.com/AhmadWaleed/eventsource/command"
)

// Constructor command implementing this interface
// can be used to construct new aggregate
type Constructor interface {
	New() bool
}

// AggregateCommand confirms that the command must have aggregate id
// which can be helpful to replay aggreate events from event store
// to build aggregate current state.
type AggregateCommand interface {
	AggregateID() string
}

// Command is an model for AggregateCommand used to
// avoid code boilerplate for commands implmenting AggregateCommand
type Command struct {
	ID string
}

func (c Command) AggregateID() string {
	return c.ID
}

type AggregateHandler interface {
	command.Handler
}

func NewCommandBus(repo AggregateRootRepository, middlewares ...command.Middleware) command.CommandSender {
	return &commandBus{
		repo:        repo,
		middlewares: middlewares,
	}
}

// commandBus default command bus which can be used to syncronously
// process aggregate commands. Its an implmentation of command.CommandSender inferface.
type commandBus struct {
	repo        AggregateRootRepository
	middlewares []command.Middleware
}

// Send process aggregate command, publishes the relevant
// events and save the aggreate state/events into event store.
func (b *commandBus) Send(ctx context.Context, cmd interface{}) error {
	for _, m := range b.middlewares {
		if err := m.Before(ctx, cmd); err != nil {
			return err
		}
	}

	agrCmd, ok := cmd.(AggregateCommand)
	if !ok {
		return errors.New("command must be a AggregateCommand")
	}

	var aggregate AggregateRoot
	if v, ok := agrCmd.(Constructor); ok && v.New() {
		aggregate = b.repo.New().(AggregateRoot)
	} else {
		aggregateID := agrCmd.AggregateID()
		v, err := b.repo.GetByID(ctx, aggregateID)
		if err != nil {
			return fmt.Errorf("unable to get aggregate by ID: %v", err)
		}
		aggregate = v
	}

	handler, ok := aggregate.(AggregateHandler)
	if !ok {
		return fmt.Errorf("%T do not implement CommandHandler", aggregate)
	}

	err := handler.Handle(ctx, agrCmd)
	if err != nil {
		return fmt.Errorf("could not apply command, %T, to aggregate, %T", cmd, aggregate)
	}

	err = b.repo.Save(ctx, aggregate)
	if err != nil {
		return fmt.Errorf("could not save aggregate %T %v", aggregate, err)
	}

	return nil
}
