package eventsource

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

type History []Model

// Model provides the shape of the records to be saved to the db
type Model struct {
	// Version is the event version the Data represents
	Version int

	// At indicates when the event happened; provided as a utility for the store
	At EpochMillis

	// Data contains the Serializer encoded version of the data
	Data []byte
}

func NewInmemStore() EventStore {
	return &inmemstore{persistence: make(map[string]History)}
}

// inmemstore is an implementation of eventsource.EventStore
// can be useful for testing or examples.
type inmemstore struct {
	mu          sync.Mutex
	persistence map[string]History
}

func (s *inmemstore) SaveEvents(ctx context.Context, agrID string, models History, version int) error {
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

func (s *inmemstore) GetEventsForAggregate(ctx context.Context, agrID string, version int) (History, error) {
	history, ok := s.persistence[agrID]
	if !ok {
		return nil, fmt.Errorf("no aggregate found with id: %s", agrID)
	}

	return history, nil
}
