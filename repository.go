package eventsource

import (
	"context"
	"errors"
	"reflect"
)

func NewRepository(agr AggregateRoot, store EventStore, m EventMarshaler) AggregateRootRepository {
	typ := reflect.TypeOf(agr)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	return &repository{
		aggregate: typ,
		store:     store,
		marshaler: m,
	}
}

type repository struct {
	aggregate reflect.Type
	store     EventStore
	marshaler EventMarshaler
}

func (r *repository) New() AggregateRoot {
	return reflect.New(r.aggregate).Interface().(AggregateRoot)
}

func (r *repository) Save(ctx context.Context, agr AggregateRoot) error {
	if agr.AggregateRootID() == "" {
		return errors.New("empty AggregateRootID")
	}
	defer agr.CommitEvents()

	events := agr.GetUncommitedEvents()
	if len(events) == 0 {
		return nil
	}

	var history History
	for _, e := range events {
		model, err := r.marshaler.Marshal(e)
		if err != nil {
			return err
		}

		history = append(history, model)
	}

	return r.store.SaveEvents(ctx, agr.AggregateRootID(), history, agr.Version())
}

func (r *repository) GetByID(ctx context.Context, agrID string) (AggregateRoot, error) {
	history, err := r.store.GetEventsForAggregate(ctx, agrID, 1)
	if err != nil {
		return nil, err
	}

	if len(history) == 0 {
		return nil, errors.New("not found")
	}

	var events []Event
	for _, model := range history {
		event, err := r.marshaler.Unmarshal(model)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	aggregate := r.New()

	if err := aggregate.LoadFromHistory(events); err != nil {
		return nil, err
	}

	return aggregate, nil
}
