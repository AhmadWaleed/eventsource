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

type ShipmentPacked struct{ EventSkeleton }
type ShipmentPickedUp struct{ EventSkeleton }
type ShipmentShipped struct{ EventSkeleton }

type Shipment struct {
	AggregateRootBase
	Status string
}

func (s *Shipment) Handle(ctx context.Context, v interface{}) error {
	var e Event
	switch cmd := v.(type) {
	case PackShipment:
		e = &ShipmentPacked{EventSkeleton{
			At: time.Now(),
			ID: cmd.AggregateID(),
		}}
	case PickupShipment:
		e = &ShipmentPickedUp{EventSkeleton{
			At: time.Now(),
			ID: cmd.AggregateID(),
		}}
	case ShipShipment:
		e = &ShipmentShipped{EventSkeleton{
			At: time.Now(),
			ID: cmd.AggregateID(),
		}}
	default:
		return fmt.Errorf("unexpected command %T", cmd)
	}

	return s.Apply(s, e, true)
}

func (s *Shipment) On(e Event) error {
	switch v := e.(type) {
	case *ShipmentPacked:
		s.ID = v.AggregateID()
		s.Status = Packed
	case *ShipmentPickedUp:
		s.ID = v.AggregateID()
		s.Status = PickedUp
	case *ShipmentShipped:
		s.ID = v.AggregateID()
		s.Status = Shipped
	default:
		return fmt.Errorf("unexpected event %T", e)
	}

	return nil
}

type PackShipment struct{ Command }
type PickupShipment struct{ Command }
type ShipShipment struct{ Command }

func (PackShipment) New() bool { return true }

func Example() {
	marshaler := new(JsonEventMarshaler)
	marshaler.Bind(ShipmentPacked{}, ShipmentPickedUp{}, ShipmentShipped{})

	var (
		ctx      = context.Background()
		shipment = &Shipment{}
		repo     = NewRepository(shipment, WithMarshaler(marshaler))
		bus      = NewCommandBus(repo)
	)

	bus.Send(ctx, PackShipment{Command{ID: "abc123"}})
	aggregate, _ := repo.GetByID(ctx, "abc123")

	shipment, _ = aggregate.(*Shipment)
	fmt.Println(shipment.Status)
	fmt.Println(shipment.GetVersion())
	// Output: packed
	// Output: 0

	bus.Send(ctx, PickupShipment{Command{ID: shipment.AggregateRootID()}})
	aggregate, _ = repo.GetByID(ctx, shipment.AggregateRootID())
	shipment, _ = aggregate.(*Shipment)

	fmt.Println(shipment.Status)
	fmt.Println(shipment.GetVersion())
	// Output: picked-up
	// Output: 1

	bus.Send(ctx, ShipShipment{Command{ID: shipment.AggregateRootID()}})
	aggregate, _ = repo.GetByID(ctx, shipment.AggregateRootID())
	shipment, _ = aggregate.(*Shipment)

	fmt.Println(shipment.Status)
	fmt.Println(shipment.GetVersion())
	// Output: shipped
	// Output: 2
}
