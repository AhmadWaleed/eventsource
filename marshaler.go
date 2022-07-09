package eventsource

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type payload struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type JsonEventMarshaler struct {
	eventTypes map[string]reflect.Type
}

func (m *JsonEventMarshaler) Bind(events ...Event) error {
	m.eventTypes = make(map[string]reflect.Type, len(events))

	for _, e := range events {
		name, typ := getType(e)
		m.eventTypes[name] = typ
	}

	return nil
}

func (m *JsonEventMarshaler) Marshal(e Event) (EventModel, error) {
	typ, _ := getType(e)

	data, err := json.Marshal(e)
	if err != nil {
		return EventModel{}, err
	}

	data, err = json.Marshal(payload{
		Type: typ,
		Data: json.RawMessage(data),
	})
	if err != nil {
		return EventModel{}, err
	}

	model := EventModel{
		Version: e.EventVersion(),
		At:      Time(e.EventAt()),
		Data:    data,
	}

	return model, nil
}

func (m *JsonEventMarshaler) Unmarshal(model EventModel) (Event, error) {
	var p payload
	if err := json.Unmarshal(model.Data, &p); err != nil {
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

type JsonSnapshotMarshaler struct {
	states map[string]reflect.Type
}

func (m *JsonSnapshotMarshaler) Bind(states ...interface{}) error {
	m.states = make(map[string]reflect.Type, len(states))

	for _, s := range states {
		name, typ := getType(s)
		m.states[name] = typ
	}

	return nil
}

func (m *JsonSnapshotMarshaler) Marshal(s Snapshot) (SnapshotModel, error) {
	typ, _ := getType(s.GetState())

	data, err := json.Marshal(s.GetState())
	if err != nil {
		return SnapshotModel{}, err
	}

	data, err = json.Marshal(payload{
		Type: typ,
		Data: json.RawMessage(data),
	})
	if err != nil {
		return SnapshotModel{}, err
	}

	model := SnapshotModel{
		ID:      s.AggregateRootID(),
		Version: s.CurrentVersion(),
		Data:    data,
	}

	return model, nil
}

func (m *JsonSnapshotMarshaler) Unmarshal(model SnapshotModel) (Snapshot, error) {
	var p payload
	if err := json.Unmarshal(model.Data, &p); err != nil {
		return nil, err
	}

	typ, ok := m.states[p.Type]
	if !ok {
		return nil, fmt.Errorf("unable to unmarshal type %v", p)
	}

	v := reflect.New(typ).Interface()
	if err := json.Unmarshal(p.Data, v); err != nil {
		return nil, fmt.Errorf("unable to unmarshal snapshot data into %#v, %v", v, err)
	}

	snap := SnapshotSkeleton{
		ID:      model.ID,
		Version: model.Version,
		State:   v,
	}

	return snap, nil
}

func getType(v interface{}) (string, reflect.Type) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t.Name(), t
}
