package eventsource

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

var ErrSnapNotFound = errors.New("snapshot not found")

func NewInmemEventStore() EventStore {
	return &inmemEventStore{persistence: make(map[string]History)}
}

// inmemEventStore is an implementation of eventsource.EventStore
// can be useful for testing or examples.
type inmemEventStore struct {
	mu          sync.Mutex
	persistence map[string]History
}

func (s *inmemEventStore) SaveEvents(ctx context.Context, agrID string, models History, version int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	i := version

	var history History
	for _, m := range models {
		m.Version = i
		history = append(s.persistence[agrID], m)
		i++
	}

	sort.Slice(history, func(i, j int) bool { return history[i].Version < history[j].Version })

	s.persistence[agrID] = history

	return nil
}

func (s *inmemEventStore) GetEventsForAggregate(ctx context.Context, agrID string, version int) (History, error) {
	history, ok := s.persistence[agrID]
	if !ok {
		return nil, fmt.Errorf("no aggregate found with id: %s", agrID)
	}

	if version != 0 {
		var h History
		for _, m := range history {
			if m.Version > version {
				h = append(h, m)
			}
		}

		return h, nil
	}

	return history, nil
}

func NewInmemSnapStore() SnapshotStore {
	return &inmemSnapStore{persistence: make(map[string][]SnapshotModel)}
}

type inmemSnapStore struct {
	mu          sync.Mutex
	persistence map[string][]SnapshotModel
}

func (r *inmemSnapStore) SaveSnapshot(ctx context.Context, agrID string, model SnapshotModel, version int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	history := append(r.persistence[model.ID], model)
	sort.Slice(history, func(i, j int) bool { return history[i].Version < history[j].Version })

	r.persistence[model.ID] = history

	return nil
}

func (s *inmemSnapStore) GetSnapshotForAggregate(ctx context.Context, agrID string, version int) (SnapshotModel, error) {

	history, ok := s.persistence[agrID]
	if !ok {
		return SnapshotModel{}, ErrSnapNotFound
	}

	if len(history) == 0 {
		return SnapshotModel{}, ErrSnapNotFound
	}

	return history[len(history)-1], nil
}
