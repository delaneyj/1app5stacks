package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/t3dotgg/1app5stacks/go-datastar/sql"
	"github.com/t3dotgg/1app5stacks/go-datastar/web"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	// Get port from environment variable, default to 4321
	port := 4321
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	log.Printf("Starting server on port %d", port)

	db, err := sql.New(ctx)
	if err != nil {
		return fmt.Errorf("error creating database: %w", err)
	}
	defer db.Close()

	return web.RunBlocking(db, port)(ctx)
}
