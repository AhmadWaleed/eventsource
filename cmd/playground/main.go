package main

import (
	"context"

	_ "github.com/lib/pq"
)

const DSN = "postgres://postgres:abc123@postgres:5432/mr-eventstore?sslmode=disable&search_path=public"

var ctx = context.Background()

func main() {
	// db, err := sql.Open("postgres", DSN)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// record := store.Record{Version: 1, At: (time.Now().UnixNano() / int64(time.Millisecond)), Data: []byte(`{}`)}
	// fmt.Printf("%+v", record)

	// storage := eventstore.NewStore(db, "eventstore")
	// if err := storage.SaveEvents(ctx, new(eventsource.AggregateRootUUID), []store.Record{record}); err != nil {
	// 	log.Fatal(err)
	// }

	// history, err := storage.GetEventsForAggregate(ctx, new(eventsource.AggregateRootUUID), 1)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(history)

	// if err := eventstore.CreateEventStoreTable(ctx, db, "eventstore"); err != nil {
	// 	log.Fatal(err)
	// }
}
