package testutils

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
)

const YahooLeagueID = "431"
const fullYahooID = "nfl.l.431"

//go:embed yahoodata
var yahoodata embed.FS

type FakeYahooServer struct {
	s *httptest.Server
}

func NewFakeYahooServer() *FakeYahooServer {
	r := chi.NewRouter()
	// https://fantasysports.yahooapis.com/fantasy/v2/league/223.l.431/standings
	r.Route("/fantasy/v2", func(r chi.Router) {
		r.Route("/league/{leagueID}", func(r chi.Router) {
			r.Get("/", leagueMetadataHandler)
			r.Get("/settings", leagueSettingsHandler)
			r.Get("/standings", leagueStandingsHandler)
		})
	})

	return &FakeYahooServer{
		s: httptest.NewServer(r),
	}
}

func (f *FakeYahooServer) Close() {
	f.s.Close()
}

func (f *FakeYahooServer) URL() string {
	return f.s.URL
}

func leagueMetadataHandler(w http.ResponseWriter, r *http.Request) {
	leagueID := chi.URLParam(r, "leagueID")
	if leagueID == fullYahooID {
		serveYahooFile(w, "league_metadata.xml")
		return
	}

	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("error"))
}

func leagueSettingsHandler(w http.ResponseWriter, r *http.Request) {
	leagueID := chi.URLParam(r, "leagueID")
	if leagueID == fullYahooID {
		serveYahooFile(w, "settings.xml")
		return
	}

	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("error"))
}

func leagueStandingsHandler(w http.ResponseWriter, r *http.Request) {
	leagueID := chi.URLParam(r, "leagueID")
	if leagueID == fullYahooID {
		serveYahooFile(w, "standings.xml")
		return
	}

	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("error"))
}

func serveYahooFile(w http.ResponseWriter, name string) {
	b, err := yahoodata.ReadFile(fmt.Sprintf("yahoodata/%s", name))
	if err != nil {
		log.Printf("error reading yahoodata/%s: %v", name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "text/xml")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}
