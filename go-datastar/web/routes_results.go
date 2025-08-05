package web

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/delaneyj/toolbelt"
	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
	"github.com/t3dotgg/1app5stacks/go-datastar/sql/zz"
	"zombiezen.com/go/sqlite"
)

func setupResultsRoutes(r chi.Router, db *toolbelt.Database, voteEventBus *toolbelt.EventBusAsync[VoteEvent]) error {
	r.Route("/results", func(resultsRouter chi.Router) {
		resultsRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
			ResultsPage().Render(r.Context(), w)
		})

		resultsRouter.Get("/stream", func(w http.ResponseWriter, r *http.Request) {
			sse := datastar.NewSSE(w, r, datastar.WithCompression())
			ctx := r.Context()

			updateResults := toolbelt.Throttle(100*time.Millisecond, func(ctx context.Context) error {
				var rows []zz.ResultsRes
				if err := db.ReadTX(ctx, func(tx *sqlite.Conn) (err error) {
					rows, err = zz.OnceResults(tx)
					return err
				}); err != nil {
					return fmt.Errorf("failed to fetch results: %w", err)
				}
				return sse.PatchElementTempl(resultRows(rows...))
			})

			unsubscribe := voteEventBus.Subscribe(ctx, func(event VoteEvent) error {
				return updateResults(ctx)
			})
			defer unsubscribe()

			updateResults(ctx)

			<-ctx.Done()
		})
	})
	return nil
}
