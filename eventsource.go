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

// EventModel provides a default implementation of an Event that is suitable for being embedded
type EventModel struct {
	// ID contains the AggregateID
	ID string

	// Version holds the event version
	Version int

	// At contains the event time
	At time.Time
}

// AggregateID implements part of the Event interface
func (m EventModel) AggregateID() string {
	return m.ID
}

// EventVersion implements part of the Event interface
func (m EventModel) EventVersion() int {
	return m.Version
}

// EventAt implements part of the Event interface
func (m EventModel) EventAt() time.Time {
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

type AggregateRootModel struct {
	ID      string
	Version int
}

type AggregateRootRepository interface {
	New() AggregateRoot
	Save(ctx context.Context, agr AggregateRoot) error
	GetByID(ctx context.Context, id string) (AggregateRoot, error)
}

type SnapshottingBehaviour interface {
	SnapshotInterval() int
	CurrentState() interface{}
}

type AggregateRootIDMarshaler interface {
	Marshal() (string, error)
}

type AggregateRootIDUnmarshaler interface {
	Unmarshal(data string) error
}

type EventStore interface {
	SaveEvents(ctx context.Context, agrID string, models History, version int) error
	GetEventsForAggregate(ctx context.Context, agrID string, version int) (History, error)
}

type EventMarshaler interface {
	Bind(e ...Event) error
	Marshal(e Event) (Model, error)
	Unmarshal(e Model) (Event, error)
}
