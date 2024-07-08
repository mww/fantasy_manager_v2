package sleeper

import (
	_ "embed"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mww/fantasy_manager_v2/model"
)

//go:embed testdata/players.json
var playerJSON []byte

func TestLoadPlayers_success(t *testing.T) {
	fakeSleeper := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Write(playerJSON)
	}))
	defer fakeSleeper.Close()

	c := &client{
		url:        fakeSleeper.URL,
		httpClient: http.DefaultClient,
	}

	players, err := c.LoadPlayers()
	if err != nil {
		t.Fatalf("error should not have been nil")
	}
	if players == nil {
		t.Fatalf("players shoud have been nil")
	}
	if len(players) != 4 {
		t.Fatalf("wrong number of players, expected 4, got %d", len(players))
	}

	expected := map[string]model.Player{
		"2374": {
			FirstName: "Tyler",
			LastName:  "Lockett",
			YahooID:   "28457",
			Position:  model.POS_WR,
			Team:      model.TEAM_SEA,
		},
		"6904": {
			FirstName: "Jalen",
			LastName:  "Hurts",
			YahooID:   "32723",
			Position:  model.POS_QB,
			Team:      model.TEAM_PHI,
		},
		"9509": {
			FirstName: "Bijan",
			LastName:  "Robinson",
			YahooID:   "",
			Position:  model.POS_RB,
			Team:      model.TEAM_ATL,
		},
		"11596": {
			FirstName: "Ben",
			LastName:  "Sinnott",
			YahooID:   "",
			Position:  model.POS_TE,
			Team:      model.TEAM_WAS,
		},
	}

	for _, p := range players {
		e, found := expected[p.ID] // Get the expected data
		if !found {
			t.Fatalf("unexpected player in the response %s", p.ID)
		}

		if p.FirstName != e.FirstName {
			t.Errorf("expected first name %s, got %s", e.FirstName, p.FirstName)
		}
		if p.LastName != e.LastName {
			t.Errorf("expected last name %s, got %s", e.LastName, p.LastName)
		}
		if p.YahooID != e.YahooID {
			t.Errorf("expected yahooID %s, got %s", e.YahooID, p.YahooID)
		}
		if p.Position != e.Position {
			t.Errorf("expected position %v, got %v", e.Position, p.Position)
		}
		if p.Team != e.Team {
			t.Errorf("expected team %v, got %v", e.Team, p.Team)
		}
	}
}

func TestLoadPlayers_httpError(t *testing.T) {
	fakeSleeper := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer fakeSleeper.Close()

	c := &client{
		url:        fakeSleeper.URL,
		httpClient: http.DefaultClient,
	}

	players, err := c.LoadPlayers()
	if err == nil {
		t.Fatalf("error should not have been nil")
	}
	if players != nil {
		t.Fatalf("players shoud have been nil")
	}
}
