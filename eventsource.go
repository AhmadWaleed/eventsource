package eventsource

import (
	"context"
	"time"
)

type Event interface {
	// AggregateID returns the aggregate id of the event
	AggregateID() string

	// Version contains version number of aggregate
	EventVersion() int

	// At indicates when the event took place
	EventAt() time.Time
}

// EventSkeleton provides a default implementation of an Event that is suitable for being embedded
type EventSkeleton struct {
	// ID contains the AggregateID
	ID string

	// Version holds the event version
	Version int

	// At contains the event time
	At time.Time
}

// AggregateID implements part of the Event interface
func (m EventSkeleton) AggregateID() string {
	return m.ID
}

// EventVersion implements part of the Event interface
func (m EventSkeleton) EventVersion() int {
	return m.Version
}

// EventAt implements part of the Event interface
func (m EventSkeleton) EventAt() time.Time {
	return m.At
}

type AggregateRoot interface {
	AggregateRootID() string
	GetVersion() int
	StreamSize() int
	On(e Event) error
	Apply(aggr AggregateRoot, e Event, isNew bool) error
	LoadFromHistory(aggr AggregateRoot, history []Event) error
	GetUncommitedEvents() []Event
	CommitEvents()
}

type AggregateRootRepository interface {
	New() interface{}
	Save(ctx context.Context, agr AggregateRoot) error
	GetByID(ctx context.Context, aggregateRootID string) (AggregateRoot, error)
}

// EventModel provides the shape of the records to be saved to the db
type EventModel struct {
	// Version is the event version the Data represents
	Version int

	// At indicates when the event happened; provided as a utility for the store
	At EpochMillis

	// Data contains the Serializer encoded version of the data
	Data []byte
}

type History []EventModel

type EventStore interface {
	SaveEvents(ctx context.Context, aggrID string, models History, version int) error
	GetEventsForAggregate(ctx context.Context, aggrID string, version int) (History, error)
}

type SnapshottingBehaviour interface {
	AggregateRoot
	SnapshotInterval() int
	GetState() interface{}
	ApplyState(s Snapshot)
	SnapshottingEnable() bool
}

type Snapshot interface {
	CurrentVersion() int
	GetState() interface{}
	AggregateRootID() string
}

type SnapshotSkeleton struct {
	ID      string
	Version int
	State   interface{}
}

func (s SnapshotSkeleton) GetState() interface{} {
	return s.State
}

func (s SnapshotSkeleton) CurrentVersion() int {
	return s.Version
}

func (s SnapshotSkeleton) AggregateRootID() string {
	return s.ID
}

type AggregateRootSnapshotRepository interface {
	Save(ctx context.Context, snap Snapshot) error
	GetByID(ctx context.Context, aggregateRootID string, version int) (Snapshot, error)
}

type SnapshotModel struct {
	ID      string
	Version int
	Data    []byte
}

type SnapshotStore interface {
	SaveSnapshot(ctx context.Context, agrID string, model SnapshotModel, version int) error
	GetSnapshotForAggregate(ctx context.Context, agrID string, version int) (SnapshotModel, error)
}

type EventMarshaler interface {
	Bind(e ...Event) error
	Marshal(e Event) (EventModel, error)
	Unmarshal(e EventModel) (Event, error)
}

type SnapshotMarshaler interface {
	Bind(v ...interface{}) error
	Marshal(s Snapshot) (SnapshotModel, error)
	Unmarshal(m SnapshotModel) (Snapshot, error)
}

type AggregateRootBase struct {
	ID         string
	streamSize int
	stream     []Event
	Version    int
}

func (aggr *AggregateRootBase) AggregateRootID() string {
	return aggr.ID
}

func (aggr *AggregateRootBase) GetVersion() int {
	return aggr.Version
}

func (aggr *AggregateRootBase) Apply(aggregate AggregateRoot, e Event, isNew bool) error {
	if isNew {
		aggr.stream = append(aggr.stream, e)
		if _, ok := e.(Constructor); !ok {
			aggr.Version++
		}
	}

	aggr.streamSize++

	return aggregate.On(e)
}

func (aggr *AggregateRootBase) LoadFromHistory(aggregate AggregateRoot, history []Event) error {
	for _, e := range history {
		if err := aggr.Apply(aggregate, e, false); err != nil {
			return err
		}
	}

	aggr.Version = len(history) - 1

	return nil
}

func (i *AggregateRootBase) StreamSize() int {
	return i.streamSize
}

func (aggr *AggregateRootBase) GetUncommitedEvents() []Event {
	return aggr.stream
}

func (aggr *AggregateRootBase) CommitEvents() {
	aggr.stream = []Event{}
}
