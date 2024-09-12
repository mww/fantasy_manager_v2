package testutils

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"

	"github.com/go-chi/chi/v5"
)

const SleeperLeagueID = "924039165950484480"

//go:embed sleeperdata
var sleeperdata embed.FS

type FakeSleeperServer struct {
	s *httptest.Server
}

func NewFakeSleeperServer() *FakeSleeperServer {
	r := chi.NewRouter()
	r.Route("/v1", func(r chi.Router) {
		r.Get("/players/nfl", nflPlayersHandler)

		r.Route("/user", func(r chi.Router) {
			r.Get("/{userID}/leagues/nfl/{year}", userLeaguesHandler)
			r.Get("/{username}", sleeperUserHandler)
		})

		r.Route("/league/{leagueID}", func(r chi.Router) {
			r.Get("/", leagueHandler)
			r.Get("/users", leagueUsersHandler)
			r.Get("/rosters", leagueRostersHandler)
			r.Get("/matchups/{week:\\d+}", leagueMatchupsHandlers)
		})
	})

	return &FakeSleeperServer{
		s: httptest.NewServer(r),
	}
}

func (f *FakeSleeperServer) Close() {
	f.s.Close()
}

func (f *FakeSleeperServer) URL() string {
	return f.s.URL
}

func nflPlayersHandler(w http.ResponseWriter, r *http.Request) {
	serveSleeperFile(w, "players.json")
}

func userLeaguesHandler(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	year := chi.URLParam(r, "year")

	if userID == "12345678" && year == "2024" {
		serveSleeperFile(w, "user_leagues.json")
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}
}

func sleeperUserHandler(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "sleeperuser" {
		serveSleeperFile(w, "sleeperuser.json")
	} else {
		// requesting a user that doesn't exist seems to return a 200 with "null" as the response body as of 2024-08-12
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("null"))
	}
}

func leagueHandler(w http.ResponseWriter, r *http.Request) {
	leagueID := chi.URLParam(r, "leagueID")
	if leagueID == SleeperLeagueID {
		serveSleeperFile(w, "league.json")
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("null"))
	}
}

func leagueUsersHandler(w http.ResponseWriter, r *http.Request) {
	leagueID := chi.URLParam(r, "leagueID")
	if leagueID == SleeperLeagueID {
		serveSleeperFile(w, "league_users.json")
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}
}

func leagueRostersHandler(w http.ResponseWriter, r *http.Request) {
	leagueID := chi.URLParam(r, "leagueID")
	if leagueID == SleeperLeagueID {
		serveSleeperFile(w, "league_rosters.json")
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}
}

func leagueMatchupsHandlers(w http.ResponseWriter, r *http.Request) {
	leagueID := chi.URLParam(r, "leagueID")
	if leagueID == SleeperLeagueID {
		if week, err := strconv.Atoi(chi.URLParam(r, "week")); err == nil {
			if week <= 5 && week >= 1 {
				serveSleeperFile(w, fmt.Sprintf("matchups-week-%02d.json", week))
				return
			}
		} else {
			log.Printf("error parsing week param: %v", err)
		}
	}

	// TODO: figure out what is actually returned
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"errMsg": "not found"}`))
}

func serveSleeperFile(w http.ResponseWriter, name string) {
	b, err := sleeperdata.ReadFile(fmt.Sprintf("sleeperdata/%s", name))
	if err != nil {
		log.Printf("error reading sleeperdata/%s: %v", name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}
