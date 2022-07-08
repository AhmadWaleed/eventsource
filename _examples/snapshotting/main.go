package main

import (
	"context"
	"fmt"
	"time"

	"github.com/AhmadWaleed/eventsource"
)

// Events
type InventoryItemCreated struct {
	eventsource.EventSkeleton
	Name     string
	Quantity int
}

func (InventoryItemCreated) New() bool { return true }

type InventoryItemAdjusted struct {
	eventsource.EventSkeleton
	Quantity int
}

type ItemState struct {
	Name     string
	Quantity int
}

type InventoryItem struct {
	eventsource.AggregateRootBase
	currentState *ItemState
}

func (i *InventoryItem) SnapshotInterval() int {
	return 2
}

func (i *InventoryItem) GetState() interface{} {
	return i.currentState
}

func (i *InventoryItem) ApplyState(s eventsource.Snapshot) {
	i.ID = s.AggregateRootID()
	i.Version = s.CurrentVersion()
	i.currentState = s.GetState().(*ItemState)
}

func (i *InventoryItem) SnapshottingEnable() bool {
	return true
}

func (i *InventoryItem) Handle(ctx context.Context, v interface{}) error {
	var err error
	switch cmd := v.(type) {
	case CreateInventoryItem:
		e := &InventoryItemCreated{
			eventsource.EventSkeleton{ID: cmd.AggregateID(), At: time.Now()},
			cmd.Name,
			cmd.Quantity,
		}
		err = i.Apply(i, e, true)
	case AdjustInventoryItem:
		e := &InventoryItemAdjusted{
			eventsource.EventSkeleton{ID: cmd.AggregateID(), At: time.Now()},
			cmd.Quantity,
		}
		err = i.Apply(i, e, true)
	default:
		err = fmt.Errorf("unexpected command %T", cmd)
	}

	return err
}

func (i *InventoryItem) On(e eventsource.Event) error {
	switch v := e.(type) {
	case *InventoryItemCreated:
		i.ID = v.AggregateID()
		i.currentState = &ItemState{
			Name:     v.Name,
			Quantity: v.Quantity,
		}
	case *InventoryItemAdjusted:
		i.ID = v.AggregateID()
		if v.Quantity >= 0 {
			i.currentState.Quantity += v.Quantity
		} else {
			i.currentState.Quantity -= v.Quantity
		}
	default:
		return fmt.Errorf("unexpected event %T", e)
	}

	return nil
}

// Commands
type CreateInventoryItem struct {
	eventsource.Command
	Name     string
	Quantity int
}
type AdjustInventoryItem struct {
	eventsource.Command
	Quantity int
}

func (CreateInventoryItem) New() bool { return true }

func main() {
	marshaler := new(eventsource.JsonEventMarshaler)
	marshaler.Bind(InventoryItemCreated{}, InventoryItemAdjusted{})

	var (
		ctx  = context.Background()
		item = &InventoryItem{}
		repo = eventsource.NewRepository(item, eventsource.WithMarshaler(marshaler),
			eventsource.WithDefaultSnapRepository(ItemState{}))
		bus = eventsource.NewCommandBus(repo)
	)

	bus.Send(ctx, CreateInventoryItem{eventsource.Command{ID: "abc123"}, "xyz", 1})
	bus.Send(ctx, AdjustInventoryItem{eventsource.Command{ID: "abc123"}, 3}) // first snap taken here
	bus.Send(ctx, AdjustInventoryItem{eventsource.Command{ID: "abc123"}, 2})
	bus.Send(ctx, AdjustInventoryItem{eventsource.Command{ID: "abc123"}, 2}) // second snap taken here

	aggregate, err := repo.GetByID(ctx, "abc123")
	checkErr(err)

	item = aggregate.(*InventoryItem)
	fmt.Printf("Item sku: %#v\n", item)
	fmt.Printf("Item State: %#v\n", item.currentState)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
