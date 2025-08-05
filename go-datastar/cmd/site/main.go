package main

import (
	"context"
	"fmt"
	"log"

	"github.com/t3dotgg/1app5stacks/go-datastar/sql"
	"github.com/t3dotgg/1app5stacks/go-datastar/web"
)

const port = 4321

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	db, err := sql.New(ctx)
	if err != nil {
		return fmt.Errorf("error creating database: %w", err)
	}
	defer db.Close()

	return web.RunBlocking(db, port)(ctx)

}
