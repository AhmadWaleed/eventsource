package eventsource

import (
	"fmt"
	"reflect"
)

type EventType string
type EventHandler func(e Event) error

var NopHandler = func(e Event) error { return nil }

type EventRegistry map[EventType]EventHandler
type Registry struct {
	registry EventRegistry
}

func (i *Registry) GetHandler(e Event) EventHandler {
	et := reflect.TypeOf(e)
	if et.Kind() == reflect.Ptr {
		et = et.Elem()
	}

	if handler, ok := i.registry[EventType(et.Name())]; ok {
		return handler
	}

	return NopHandler
}

func (i *Registry) BindDynamicHandlers(v interface{}, events ...Event) error {
	i.registry = make(EventRegistry)
	for _, event := range events {
		typ, handler, err := buildHandler(v, event)
		if err != nil {
			return err
		}

		i.registry[EventType(typ)] = handler
	}

	return nil
}

func buildHandler(v interface{}, e Event) (string, EventHandler, error) {
	et := reflect.TypeOf(e)
	if et.Kind() == reflect.Ptr {
		et = et.Elem()
	}

	vv := reflect.ValueOf(v)
	name := fmt.Sprintf("Apply%s", et.Name())
	mv := vv.MethodByName(name)
	if mv.IsZero() {
		return "", nil, fmt.Errorf("undefined handler method %s", name)
	}

	handler := func(e Event) error {
		out := mv.Call([]reflect.Value{reflect.ValueOf(e)})
		if len(out) != 1 {
			return fmt.Errorf("expect handler method %s to return only one value (nil|error)", name)
		}

		err := out[0]
		if err.IsNil() {
			return nil
		}

		return err.Interface().(error)
	}

	return et.Name(), handler, nil
}
