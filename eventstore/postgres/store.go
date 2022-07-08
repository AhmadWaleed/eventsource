package eventstore

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"

	"github.com/AhmadWaleed/eventsource"
)

func NewStore(db *sql.DB, table string) eventsource.EventStore {
	return &store{
		db:    db,
		table: table,
	}
}

func CreateEventStoreTable(ctx context.Context, db *sql.DB, table string) error {
	sql := `
	CREATE TABLE IF NOT EXISTS %s (
	    "offset"    BIGSERIAL PRIMARY KEY NOT NULL,
	    id        VARCHAR(255) NOT NULL,
	    version   INTEGER NOT NULL,
	    data      JSON NOT NULL,
	    at        BIGINT NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_%s ON %s (id, version);
`
	_, err := db.ExecContext(ctx, fmt.Sprintf(sql, table, table, table))
	return err
}

type store struct {
	db    *sql.DB
	table string
}

func (s *store) SaveEvents(ctx context.Context, agrID string, models eventsource.History, version int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(fmt.Sprintf(`INSERT INTO %s (id, version, data, at) VALUES ($1, $2, $3, $4)`, s.table))
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, model := range models {
		_, err = stmt.ExecContext(ctx, agrID, model.Version, model.Data, model.At)
		if err != nil {
			return err
		}
	}

	if err != nil {
		return tx.Rollback()
	}

	return tx.Commit()
}

func (s *store) GetEventsForAggregate(ctx context.Context, agrID string, version int) (eventsource.History, error) {
	if version == 0 {
		version = math.MaxInt32
	}

	sql := fmt.Sprintf(`SELECT version, data, at FROM %s WHERE id = $1 and version <= $2`, s.table)
	rows, err := s.db.QueryContext(ctx, sql, agrID, version)
	if err != nil {
		return eventsource.History{}, err
	}
	defer rows.Close()

	history := make(eventsource.History, 0, version+1)
	for rows.Next() {
		var rec eventsource.Model
		err := rows.Scan(&rec.Version, &rec.Data, &rec.At)
		if err != nil {
			return eventsource.History{}, err
		}

		history = append(history, rec)
	}

	sort.Slice(history, func(x, y int) bool {
		return history[x].Version < history[y].Version
	})

	return history, nil
}
