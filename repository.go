package eventsource

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
)

type Option func(r *AggregateRepository)

func WithMarshaler(m EventMarshaler) Option {
	return func(r *AggregateRepository) {
		r.Marshaler = m
	}
}

func WithEventStore(s EventStore) Option {
	return func(r *AggregateRepository) {
		r.store = s
	}
}

func WithSnapRepository(s SnapshotStore, m SnapshotMarshaler) Option {
	return func(r *AggregateRepository) {
		r.snaprepo = NewSnapRepository(s, m)
	}
}

func WithDefaultSnapRepository(states ...interface{}) Option {
	m := &JsonSnapshotMarshaler{}
	m.Bind(states...)

	return func(r *AggregateRepository) {
		r.snaprepo = &SnapshotRepository{
			marshaler: m,
			store:     NewInmemSnapStore(),
		}
	}
}

func NewRepository(aggr AggregateRoot, opts ...Option) AggregateRootRepository {
	typ := reflect.TypeOf(aggr)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	repo := &AggregateRepository{
		aggregate: typ,
		store:     NewInmemEventStore(),
		Marshaler: &JsonEventMarshaler{},
	}

	for _, opt := range opts {
		opt(repo)
	}

	if _, ok := aggr.(SnapshottingBehaviour); ok && repo.snaprepo == nil {
		_, err := fmt.Printf("%T must provide AggregateRootSnapshotRepository implementation\n", aggr)
		panic(err)
	}

	return repo
}

type AggregateRepository struct {
	aggregate reflect.Type
	store     EventStore
	Marshaler EventMarshaler
	snaprepo  AggregateRootSnapshotRepository
}

func (r *AggregateRepository) New() interface{} {
	return reflect.New(r.aggregate).Interface()
}

func (r *AggregateRepository) Save(ctx context.Context, aggr AggregateRoot) error {
	if aggr.AggregateRootID() == "" {
		return errors.New("empty AggregateRootID")
	}
	defer aggr.CommitEvents()

	events := aggr.GetUncommitedEvents()
	if len(events) == 0 {
		return nil
	}

	var history History
	for _, e := range events {
		model, err := r.Marshaler.Marshal(e)
		if err != nil {
			return err
		}

		history = append(history, model)
	}

	err := r.store.SaveEvents(ctx, aggr.AggregateRootID(), history, aggr.GetVersion())
	if err != nil {
		return err
	}

	if v, ok := aggr.(SnapshottingBehaviour); ok {
		if v.StreamSize()%v.SnapshotInterval() == 0 {
			snap := SnapshotSkeleton{
				ID:      v.AggregateRootID(),
				Version: aggr.GetVersion(),
				State:   v.GetState(),
			}

			if err := r.snaprepo.Save(ctx, snap); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *AggregateRepository) GetByID(ctx context.Context, aggrID string) (AggregateRoot, error) {
	aggregate, err := r.LoadFromSnap(ctx, aggrID)
	if err != nil {
		return nil, err
	}

	if aggregate != nil {
		return aggregate, nil
	}

	history, err := r.store.GetEventsForAggregate(ctx, aggrID, 0)
	if err != nil {
		return nil, err
	}

	var events []Event
	for _, model := range history {
		event, err := r.Marshaler.Unmarshal(model)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	aggr, _ := r.New().(AggregateRoot)
	if err := aggr.LoadFromHistory(aggr, events); err != nil {
		return nil, err
	}

	return aggr, nil
}

func (r *AggregateRepository) LoadFromSnap(ctx context.Context, aggrID string) (SnapshottingBehaviour, error) {
	if aggr, ok := r.New().(SnapshottingBehaviour); ok {
		if !ok || !aggr.SnapshottingEnable() {
			return nil, nil
		}

		snap, err := r.snaprepo.GetByID(ctx, aggr.AggregateRootID(), 0)
		if err != nil {
			if errors.Is(err, ErrSnapNotFound) {
				return nil, nil
			}

			return nil, err
		}

		version := snap.CurrentVersion()
		aggr.ApplyState(snap)

		history, err := r.store.GetEventsForAggregate(ctx, aggrID, version)
		if err != nil {
			return nil, err
		}

		var events []Event
		for _, model := range history {
			event, err := r.Marshaler.Unmarshal(model)
			if err != nil {
				return nil, err
			}

			events = append(events, event)
		}

		if err := aggr.LoadFromHistory(aggr, events); err != nil {
			return nil, err
		}

		return aggr, nil
	}

	return nil, nil
}

func NewSnapRepository(store SnapshotStore, marshaler SnapshotMarshaler) AggregateRootSnapshotRepository {
	return &SnapshotRepository{
		store:     store,
		marshaler: marshaler,
	}
}

type SnapshotRepository struct {
	store     SnapshotStore
	marshaler SnapshotMarshaler
}

func (r *SnapshotRepository) Save(ctx context.Context, snap Snapshot) error {
	model, err := r.marshaler.Marshal(snap)
	if err != nil {
		return fmt.Errorf("unable to marshal snapshot: %v", err)
	}

	r.store.SaveSnapshot(ctx, snap.AggregateRootID(), model, snap.CurrentVersion())

	return nil
}

func (r *SnapshotRepository) GetByID(ctx context.Context, agrID string, version int) (Snapshot, error) {
	if version == 0 {
		version = math.MaxInt32
	}

	model, err := r.store.GetSnapshotForAggregate(ctx, agrID, version)
	if err != nil {
		return nil, err
	}

	snap, err := r.marshaler.Unmarshal(model)
	if err != nil {
		return nil, err
	}

	return snap, nil
}
