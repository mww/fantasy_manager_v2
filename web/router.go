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
		r.Get("/{playerID:\\w+}", getPlayerHandler(ctrl, render))
		r.Post("/{playerID:\\w+}", updatePlayerHandler(ctrl, render))

		r.Route("/rankings", func(r chi.Router) {
			r.Get("/", rankingsRootHandler(ctrl, render))
			r.Post("/", rankingsUploadHandler(ctrl, render))
			r.Get("/{rankingID:\\d+}", rankingsHandler(ctrl, render))
		})
	})

	r.Route("/leagues", func(r chi.Router) {
		r.Get("/{leagueID:\\d+}", getLeagueHandler(ctrl, render))
		r.Post("/{leagueID:\\d+}/managers", refreshLeagueManagersHandler(ctrl, render))
		r.Post("/{leagueID:\\d+}/results/sync", syncWeekResultsHandler(ctrl, render))
		r.Get("/{leagueID:\\d+}/week/{week:\\d+}", getLeagueResultsHandler(ctrl, render))
		r.Get("/{leagueID:\\d+}/power", getPowerRankingsHandler(ctrl, render))
		r.Post("/{leagueID:\\d+}/power", createPowerRankingsHandler(ctrl, render))
		r.Get("/{leagueID:\\d+}/power/{powerRankingID:\\d+}", showPowerRankingHandler(ctrl, render))
		r.Get("/{leagueID:\\d+}/power/{powerRankingID:\\d+}/text", showPowerRankingsTextHandler(ctrl, render))
		r.Get("/platformLeagues", platformLeaguesHandler(ctrl, render))
		r.Get("/", leaguesHandler(ctrl, render))
		r.Post("/", leaguesPostHandler(ctrl, render))
	})

	r.Route("/oauth", func(r chi.Router) {
		r.Get("/link", oauthLinkHandler(ctrl, render))
		r.Get("/redirect", oauthRedirectHandler(ctrl, render))
	})

	r.Route("/admin", func(r chi.Router) {
		r.Use(middleware.BasicAuth("ff", map[string]string{"admin": "pa55word"})) // TODO: read from DB instead
		r.Use(middleware.Timeout(30 * time.Second))                               // Set a longer timeout for /admin actions

		r.Post("/players", forceUpdatePlayers(ctrl, render))
	})

	return r
}
