package main

import (
	"context"
	"fmt"
	"time"

	"github.com/AhmadWaleed/eventsource"
	"github.com/AhmadWaleed/eventsource/command"
	"github.com/AhmadWaleed/eventsource/command/bus"
)

type CustomerCreated struct {
	eventsource.EventSkeleton
	Name  string
	Email string
}

type CustomerEmailChanged struct {
	eventsource.EventSkeleton
	Email string
}

type CustomerCreatedHandler struct{}

func (h *CustomerCreatedHandler) Handle(ctx context.Context, v interface{}) error {
	e, _ := v.(*CustomerCreated)
	_, err := fmt.Printf("Handled: %#v\n", e)
	return err
}

type CustomerEmailChangedHandler struct{}

func (h *CustomerEmailChangedHandler) Handle(ctx context.Context, v interface{}) error {
	e, _ := v.(*CustomerEmailChanged)
	_, err := fmt.Printf("Handled: %#v\n", e)
	return err
}

type CreateCustomer struct {
	eventsource.Command
	Name  string
	Email string
}

func (CreateCustomer) New() bool { return true }

type ChangeCustomerEmail struct {
	eventsource.Command
	Email string
}

type CreateCustomerHandler struct {
	bus        command.EventPublisher
	repository eventsource.AggregateRootRepository
}

func (h *CreateCustomerHandler) Handle(ctx context.Context, v interface{}) error {
	c, _ := v.(CreateCustomer)

	customer := new(Customer)
	event := &CustomerCreated{
		eventsource.EventSkeleton{ID: c.AggregateID(), At: time.Now()},
		c.Name,
		c.Email,
	}

	if err := customer.Apply(customer, event, true); err != nil {
		return err
	}

	if err := h.repository.Save(ctx, customer); err != nil {
		return fmt.Errorf("could not save aggregate %T %v", customer, err)
	}

	return h.bus.Publish(ctx, event)
}

type ChangeCustomerEmailHandler struct {
	bus        command.EventPublisher
	repository eventsource.AggregateRootRepository
}

func (h *ChangeCustomerEmailHandler) Handle(ctx context.Context, v interface{}) error {
	c, _ := v.(ChangeCustomerEmail)

	aggregateID := c.AggregateID()

	aggregate, err := h.repository.GetByID(ctx, aggregateID)
	if err != nil {
		return fmt.Errorf("unable to get aggregate by ID: %v", err)
	}

	customer, _ := aggregate.(*Customer)
	event := &CustomerEmailChanged{
		eventsource.EventSkeleton{ID: c.AggregateID(), At: time.Now()},
		c.Email,
	}

	if err := customer.Apply(customer, event, true); err != nil {
		return err
	}

	err = h.repository.Save(ctx, customer)
	if err != nil {
		return fmt.Errorf("could not save aggregate %T %v", customer, err)
	}

	return h.bus.Publish(ctx, &CustomerEmailChanged{
		Email: c.Email,
	})
}

type Customer struct {
	eventsource.AggregateRootBase
	Name  string
	Email string
}

func (c *Customer) On(e eventsource.Event) error {
	switch e := e.(type) {
	case *CustomerCreated:
		c.ID = e.AggregateID()
		c.Name = e.Name
		c.Email = e.Email
	case *CustomerEmailChanged:
		c.ID = e.AggregateID()
		c.Email = e.Email
	default:
		return fmt.Errorf("unexpected event: %T", e)
	}

	return nil
}

func main() {
	ctx := context.Background()

	marshaler := new(eventsource.JsonEventMarshaler)
	marshaler.Bind(CustomerCreated{}, CustomerEmailChanged{})

	customer := &Customer{}
	repo := eventsource.NewRepository(customer, eventsource.WithMarshaler(marshaler))

	bus := bus.NewSyncBus()
	bus.Bind(CreateCustomer{}, []command.Handler{&CreateCustomerHandler{bus: bus, repository: repo}})
	bus.Bind(ChangeCustomerEmail{}, []command.Handler{&ChangeCustomerEmailHandler{bus: bus, repository: repo}})
	bus.Bind(CustomerCreated{}, []command.Handler{&CustomerCreatedHandler{}})
	bus.Bind(CustomerEmailChanged{}, []command.Handler{&CustomerEmailChangedHandler{}})

	err := bus.Send(ctx, CreateCustomer{
		eventsource.Command{ID: "abc123"},
		"John",
		"j.doe@example.com",
	})
	checkErr(err)

	err = bus.Send(ctx, ChangeCustomerEmail{
		eventsource.Command{ID: "abc123"},
		"j.foo@example.com",
	})
	checkErr(err)

	aggregate, err := repo.GetByID(ctx, "abc123")
	checkErr(err)

	fmt.Printf("%#v", aggregate)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
