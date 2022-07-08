package eventsource

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type JsonMarshaller struct {
	eventTypes map[string]reflect.Type
}

func (m *JsonMarshaller) Bind(events ...Event) error {
	m.eventTypes = make(map[string]reflect.Type, len(events))

	for _, e := range events {
		name, typ := getType(e)
		m.eventTypes[name] = typ
	}

	return nil
}

type payload struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func (m *JsonMarshaller) Marshal(e Event) (Model, error) {
	typ, _ := getType(e)

	data, err := json.Marshal(e)
	if err != nil {
		return Model{}, err
	}

	data, err = json.Marshal(payload{
		Type: typ,
		Data: json.RawMessage(data),
	})
	if err != nil {
		return Model{}, err
	}

	model := Model{
		Version: e.EventVersion(),
		At:      Time(e.EventAt()),
		Data:    data,
	}

	return model, nil
}

func (m *JsonMarshaller) Unmarshal(r Model) (Event, error) {
	var p payload
	if err := json.Unmarshal(r.Data, &p); err != nil {
		return nil, err
	}

	typ, ok := m.eventTypes[p.Type]
	if !ok {
		return nil, fmt.Errorf("unable to unmarshal type %v", p)
	}

	v := reflect.New(typ).Interface()
	if err := json.Unmarshal(p.Data, v); err != nil {
		return nil, fmt.Errorf("unable to unmarshal event data into %#v, %v", v, err)
	}

	return v.(Event), nil
}

func getType(event Event) (string, reflect.Type) {
	t := reflect.TypeOf(event)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t.Name(), t
}
