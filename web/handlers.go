package web

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mww/fantasy_manager_v2/controller"
	"github.com/mww/fantasy_manager_v2/db"
	"github.com/unrolled/render"
)

func rootHandler(ctrl *controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render.Text(w, http.StatusOK, "root page")
	}
}

func playerSearchHandler(ctrl *controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render.Text(w, http.StatusOK, "search page here")
	}
}

func getPlayerHandler(ctrl *controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := chi.URLParam(r, "playerID")
		p, err := ctrl.GetPlayer(r.Context(), playerID)
		if err != nil {
			if errors.Is(err, db.ErrPlayerNotFound) {
				render.HTML(w, http.StatusNotFound, "404", "player not found")
			} else {
				render.HTML(w, http.StatusInternalServerError, "500", err.Error())
			}
			return
		}

		render.HTML(w, http.StatusOK, "player", p)
	}
}

func forceUpdatePlayers(ctrl *controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := ctrl.UpdatePlayers(r.Context()); err != nil {
			render.Text(w, http.StatusInternalServerError, fmt.Sprintf("error updating players: %v", err))
			return
		}

		render.Text(w, http.StatusOK, "update players completed successfully")
	}
}
