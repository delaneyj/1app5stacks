package web

import (
	"compress/gzip"
	"compress/zlib"
	"context"
	"embed"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/CAFxX/httpcompression"
	"github.com/andybalholm/brotli"
	"github.com/benbjohnson/hashfs"
	"github.com/delaneyj/toolbelt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

//go:embed static/*
var staticFS embed.FS

var (
	staticSys             = hashfs.NewFS(staticFS)
	compressionMiddleware func(http.Handler) http.Handler
)

func init() {
	opts := []httpcompression.Option{
		httpcompression.DeflateCompressionLevel(zlib.DefaultCompression),
		httpcompression.GzipCompressionLevel(gzip.DefaultCompression),
		httpcompression.BrotliCompressionLevel(brotli.DefaultCompression),
	}
	var err error
	compressionMiddleware, err = httpcompression.Adapter(opts...)
	if err != nil {
		panic(err)
	}
}

// VoteEvent represents a voting event that occurred
type VoteEvent struct {
	UpvotedID   int64
	DownvotedID int64
}

func staticPath(path string) string {
	return "/" + staticSys.HashName("static/"+path)
}

func RunBlocking(db *toolbelt.Database, port int) toolbelt.CtxErrFunc {
	return func(ctx context.Context) (err error) {
		router := chi.NewRouter()
		router.Use(
			middleware.Recoverer,
			compressionMiddleware,
		)
		router.Handle("/static/*", hashfs.FileServer(staticSys))

		router.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/vote", http.StatusSeeOther)
		})

		voteEventBus := toolbelt.NewEventBusAsync[VoteEvent]()

		if err := errors.Join(

			setupVoteRoutes(router, db, voteEventBus),
			setupResultsRoutes(router, db, voteEventBus),
			setupSpriteRoutes(router, db),
		); err != nil {
			return fmt.Errorf("error setting up routes: %w", err)
		}

		srv := &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: router,
		}
		go func() {
			<-ctx.Done()
			srv.Shutdown(context.Background())
		}()

		log.Printf("Hosting on http://localhost:%d", port)
		return srv.ListenAndServe()
	}
}
