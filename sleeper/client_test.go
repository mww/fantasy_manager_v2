package sleeper

import (
	_ "embed"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/mww/fantasy_manager_v2/model"
)

//go:embed testdata/players.json
var playerJSON []byte

//go:embed testdata/user.json
var userJSON []byte

//go:embed testdata/user_leagues.json
var userLeaguesJSON []byte

func TestLoadPlayers_success(t *testing.T) {
	fakeSleeper := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Write(playerJSON)
	}))
	defer fakeSleeper.Close()

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
		"1379": {
			FirstName: "Kyle",
			LastName:  "Juszczyk",
			YahooID:   "26753",
			Position:  model.POS_RB,
			Team:      model.TEAM_SFO,
		},
	}

	c := &client{
		url:        fakeSleeper.URL,
		httpClient: http.DefaultClient,
	}

	players, err := c.LoadPlayers()
	if err != nil {
		t.Fatalf("error should not have been nil, was: %v", err)
	}
	if players == nil {
		t.Fatalf("players shoud have been nil")
	}
	if len(players) != len(expected) {
		t.Fatalf("wrong number of players, expected 4, got %d", len(players))
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

func TestGetUserID(t *testing.T) {
	fakeSleeper := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/v1/user/sleeperuser" {
			rw.WriteHeader(http.StatusOK)
			rw.Write(userJSON)
		} else {
			// requesting a user that doesn't exist seems to return a 200 with "null" as the response body as of 2024-08-12
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte("null"))
		}
	}))
	defer fakeSleeper.Close()

	c := &client{
		url:        fakeSleeper.URL,
		httpClient: http.DefaultClient,
	}

	tests := []struct {
		username string
		expected string
		err      error
	}{
		{username: "sleeperuser", expected: "12345678"},
		{username: "badusername", expected: "", err: errors.New("no user found")},
	}

	for _, tc := range tests {
		t.Run(tc.username, func(t *testing.T) {
			userID, err := c.GetUserID(tc.username)
			if tc.err != nil {
				if err.Error() != tc.err.Error() {
					t.Errorf("expected err to be: '%v', got '%v' instead", tc.err, err)
				}
			} else {
				if err != nil {
					t.Fatalf("error was not nil, was %v", err)
				}
				if userID != tc.expected {
					t.Errorf("user id was not expected, wanted: '%s', got: %s'", tc.expected, userID)
				}
			}
		})
	}
}

func TestGetLeaguesForUser(t *testing.T) {
	fakeSleeper := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/v1/user/12345678/leagues/nfl/2024" {
			rw.WriteHeader(http.StatusOK)
			rw.Write(userLeaguesJSON)
		} else {
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte("[]"))
		}
	}))
	defer fakeSleeper.Close()

	c := &client{
		url:        fakeSleeper.URL,
		httpClient: http.DefaultClient,
	}

	tests := []struct {
		userID   string
		year     string
		expected []model.League
		err      error
	}{
		{userID: "12345678", year: "2024", expected: []model.League{
			{ExternalID: "924039165950484480", Name: "Footclan & Friends Dynasty", Platform: "sleeper", Year: "2024", Archived: false},
			{ExternalID: "1005178517580746753", Name: "The Megalabowl", Platform: "sleeper", Year: "2024", Archived: false}}},
		{userID: "98765432", year: "2024", expected: nil, err: errors.New("no leagues found")},
	}

	for _, tc := range tests {
		t.Run(tc.userID, func(t *testing.T) {
			l, err := c.GetLeaguesForUser(tc.userID, tc.year)
			if !reflect.DeepEqual(l, tc.expected) {
				t.Errorf("result does not match expected leagues: %v", l)
			}
			if tc.err != nil {
				if tc.err.Error() != err.Error() {
					t.Errorf("expected error '%v' but got '%v'", tc.err, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, but got: '%v'", err)
				}
			}
		})
	}
}
