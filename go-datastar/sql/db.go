package sql

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"embed"

	"github.com/delaneyj/toolbelt"
	"github.com/goccy/go-json"
	"github.com/t3dotgg/1app5stacks/go-datastar/sql/zz"
	"github.com/valyala/bytebufferpool"
	"zombiezen.com/go/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func New(ctx context.Context) (*toolbelt.Database, error) {
	migrations, err := toolbelt.MigrationsFromFS(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("err creating migrations: %w", err)
	}

	db, err := toolbelt.NewDatabase(ctx, "pokemon.sqlite", migrations)
	if err != nil {
		return nil, fmt.Errorf("err creating database: %w", err)
	}

	var isEmpty bool
	if err := db.ReadTX(ctx, func(tx *sqlite.Conn) error {
		count, err := zz.OnceCountPokemon(tx)
		if err != nil {
			return fmt.Errorf("error counting pokemon: %w", err)
		}
		isEmpty = count == 0
		return nil
	}); err != nil {
		return nil, fmt.Errorf("error checking if database is empty: %w", err)
	}

	if isEmpty {
		log.Print("Empty database, seeding")
		if err := Seed(db); err != nil {
			return nil, fmt.Errorf("error seeding database: %w", err)
		}
	}

	// Fetch missing sprites
	log.Print("Checking for missing sprites...")
	if err := FetchMissingSprites(ctx, db); err != nil {
		// Don't fail startup if sprite fetching fails
		log.Printf("Warning: Failed to fetch some sprites: %v", err)
	}

	return db, nil
}

func Seed(db *toolbelt.Database) error {
	const q = `
query GetAllPokemon {
	pokemon_v2_pokemon {
		id
		pokemon_v2_pokemonspecy {
			name
		}
	}
}`
	b, err := json.Marshal(map[string]string{"query": q})
	if err != nil {
		return fmt.Errorf("error marshalling query: %w", err)
	}
	r := bytes.NewReader(b)

	req, err := http.NewRequest("POST", "https://beta.pokeapi.co/graphql/v1beta", r)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer res.Body.Close()
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	if _, err := buf.ReadFrom(res.Body); err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("error response code: %d", res.StatusCode)
	}

	type GraphQLResponse struct {
		Data struct {
			Pokemon []struct {
				ID           int64 `json:"id"`
				PokemonSpecy struct {
					Name string `json:"name"`
				} `json:"pokemon_v2_pokemonspecy"`
			} `json:"pokemon_v2_pokemon"`
		} `json:"data"`
	}
	gqlRes := &GraphQLResponse{}
	if err := json.Unmarshal(buf.Bytes(), gqlRes); err != nil {
		return fmt.Errorf("error unmarshalling response: %w", err)
	}

	if err := db.WriteTX(context.Background(), func(tx *sqlite.Conn) error {
		insert := zz.CreatePokemon(tx)
		now := time.Now()
		for _, p := range gqlRes.Data.Pokemon {
			// https://github.com/t3dotgg/1app5stacks/blob/main/go-graphql-spa-version/go-gql-server/main.go#L144C108-L144C124
			if p.ID >= 1025 {
				continue
			}
			if err := insert.Run(&zz.PokemonModel{
				Id:         p.ID,
				Name:       p.PokemonSpecy.Name,
				DexId:      p.ID,
				InsertedAt: now,
				UpdatedAt:  now,
			}); err != nil {
				return fmt.Errorf("error inserting pokemon: %w", err)
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("error inserting pokemon: %w", err)
	}

	return nil
}

func RandomResToPokemonModel(p zz.RandomPokemonRes) *zz.PokemonModel {
	return &zz.PokemonModel{
		Id:         p.Id,
		Name:       p.Name,
		DexId:      p.DexId,
		UpVotes:    p.UpVotes,
		DownVotes:  p.DownVotes,
		InsertedAt: p.InsertedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

const PokemonSpriteURLFormat = `https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/%d.png`

func FetchMissingSprites(ctx context.Context, db *toolbelt.Database) error {
	var missingIds []int64
	if err := db.ReadTX(ctx, func(tx *sqlite.Conn) error {
		ids, err := zz.OncePokemonWithMissingSprites(tx)
		if err != nil {
			return err
		}
		missingIds = ids
		return nil
	}); err != nil {
		return fmt.Errorf("error getting missing sprites: %w", err)
	}

	if len(missingIds) == 0 {
		log.Print("All sprites already cached")
		return nil
	}

	log.Printf("Fetching %d missing sprites...", len(missingIds))

	// Use a worker pool for concurrent fetching
	const numWorkers = 10
	jobs := make(chan int64, len(missingIds))
	errors := make(chan error, len(missingIds))

	// Start workers
	for w := 0; w < numWorkers; w++ {
		go func() {
			for id := range jobs {
				if err := fetchAndStoreSprite(ctx, db, id); err != nil {
					errors <- fmt.Errorf("Pokemon %d: %w", id, err)
				} else {
					errors <- nil
				}
			}
		}()
	}

	// Send all IDs to workers
	for _, id := range missingIds {
		jobs <- id
	}
	close(jobs)

	// Collect results
	var failedCount int
	for i := 0; i < len(missingIds); i++ {
		if err := <-errors; err != nil {
			log.Printf("Failed to fetch sprite: %v", err)
			failedCount++
		}

		// Progress logging every 50 sprites
		if (i+1)%50 == 0 {
			log.Printf("Progress: %d/%d sprites fetched", i+1, len(missingIds))
		}
	}

	if failedCount > 0 {
		log.Printf("Finished fetching sprites. Failed: %d/%d", failedCount, len(missingIds))
	} else {
		log.Printf("Successfully fetched all %d sprites", len(missingIds))
	}

	return nil
}

func fetchAndStoreSprite(ctx context.Context, db *toolbelt.Database, id int64) error {
	spriteURL := fmt.Sprintf(PokemonSpriteURLFormat, id)
	resp, err := http.Get(spriteURL)
	if err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return fmt.Errorf("failed to read: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}

	spriteData := buf.Bytes()
	now := time.Now().UTC()

	return db.WriteTX(ctx, func(tx *sqlite.Conn) error {
		params := zz.UpdatePokemonSpriteParams{
			SpriteData:        &spriteData,
			SpriteContentType: &contentType,
			SpriteFetchedAt:   &now,
			Id:                id,
		}
		return zz.OnceUpdatePokemonSprite(tx, params)
	})
}
