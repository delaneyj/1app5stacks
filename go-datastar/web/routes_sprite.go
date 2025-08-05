package web

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/delaneyj/toolbelt"
	"github.com/go-chi/chi/v5"
	"github.com/t3dotgg/1app5stacks/go-datastar/sql/zz"
	"zombiezen.com/go/sqlite"
)

func spriteRoute(id int64) string {
	return fmt.Sprintf("/sprites/%d", id)
}

func setupSpriteRoutes(r chi.Router, db *toolbelt.Database) error {
	r.Get("/sprites/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid pokemon ID", http.StatusBadRequest)
			return
		}

		var spriteData []byte
		var contentType string

		// Get sprite from database (all sprites are fetched at startup)
		if err := db.ReadTX(r.Context(), func(tx *sqlite.Conn) error {
			res, err := zz.OnceGetPokemonSprite(tx, id)
			if err != nil {
				return err
			}
			if res == nil || res.SpriteData == nil {
				return fmt.Errorf("sprite not found")
			}

			spriteData = *res.SpriteData
			if res.SpriteContentType != nil {
				contentType = *res.SpriteContentType
			} else {
				contentType = "image/png"
			}
			return nil
		}); err != nil {
			if err.Error() == "sprite not found" {
				http.Error(w, "Sprite not found", http.StatusNotFound)
			} else {
				http.Error(w, "Database error", http.StatusInternalServerError)
			}
			return
		}

		// Set caching headers - sprites are immutable
		h := w.Header()
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
		h.Set("Content-Type", contentType)
		h.Set("Content-Length", strconv.Itoa(len(spriteData)))

		// Write sprite data
		io.Copy(w, bytes.NewReader(spriteData))
	})

	return nil
}
