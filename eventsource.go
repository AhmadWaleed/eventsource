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
	Version() int
	Apply(e Event, isNew bool) error
	LoadFromHistory(history []Event) error
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
	StreamSize() int
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
