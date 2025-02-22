package web

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mww/fantasy_manager_v2/controller"
	"github.com/mww/fantasy_manager_v2/db"
	"github.com/mww/fantasy_manager_v2/model"
	"github.com/unrolled/render"
)

func rootHandler(_ controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render.HTML(w, http.StatusOK, "root", nil)
	}
}

func versionHandler(_ controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render.HTML(w, http.StatusOK, "version", nil)
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

		scores, err := ctrl.GetPlayerScores(r.Context(), playerID)
		if err != nil {
			log.Printf("error getting player scores: %v", err)
		}

		data := map[string]any{
			"player": p,
			"scores": scores,
		}
		render.HTML(w, http.StatusOK, "player", data)
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

		scores, err := ctrl.GetPlayerScores(r.Context(), playerID)
		if err != nil {
			log.Printf("error getting player scores: %v", err)
		}

		data := map[string]any{
			"player": p,
			"scores": scores,
		}
		render.HTML(w, http.StatusOK, "player", data)
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

		players := make([]model.RankingPlayer, 0, len(ranking.Players))
		for _, p := range ranking.Players {
			players = append(players, p)
		}
		slices.SortFunc(players, func(a, b model.RankingPlayer) int {
			return int(a.Rank - b.Rank)
		})

		data := map[string]any{
			"date":    ranking.Date,
			"players": players,
		}
		render.HTML(w, http.StatusOK, "rankings", data)
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
		leagueID, err := getID(r, "leagueID")
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", err)
			return
		}

		l, err := ctrl.GetLeague(r.Context(), leagueID)
		if err != nil {
			render.HTML(w, http.StatusNotFound, "404", err.Error())
			return
		}

		resultWeeks, err := ctrl.ListLeagueResultWeeks(r.Context(), leagueID)
		if err != nil {
			log.Printf("error listing result weeks for league: %d", leagueID)
			resultWeeks = make([]int, 0)
		}

		powerRankings, err := ctrl.ListPowerRankings(r.Context(), leagueID)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err)
			return
		}

		// List the rankings so we can create a new power ranking if wanted
		rankings, err := ctrl.ListRankings(r.Context())
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err)
			return
		}

		data := map[string]any{
			"league":        l,
			"results":       resultWeeks,
			"powerRankings": powerRankings,
			"rankings":      rankings,
		}
		render.HTML(w, http.StatusOK, "league", data)
	}
}

func refreshLeagueManagersHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		leagueID, err := getID(r, "leagueID")
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", err)
			return
		}

		_, err = ctrl.AddLeagueManagers(r.Context(), leagueID)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err.Error())
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/leagues/%d", leagueID), http.StatusSeeOther)
	}
}

func syncWeekResultsHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		leagueID, err := getID(r, "leagueID")
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", err)
			return
		}

		if err := r.ParseForm(); err != nil {
			render.HTML(w, http.StatusBadRequest, "400", err)
			return
		}
		week, err := strconv.Atoi(r.FormValue("week"))
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", fmt.Errorf("error getting week value: %v", err))
			return
		}
		if week < 1 || week > 18 {
			render.HTML(w, http.StatusBadRequest, "400", "week must be between 1 and 18")
			return
		}

		if err := ctrl.SyncResultsFromPlatform(r.Context(), leagueID, week); err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/leagues/%d/week/%d", leagueID, week), http.StatusSeeOther)
	}
}

func getLeagueResultsHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		leagueID, err := getID(r, "leagueID")
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", err)
			return
		}

		week, err := strconv.Atoi(chi.URLParam(r, "week"))
		if err != nil {
			msg := fmt.Sprintf("error reading week value: %v", err)
			render.HTML(w, http.StatusBadRequest, "400", msg)
			return
		}

		league, err := ctrl.GetLeague(r.Context(), leagueID)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err)
			return
		}

		matchups, err := ctrl.GetLeagueResults(r.Context(), leagueID, week)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err)
			return
		}

		data := map[string]any{
			"matchups": matchups,
			"league":   league,
			"week":     week,
		}
		render.HTML(w, http.StatusOK, "leagueResults", data)
	}
}

func getLeagueResultsTemplateHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		leagueID, err := getID(r, "leagueID")
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", err)
			return
		}

		week, err := strconv.Atoi(chi.URLParam(r, "week"))
		if err != nil {
			msg := fmt.Sprintf("error reading week value: %v", err)
			render.HTML(w, http.StatusBadRequest, "400", msg)
			return
		}

		league, err := ctrl.GetLeague(r.Context(), leagueID)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err)
			return
		}

		matchups, err := ctrl.GetLeagueResults(r.Context(), leagueID, week)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err)
			return
		}

		topScores, err := ctrl.GetTopScores(r.Context(), leagueID, week)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err)
			return
		}

		prList, err := ctrl.ListPowerRankings(r.Context(), leagueID)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", fmt.Errorf("error listing power rankings: %v", err))
			return
		}
		var powerRanking *model.PowerRanking
		if int(prList[0].Week) == week {
			powerRanking, err = ctrl.GetPowerRanking(r.Context(), leagueID, prList[0].ID)
			if err != nil {
				render.HTML(w, http.StatusInternalServerError, "500", fmt.Errorf("error fetching power ranking: %v", err))
				return
			}
		}

		var standings []model.LeagueStanding
		standings, err = ctrl.GetLeagueStandings(r.Context(), leagueID)
		if err != nil {
			log.Printf("error getting league standings, non-fatal: %v", err)
		}

		var res strings.Builder
		res.WriteString("---\n")
		res.WriteString(fmt.Sprintf("title: \"%s\"\n", league.Name))
		res.WriteString(fmt.Sprintf("date: %s\n", time.Now().Format(time.DateOnly)))
		res.WriteString("image: cover.jpeg\n")
		res.WriteString(fmt.Sprintf("description: \"Week %d, %s\"\n", week, time.Now().Format("2006")))
		res.WriteString("layout: \"season\"\n")
		res.WriteString(fmt.Sprintf("url: \"/ff/seasons/%s/week-%02d\"\n", time.Now().Format("2006"), week))
		res.WriteString("---\n\n")

		res.WriteString("TODO - short statement about the week\n\n")

		res.WriteString("# Results\n\n")
		res.WriteString("League Median: TODO\n")

		res.WriteString("{{< table-with-class \"results-table\" >}}\n")
		res.WriteString("| Team | Score |\n")
		res.WriteString("| ---- | ----- |\n")
		for _, m := range matchups {
			var w, l *model.TeamResult
			if m.TeamA.Score > m.TeamB.Score {
				w = m.TeamA
				l = m.TeamB
			} else {
				w = m.TeamB
				l = m.TeamA
			}
			res.WriteString(fmt.Sprintf("| **%s** | **%s** |\n", w.TeamName, formatScore(w.Score)))
			res.WriteString(fmt.Sprintf("| %s | %s |\n", l.TeamName, formatScore(l.Score)))
			res.WriteString("| | |\n")
		}
		res.WriteString("{{< /table-with-class >}}\n\n")

		res.WriteString("# Fantasy Heros\n")
		res.WriteString("TODO - short commentary\n\n")
		res.WriteString("| Name | Team | Score |\n")
		res.WriteString("| ---- | ---- | ----- |\n")
		for _, t := range topScores {
			res.WriteString(fmt.Sprintf("| %s %s | TODO | %s |\n", t.FirstName, t.LastName, formatScore(t.Score)))
		}

		res.WriteString("\n")
		res.WriteString("# Blunder of the Week\n")
		res.WriteString("TODO - Choose a major blunder\n")

		res.WriteString("\n")
		res.WriteString("# Transaction action\n")
		res.WriteString("TODO - write about and trades or waiver wire moves\n")

		if powerRanking != nil {
			res.WriteString("\n")
			res.WriteString("# Power Rankings\n")
			res.WriteString("TODO - Short blurb\n\n")
			res.WriteString("| Team | Change |\n")
			res.WriteString("| ---- | ------ |\n")
			for _, t := range powerRanking.Teams {
				change := "-"
				if t.RankChange > 0 {
					change = fmt.Sprintf("{{< triangle-up %d >}}", t.RankChange)
				} else if t.RankChange < 0 {
					change = fmt.Sprintf("{{< triangle-down %d >}}", t.RankChange)
				}

				res.WriteString(fmt.Sprintf("| %s | %s |\n", t.TeamName, change))
			}
		}

		if standings != nil {
			res.WriteString("\n")
			res.WriteString("# Standings\n")
			res.WriteString("| Team | Record | Scored |\n")
			res.WriteString("| ---- | ------ | ------ |\n")
			for _, s := range standings {
				record := fmt.Sprintf("%d-%d", s.Wins, s.Losses)
				if s.Draws > 0 {
					record = fmt.Sprintf("%d-%d-%d", s.Wins, s.Losses, s.Draws)
				}
				res.WriteString(fmt.Sprintf("| %s | %s | %s |\n", s.TeamName, record, s.Scored))
			}
		}

		res.WriteString("\n")
		res.WriteString("# Weekly high scores\n")
		res.WriteString("1. TODO\n")

		res.WriteString("\n")
		res.WriteString("Cover Photo: [name/Icon Sportswire](link-to-image)\n")

		render.Text(w, http.StatusOK, res.String())
	}
}

func formatScore(score int32) string {
	s := float64(score) / 1000.00
	return fmt.Sprintf("%.02f", s)
}

func createPowerRankingsHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		leagueID, err := getID(r, "leagueID")
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", err)
			return
		}

		if err := r.ParseForm(); err != nil {
			render.HTML(w, http.StatusBadRequest, "400", fmt.Sprintf("unable to parse form: %v", err))
			return
		}

		rankingID, err := strconv.Atoi(r.FormValue("ranking"))
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", fmt.Sprintf("unable to parse ranking id: %v", err))
			return
		}

		week, err := strconv.Atoi(r.FormValue("week"))
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", fmt.Sprintf("unable to parse week value: %v", err))
			return
		}
		if week < 0 || week > 17 {
			render.HTML(w, http.StatusBadRequest, "400", fmt.Sprintf("week must be between 0 and 17, got: %d", week))
			return
		}

		id, err := ctrl.CalculatePowerRanking(r.Context(), leagueID, int32(rankingID), week)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/leagues/%d/power/%d", leagueID, id), http.StatusSeeOther)
	}
}

func showPowerRankingHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		leagueID, err := getID(r, "leagueID")
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", err)
			return
		}

		league, err := ctrl.GetLeague(r.Context(), leagueID)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err)
			return
		}

		powerRankingID, err := getID(r, "powerRankingID")
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", err)
			return
		}

		pr, err := ctrl.GetPowerRanking(r.Context(), leagueID, powerRankingID)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err)
			return
		}

		data := map[string]any{
			"league": league,
			"power":  pr,
		}
		render.HTML(w, http.StatusOK, "powerRanking", data)
	}
}

func showPowerRankingsTextHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		leagueID, err := getID(r, "leagueID")
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", err)
			return
		}

		powerRankingID, err := getID(r, "powerRankingID")
		if err != nil {
			render.HTML(w, http.StatusBadRequest, "400", err)
			return
		}

		pr, err := ctrl.GetPowerRanking(r.Context(), leagueID, powerRankingID)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err)
			return
		}

		var builder strings.Builder
		for i := range pr.Teams {
			var s string
			if pr.Teams[i].RankChange > 0 {
				s = fmt.Sprintf("%d. %s (+%d)\n", pr.Teams[i].Rank, pr.Teams[i].TeamName, pr.Teams[i].RankChange)
			} else if pr.Teams[i].RankChange < 0 {
				s = fmt.Sprintf("%d. %s (%d)\n", pr.Teams[i].Rank, pr.Teams[i].TeamName, pr.Teams[i].RankChange)
			} else {
				s = fmt.Sprintf("%d. %s\n", pr.Teams[i].Rank, pr.Teams[i].TeamName)
			}
			builder.WriteString(s)
		}
		render.Text(w, http.StatusOK, builder.String())
	}
}

func platformLeaguesHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		platform := r.URL.Query().Get("platform")
		username := r.URL.Query().Get("username")
		year := time.Now().Format("2006") // Just get the 4 digit year

		// For Yahoo, start the oauth flow
		if platform == model.PlatformYahoo {
			http.Redirect(w, r, "/oauth/link", http.StatusSeeOther)
			return
		}

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
		league := r.FormValue("league")
		year := r.FormValue("year")
		stateToken := r.FormValue("state")

		l, err := ctrl.AddLeague(r.Context(), platform, league, year, stateToken)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err.Error())
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/leagues/%d", l.ID), http.StatusSeeOther)
	}
}

func getID(r *http.Request, name string) (int32, error) {
	strID := chi.URLParam(r, name)
	id, err := strconv.Atoi(strID)
	if err != nil {
		return 0, fmt.Errorf("error parsing %s: %w", name, err)
	}
	return int32(id), err
}
