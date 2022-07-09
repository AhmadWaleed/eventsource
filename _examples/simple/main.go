package main

import (
	"context"
	"fmt"
	"time"

	"github.com/AhmadWaleed/eventsource"
)

// Flight states
const (
	Ready   = "ReadyForDeparture"
	Airborn = "Flying"
	Landed  = "Landed"
)

// Events
type FlightPrepared struct{ eventsource.EventSkeleton }
type FlightDepartured struct{ eventsource.EventSkeleton }
type FlightLanded struct{ eventsource.EventSkeleton }

func NewFlight(num string) *Flight {
	return &Flight{number: num}
}

// AggregateRoot
type Flight struct {
	number        string
	version       int
	CurrentStatus string
	stream        []eventsource.Event
}

func (f *Flight) Ready() error {
	e := &FlightPrepared{eventsource.EventSkeleton{
		At: time.Now(),
		ID: f.number,
	}}

	return f.Apply(e, true)
}

func (f *Flight) Depart() error {
	e := &FlightDepartured{eventsource.EventSkeleton{
		At: time.Now(),
		ID: f.number,
	}}

	return f.Apply(e, true)
}

func (f *Flight) Land() error {
	e := &FlightLanded{eventsource.EventSkeleton{
		At: time.Now(),
		ID: f.number,
	}}

	return f.Apply(e, true)
}

func (f *Flight) AggregateRootID() string {
	return f.number
}

func (f *Flight) Version() int {
	return f.version
}

func (f *Flight) Handle(ctx context.Context, v interface{}) error {
	var err error
	switch cmd := v.(type) {
	case ReadyFlight:
		f.number = cmd.AggregateID()
		err = f.Ready()
	case DepartFlight:
		f.number = cmd.AggregateID()
		err = f.Depart()
	case LandFlight:
		f.number = cmd.AggregateID()
		err = f.Land()
	default:
		err = fmt.Errorf("unexpected command %T", cmd)
	}

	return err
}

func (f *Flight) Apply(e eventsource.Event, isNew bool) error {
	switch v := e.(type) {
	case *FlightPrepared:
		f.number = v.AggregateID()
		f.CurrentStatus = Ready
	case *FlightDepartured:
		f.number = v.AggregateID()
		f.CurrentStatus = Airborn
	case *FlightLanded:
		f.number = v.AggregateID()
		f.CurrentStatus = Landed
	default:
		return fmt.Errorf("unexpected event %T", e)
	}

	if isNew {
		f.stream = append(f.stream, e)
	}

	return nil
}

func (f *Flight) LoadFromHistory(history []eventsource.Event) error {
	for _, e := range history {
		if err := f.Apply(e, false); err != nil {
			return err
		}
	}

	f.version = len(history) - 1

	return nil
}

func (f *Flight) GetUncommitedEvents() []eventsource.Event {
	return f.stream
}

func (f *Flight) CommitEvents() {
	f.stream = []eventsource.Event{}
}

// Commands
type ReadyFlight struct{ eventsource.Command }
type DepartFlight struct{ eventsource.Command }
type LandFlight struct{ eventsource.Command }

func (ReadyFlight) New() bool { return true }

type logcmd struct{}

func (logcmd) Before(_ context.Context, v interface{}) error {
	_, err := fmt.Printf("Executing %T command\n\n", v)
	return err
}

func main() {
	marshaler := new(eventsource.JsonEventMarshaler)
	marshaler.Bind(FlightPrepared{}, FlightDepartured{}, FlightLanded{})

	var (
		ctx    = context.Background()
		flight = NewFlight("ABC-1233")
		repo   = eventsource.NewRepository(flight, eventsource.WithMarshaler(marshaler))
		bus    = eventsource.NewCommandBus(repo, logcmd{})
	)

	bus.Send(ctx, ReadyFlight{eventsource.Command{ID: flight.AggregateRootID()}})
	aggregate, err := repo.GetByID(ctx, flight.AggregateRootID())
	checkErr(err)

	print(aggregate.(*Flight))

	bus.Send(ctx, DepartFlight{eventsource.Command{ID: flight.AggregateRootID()}})
	aggregate, err = repo.GetByID(ctx, flight.AggregateRootID())
	checkErr(err)

	print(aggregate.(*Flight))

	bus.Send(ctx, LandFlight{eventsource.Command{ID: flight.AggregateRootID()}})
	aggregate, err = repo.GetByID(ctx, flight.AggregateRootID())
	checkErr(err)

	print(aggregate.(*Flight))
}

func print(f *Flight) {
	fmt.Printf("Flight Number: %s\n", f.AggregateRootID())
	fmt.Printf("Flight Status: %s\n", f.CurrentStatus)
	fmt.Printf("Event Version: %d\n", f.Version())
	fmt.Println()
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
