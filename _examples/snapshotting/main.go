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
	sku          string
	currentState *ItemState
	streamSize   int
	stream       []eventsource.Event
	version      int
}

func (i *InventoryItem) StreamSize() int {
	return i.streamSize
}

func (i *InventoryItem) SnapshotInterval() int {
	return 2
}

func (i *InventoryItem) GetState() interface{} {
	return i.currentState
}

func (i *InventoryItem) ApplyState(s eventsource.Snapshot) {
	i.sku = s.AggregateRootID()
	i.version = s.CurrentVersion()
	i.currentState = s.GetState().(*ItemState)
}

func (i *InventoryItem) SnapshottingEnable() bool {
	return true
}

func (i *InventoryItem) AggregateRootID() string {
	return i.sku
}

func (i *InventoryItem) Version() int {
	return i.version
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
		err = i.Apply(e, true)
	case AdjustInventoryItem:
		e := &InventoryItemAdjusted{
			eventsource.EventSkeleton{ID: cmd.AggregateID(), At: time.Now()},
			cmd.Quantity,
		}
		err = i.Apply(e, true)
	default:
		err = fmt.Errorf("unexpected command %T", cmd)
	}

	return err
}

func (i *InventoryItem) Apply(e eventsource.Event, isNew bool) error {
	switch v := e.(type) {
	case *InventoryItemCreated:
		i.sku = v.AggregateID()
		i.currentState = &ItemState{
			Name:     v.Name,
			Quantity: v.Quantity,
		}
	case *InventoryItemAdjusted:
		i.sku = v.AggregateID()
		if v.Quantity >= 0 {
			i.currentState.Quantity += v.Quantity
		} else {
			i.currentState.Quantity -= v.Quantity
		}
	default:
		return fmt.Errorf("unexpected event %T", e)
	}

	if isNew {
		i.stream = append(i.stream, e)
		if _, ok := e.(eventsource.Constructor); !ok {
			i.version++
		}
	}

	i.streamSize++

	return nil
}

func (i *InventoryItem) LoadFromHistory(history []eventsource.Event) error {
	for _, e := range history {
		if err := i.Apply(e, false); err != nil {
			return err
		}
	}

	i.version = len(history) - 1

	return nil
}

func (i *InventoryItem) GetUncommitedEvents() []eventsource.Event {
	return i.stream
}

func (i *InventoryItem) CommitEvents() {
	i.stream = []eventsource.Event{}
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
		item = &InventoryItem{sku: "abc123"}
		repo = eventsource.NewRepository(item, eventsource.WithMarshaler(marshaler),
			eventsource.WithDefaultSnapRepository(ItemState{}))
		bus = eventsource.NewCommandBus(repo)
	)

	bus.Send(ctx, CreateInventoryItem{eventsource.Command{ID: item.AggregateRootID()}, "xyz", 1})
	bus.Send(ctx, AdjustInventoryItem{eventsource.Command{ID: item.AggregateRootID()}, 3}) // first snap taken here
	bus.Send(ctx, AdjustInventoryItem{eventsource.Command{ID: item.AggregateRootID()}, 2})
	bus.Send(ctx, AdjustInventoryItem{eventsource.Command{ID: item.AggregateRootID()}, 2}) // second snap taken here

	aggregate, err := repo.GetByID(ctx, item.AggregateRootID())
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
