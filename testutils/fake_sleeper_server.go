package testutils

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
)

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
	serveFile(w, "players.json")
}

func userLeaguesHandler(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	year := chi.URLParam(r, "year")

	if userID == "12345678" && year == "2024" {
		serveFile(w, "user_leagues.json")
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}
}

func sleeperUserHandler(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "sleeperuser" {
		serveFile(w, "sleeperuser.json")
	} else {
		// requesting a user that doesn't exist seems to return a 200 with "null" as the response body as of 2024-08-12
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("null"))
	}
}

func serveFile(w http.ResponseWriter, name string) {
	b, err := sleeperdata.ReadFile(fmt.Sprintf("sleeperdata/%s", name))
	if err != nil {
		log.Printf("error reading sleeperdata/%s: %v", name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(b)
}
