package api

import (
	"github.com/go-chi/chi/v5"

	"github.com/getflaggy/flaggy/internal/sse"
	"github.com/getflaggy/flaggy/internal/store"
)

// Server holds dependencies for all HTTP handlers.
type Server struct {
	store       store.Store
	broadcaster *sse.Broadcaster
}

// NewRouter creates a Chi router with all routes wired.
// masterKey protects admin routes. If empty, auth is disabled (dev mode).
func NewRouter(s store.Store, b *sse.Broadcaster, masterKey string, corsEnabled bool) *chi.Mux {
	srv := &Server{store: s, broadcaster: b}

	r := chi.NewRouter()
	r.Use(RequestLogger)
	if corsEnabled {
		r.Use(CORS)
	}

	r.Route("/api/v1", func(r chi.Router) {
		// Admin routes — protected by master key
		r.Group(func(r chi.Router) {
			r.Use(RequireMasterKey(masterKey))

			// Flags CRUD
			r.Post("/flags", srv.CreateFlag)
			r.Get("/flags", srv.ListFlags)
			r.Get("/flags/{key}", srv.GetFlag)
			r.Put("/flags/{key}", srv.UpdateFlag)
			r.Delete("/flags/{key}", srv.DeleteFlag)
			r.Patch("/flags/{key}/toggle", srv.ToggleFlag)

			// Rules CRUD
			r.Post("/flags/{key}/rules", srv.CreateRule)
			r.Put("/flags/{key}/rules/{ruleID}", srv.UpdateRule)
			r.Delete("/flags/{key}/rules/{ruleID}", srv.DeleteRule)

			// Segments CRUD
			r.Post("/segments", srv.CreateSegment)
			r.Get("/segments", srv.ListSegments)
			r.Get("/segments/{key}", srv.GetSegment)
			r.Put("/segments/{key}", srv.UpdateSegment)
			r.Delete("/segments/{key}", srv.DeleteSegment)

			// API Keys management
			r.Post("/api-keys", srv.CreateAPIKey)
			r.Get("/api-keys", srv.ListAPIKeys)
			r.Delete("/api-keys/{id}", srv.RevokeAPIKey)
		})

		// Client routes — protected by API key (or master key)
		r.Group(func(r chi.Router) {
			r.Use(RequireAPIKey(s, masterKey))

			r.Post("/evaluate", srv.Evaluate)
			r.Post("/evaluate/batch", srv.EvaluateBatch)
		})

		// SSE Stream — protected by API key (or master key)
		r.Group(func(r chi.Router) {
			r.Use(RequireAPIKey(s, masterKey))

			r.Get("/stream", srv.Stream)
		})
	})

	return r
}
