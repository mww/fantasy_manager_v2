package web

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mww/fantasy_manager_v2/controller"
	"github.com/mww/fantasy_manager_v2/db"
	"github.com/mww/fantasy_manager_v2/model"
	"github.com/unrolled/render"
)

func rootHandler(ctrl *controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render.Text(w, http.StatusOK, "root page")
	}
}

func playerSearchHandler(ctrl *controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")

		var err error
		var results []model.Player = nil
		if query != "" {
			results, err = ctrl.Search(r.Context(), query)
			if err != nil {
				render.HTML(w, http.StatusInternalServerError, "500", err.Error())
				return
			}
		}

		data := map[string]any{
			"q":       query,
			"results": results,
		}
		render.HTML(w, http.StatusOK, "playerSearch", data)
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

func updatePlayerHandler(ctrl *controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			render.HTML(w, http.StatusBadRequest, "400", err.Error())
			return
		}

		playerID := chi.URLParam(r, "playerID")

		updating := r.PostForm.Get("update")
		if updating == "nickname" {
			nn := r.PostForm.Get("nickname")
			err := ctrl.UpdatePlayerNickname(r.Context(), playerID, nn)
			if err != nil {
				render.HTML(w, http.StatusInternalServerError, "500", err.Error())
				return
			}
		} else {
			render.HTML(w, http.StatusBadRequest, "400", fmt.Sprintf("unknown update type: %s", updating))
			return
		}

		// Now fetch the updated player and render
		p, err := ctrl.GetPlayer(r.Context(), playerID)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err.Error())
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
