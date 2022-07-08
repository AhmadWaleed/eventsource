package eventsource

import (
	"context"
	"fmt"
	"time"
)

const (
	Packed   = "packed"
	PickedUp = "picked-up"
	Shipped  = "shipped"
)

type ShipmentPacked struct{ EventModel }
type ShipmentPickedUp struct{ EventModel }
type ShipmentShipped struct{ EventModel }

func NewShipment() *Shipment {
	return &Shipment{sku: "abc123"}
}

type Shipment struct {
	sku     string
	Status  string
	stream  []Event
	version int
}

func (s *Shipment) AggregateRootID() string {
	return s.sku
}

func (s *Shipment) Version() int {
	return s.version
}

func (s *Shipment) Handle(ctx context.Context, v interface{}) error {
	var e Event
	switch cmd := v.(type) {
	case PackShipment:
		e = &ShipmentPacked{EventModel{
			At: time.Now(),
			ID: cmd.AggregateID(),
		}}
	case PickupShipment:
		e = &ShipmentPickedUp{EventModel{
			At: time.Now(),
			ID: cmd.AggregateID(),
		}}
	case ShipShipment:
		e = &ShipmentShipped{EventModel{
			At: time.Now(),
			ID: cmd.AggregateID(),
		}}
	default:
		return fmt.Errorf("unexpected command %T", cmd)
	}

	return s.Apply(e, true)
}

func (s *Shipment) Apply(e Event, isNew bool) error {
	switch v := e.(type) {
	case *ShipmentPacked:
		s.sku = v.AggregateID()
		s.Status = Packed
	case *ShipmentPickedUp:
		s.sku = v.AggregateID()
		s.Status = PickedUp
	case *ShipmentShipped:
		s.sku = v.AggregateID()
		s.Status = Shipped
	default:
		return fmt.Errorf("unexpected event %T", e)
	}

	if isNew {
		s.stream = append(s.stream, e)
	}

	return nil
}

func (s *Shipment) LoadFromHistory(history []Event) error {
	for _, e := range history {
		if err := s.Apply(e, false); err != nil {
			return err
		}
	}

	s.version = len(history) - 1

	return nil
}

func (s *Shipment) GetUncommitedEvents() []Event {
	return s.stream
}

func (s *Shipment) CommitEvents() {
	s.stream = []Event{}
}

type PackShipment struct{ Command }
type PickupShipment struct{ Command }
type ShipShipment struct{ Command }

func (PackShipment) New() bool { return true }

func Example() {
	var (
		ctx       = context.Background()
		marshalel = new(JsonMarshaller)
		shipment  = NewShipment()
		repo      = NewRepository(shipment, NewInmemStore(), marshalel)
		bus       = NewCommandBus(repo)
	)

	marshalel.Bind(ShipmentPacked{}, ShipmentPickedUp{}, ShipmentShipped{})
	bus.Send(ctx, PackShipment{Command{ID: shipment.AggregateRootID()}})
	aggregate, _ := repo.GetByID(ctx, shipment.AggregateRootID())

	shipment, _ = aggregate.(*Shipment)
	fmt.Println(shipment.Status)
	fmt.Println(shipment.Version())
	// Output: packed
	// Output: 0

	bus.Send(ctx, PickupShipment{Command{ID: shipment.AggregateRootID()}})
	aggregate, _ = repo.GetByID(ctx, shipment.AggregateRootID())
	shipment, _ = aggregate.(*Shipment)

	fmt.Println(shipment.Status)
	fmt.Println(shipment.Version())
	// Output: picked-up
	// Output: 1

	bus.Send(ctx, ShipShipment{Command{ID: shipment.AggregateRootID()}})
	aggregate, _ = repo.GetByID(ctx, shipment.AggregateRootID())
	shipment, _ = aggregate.(*Shipment)

	fmt.Println(shipment.Status)
	fmt.Println(shipment.Version())
	// Output: shipped
	// Output: 2
}
