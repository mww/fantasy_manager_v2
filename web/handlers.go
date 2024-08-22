package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mww/fantasy_manager_v2/controller"
	"github.com/mww/fantasy_manager_v2/db"
	"github.com/mww/fantasy_manager_v2/model"
	"github.com/unrolled/render"
)

func rootHandler(_ controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render.Text(w, http.StatusOK, "root page")
	}
}

func playerSearchHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
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

func getPlayerHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
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

func updatePlayerHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
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

func rankingsRootHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rankings, err := ctrl.ListRankings(r.Context())
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err)
			return
		}
		render.HTML(w, http.StatusOK, "rankingsUploadPage", rankings)
	}
}

func rankingsHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rankingsID := chi.URLParam(r, "rankingID")
		id, err := strconv.Atoi(rankingsID)
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", fmt.Sprintf("error parsing ranking id: %v", err))
			return
		}
		ranking, err := ctrl.GetRanking(r.Context(), int32(id))
		if err != nil {
			render.HTML(w, http.StatusNotFound, "404", fmt.Sprintf("ranking not found: %v", err))
			return
		}

		render.HTML(w, http.StatusOK, "rankings", ranking)
	}
}

func rankingsUploadHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse the multipart form. 5 << 20 specifices a maximum upload of 5 MB files.
		r.ParseMultipartForm(5 << 20)

		file, handler, err := r.FormFile("rankings-file")
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", err.Error())
			return
		}
		defer file.Close()

		if handler.Header.Get("Content-Type") != "text/csv" {
			msg := fmt.Sprintf("Only CSV files are supported. Got %s", handler.Header.Get("Content-Type"))
			render.HTML(w, http.StatusBadRequest, "400", msg)
			return
		}

		d := r.FormValue("rankings-date")
		t, err := time.Parse(time.DateOnly, d)
		if err != nil {
			msg := fmt.Sprintf("Unable to parse rankings date. Expected format is YYYY-MM-DD: %v", err)
			render.HTML(w, http.StatusBadRequest, "400", msg)
			return
		}

		id, err := ctrl.AddRanking(r.Context(), file, t)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err.Error())
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/players/rankings/%d", id), http.StatusSeeOther)
	}
}

func forceUpdatePlayers(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := ctrl.UpdatePlayers(r.Context()); err != nil {
			render.Text(w, http.StatusInternalServerError, fmt.Sprintf("error updating players: %v", err))
			return
		}

		render.Text(w, http.StatusOK, "update players completed successfully")
	}
}

func leaguesHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		leagues, err := ctrl.ListLeagues(r.Context())
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err.Error())
		}

		render.HTML(w, http.StatusOK, "leagues", leagues)
	}
}

func getLeagueHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		leagueID := chi.URLParam(r, "leagueID")
		id, err := strconv.Atoi(leagueID)
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", fmt.Sprintf("error parsing league id: %v", err))
			return
		}

		l, err := ctrl.GetLeague(r.Context(), int32(id))
		if err != nil {
			render.HTML(w, http.StatusNotFound, "404", err.Error())
			return
		}

		render.HTML(w, http.StatusOK, "league", l)
	}
}

func refreshLeagueManagersHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		leagueID := chi.URLParam(r, "leagueID")
		id, err := strconv.Atoi(leagueID)
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", fmt.Sprintf("error parsing league id: %v", err))
			return
		}

		_, err = ctrl.AddLeagueManagers(r.Context(), int32(id))
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err.Error())
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/leagues/%d", id), http.StatusSeeOther)
	}
}

func platformLeaguesHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		platform := r.URL.Query().Get("platform")
		username := r.URL.Query().Get("username")
		year := "2024"

		leagues, err := ctrl.GetLeaguesFromPlatform(r.Context(), username, platform, year)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err)
			return
		}

		data := map[string]any{
			"platform": platform,
			"leagues":  leagues,
			"year":     year,
		}
		render.HTML(w, http.StatusOK, "leaguesPlatformLeagues", data)
	}
}

func leaguesPostHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			render.HTML(w, http.StatusBadRequest, "400", err.Error())
			return
		}

		platform := r.FormValue("platform")
		leagueData := r.FormValue("league")
		year := r.FormValue("year")
		var parsed struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal([]byte(leagueData), &parsed); err != nil {
			msg := fmt.Sprintf("error parsing league data: %v", err)
			log.Print(msg)
			render.HTML(w, http.StatusBadRequest, "400", msg)
		}

		l, err := ctrl.AddLeague(r.Context(), platform, parsed.ID, parsed.Name, year)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err.Error())
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/leagues/%d", l.ID), http.StatusSeeOther)
	}
}
