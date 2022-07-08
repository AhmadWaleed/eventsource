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

// AggregateRoot
type Flight struct {
	eventsource.AggregateRootBase
	CurrentStatus string
}

func (f *Flight) Ready() error {
	e := &FlightPrepared{eventsource.EventSkeleton{
		At: time.Now(),
		ID: f.ID,
	}}

	return f.Apply(f, e, true)
}

func (f *Flight) Depart() error {
	e := &FlightDepartured{eventsource.EventSkeleton{
		At: time.Now(),
		ID: f.ID,
	}}

	return f.Apply(f, e, true)
}

func (f *Flight) Land() error {
	e := &FlightLanded{eventsource.EventSkeleton{
		At: time.Now(),
		ID: f.ID,
	}}

	return f.Apply(f, e, true)
}

func (f *Flight) Handle(ctx context.Context, v interface{}) error {
	var err error
	switch cmd := v.(type) {
	case ReadyFlight:
		f.ID = cmd.AggregateID()
		err = f.Ready()
	case DepartFlight:
		f.ID = cmd.AggregateID()
		err = f.Depart()
	case LandFlight:
		f.ID = cmd.AggregateID()
		err = f.Land()
	default:
		err = fmt.Errorf("unexpected command %T", cmd)
	}

	return err
}

func (f *Flight) On(e eventsource.Event) error {
	switch v := e.(type) {
	case *FlightPrepared:
		f.ID = v.AggregateID()
		f.CurrentStatus = Ready
	case *FlightDepartured:
		f.ID = v.AggregateID()
		f.CurrentStatus = Airborn
	case *FlightLanded:
		f.ID = v.AggregateID()
		f.CurrentStatus = Landed
	default:
		return fmt.Errorf("unexpected event %T", e)
	}

	return nil
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
		flight = &Flight{}
		repo   = eventsource.NewRepository(flight, eventsource.WithMarshaler(marshaler))
		bus    = eventsource.NewCommandBus(repo, logcmd{})
	)

	bus.Send(ctx, ReadyFlight{eventsource.Command{ID: "ABC1122"}})
	aggregate, err := repo.GetByID(ctx, "ABC1122")
	checkErr(err)

	flight = aggregate.(*Flight)
	print(flight)

	bus.Send(ctx, DepartFlight{eventsource.Command{ID: flight.AggregateRootID()}})
	aggregate, err = repo.GetByID(ctx, flight.AggregateRootID())
	checkErr(err)

	flight = aggregate.(*Flight)
	print(flight)

	bus.Send(ctx, LandFlight{eventsource.Command{ID: flight.AggregateRootID()}})
	aggregate, err = repo.GetByID(ctx, flight.AggregateRootID())
	checkErr(err)

	flight = aggregate.(*Flight)
	print(flight)
}

func print(f *Flight) {
	fmt.Printf("Flight Number: %s\n", f.AggregateRootID())
	fmt.Printf("Flight Status: %s\n", f.CurrentStatus)
	fmt.Printf("Event Version: %d\n", f.GetVersion())
	fmt.Println()
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
