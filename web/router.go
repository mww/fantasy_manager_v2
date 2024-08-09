package web

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mww/fantasy_manager_v2/controller"
	"github.com/unrolled/render"
)

func getRouter(ctrl controller.C, render *render.Render) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(10 * time.Second))

	r.Get("/", rootHandler(ctrl, render))

	r.Route("/players", func(r chi.Router) {
		// Show either the search page if the q parameter is not present, or perform
		// the search if it is.
		r.Get("/", playerSearchHandler(ctrl, render))
		r.Get("/{playerID:\\d+}", getPlayerHandler(ctrl, render))
		r.Post("/{playerID:\\d+}", updatePlayerHandler(ctrl, render))

		r.Route("/rankings", func(r chi.Router) {
			r.Get("/", rankingsRootHandler(ctrl, render))
			r.Post("/", rankingsUploadHandler(ctrl, render))
			r.Get("/{rankingID:\\d+}", rankingsHandler(ctrl, render))
		})
	})

	r.Route("/admin", func(r chi.Router) {
		r.Use(middleware.BasicAuth("ff", map[string]string{"admin": "pa55word"})) // TODO: read from DB instead
		r.Use(middleware.Timeout(30 * time.Second))                               // Set a longer timeout for /admin actions

		r.Post("/players", forceUpdatePlayers(ctrl, render))
	})

	return r
}
