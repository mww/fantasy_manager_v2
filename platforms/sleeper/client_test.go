package sleeper

import (
	_ "embed"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strconv"
	"testing"

	"github.com/mww/fantasy_manager_v2/model"
	"github.com/mww/fantasy_manager_v2/testutils"
)

func TestLoadPlayers_success(t *testing.T) {
	fakeSleeper := testutils.NewFakeSleeperServer()
	defer fakeSleeper.Close()

	c := NewForTest(fakeSleeper.URL())

	expected := map[string]model.Player{
		"SEA": {
			FirstName: "Seattle",
			LastName:  "Seahawks",
			Position:  model.POS_DEF,
			Team:      model.TEAM_SEA,
		},
		"1264": {
			FirstName: "Justin",
			LastName:  "Tucker",
			YahooID:   "26534",
			Position:  model.POS_K,
			Team:      model.TEAM_BAL,
		},
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

	players, err := c.LoadPlayers()
	if err != nil {
		t.Fatalf("error should not have been nil, was: %v", err)
	}
	if players == nil {
		t.Fatalf("players shoud have been nil")
	}

	// Validate that at least all of the expected players are in the results.
	// There may be more players, as I add more to the fake sleeper data as
	// needed, but at the very least we should have the expected ones.
	playerMap := make(map[string]*model.Player)
	for _, p := range players {
		playerMap[p.ID] = &p
	}

	for id, e := range expected {
		p, found := playerMap[id] // Get the expected data
		if !found {
			t.Fatalf("expected player not found in response %s", id)
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

	c := NewForTest(fakeSleeper.URL)

	players, err := c.LoadPlayers()
	if err == nil {
		t.Fatalf("error should not have been nil")
	}
	if players != nil {
		t.Fatalf("players shoud have been nil")
	}
}

func TestGetUserID(t *testing.T) {
	fakeSleeper := testutils.NewFakeSleeperServer()
	defer fakeSleeper.Close()

	c := NewForTest(fakeSleeper.URL())

	tests := []struct {
		username string
		expected string
		err      error
	}{
		{username: "sleeperuser", expected: "12345678"},
		{username: "badusername", expected: "", err: errors.New("user not found")},
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
	fakeSleeper := testutils.NewFakeSleeperServer()
	defer fakeSleeper.Close()

	c := NewForTest(fakeSleeper.URL())

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

func TestGetLeagueName(t *testing.T) {
	fakeSleeper := testutils.NewFakeSleeperServer()
	defer fakeSleeper.Close()
	c := NewForTest(fakeSleeper.URL())

	name, err := c.GetLeagueName(testutils.SleeperLeagueID)
	if err != nil {
		t.Fatalf("unexpected error getting league name: %v", err)
	}
	if name != "Footclan & Friends Dynasty" {
		t.Errorf("name was not the expected value, got: %s", name)
	}
}

func TestGetLeagueManagers(t *testing.T) {
	fakeSleeper := testutils.NewFakeSleeperServer()
	defer fakeSleeper.Close()
	c := NewForTest(fakeSleeper.URL())

	expectedManagers := []model.LeagueManager{
		{ExternalID: "300638784440004608", TeamName: "Puk Nukem", ManagerName: "8thAndFinalRule", JoinKey: "1"},
		{ExternalID: "362744067425296384", TeamName: "No-Bell Prizes", ManagerName: "mww", JoinKey: "4"},
		{ExternalID: "300368913101774848", ManagerName: "gee17", JoinKey: "6"},
		{ExternalID: "325106323354046464", TeamName: "Jolly Roger", ManagerName: "Jollymon", JoinKey: "7"},
	}

	tests := []struct {
		league   string
		expected []model.LeagueManager
		errMsg   string
	}{
		{league: "924039165950484480", expected: expectedManagers, errMsg: ""},
		{league: "1234", expected: nil, errMsg: "no managers found"},
	}

	for _, tc := range tests {
		t.Run(tc.league, func(t *testing.T) {
			managers, err := c.GetLeagueManagers(tc.league)
			if tc.errMsg != "" {
				if err.Error() != tc.errMsg {
					t.Errorf("expected error to be: %s, but got: %v", tc.errMsg, err)
				}
			}
			if !reflect.DeepEqual(tc.expected, managers) {
				t.Errorf("expected mangers to be: %v, but was: %v", tc.expected, managers)
			}
		})
	}
}

func TestGetMatchupResults(t *testing.T) {
	fakeSleeper := testutils.NewFakeSleeperServer()
	defer fakeSleeper.Close()
	c := NewForTest(fakeSleeper.URL())

	expectedMatchups := []model.Matchup{
		{
			TeamA:     &model.TeamResult{JoinKey: "1", Score: 107540},
			TeamB:     &model.TeamResult{JoinKey: "4", Score: 84300},
			MatchupID: 3,
			Week:      1,
		},
		{
			TeamA:     &model.TeamResult{JoinKey: "6", Score: 85060},
			TeamB:     &model.TeamResult{JoinKey: "7", Score: 114240},
			MatchupID: 5,
			Week:      1,
		},
	}
	expectedScores := []model.PlayerScore{
		{PlayerID: "1352", Score: 8700},
		{PlayerID: "3225", Score: 2000},
		{PlayerID: "4198", Score: -700},
		{PlayerID: "4993", Score: 5130},
		{PlayerID: "7601", Score: 6000},
		{PlayerID: "8154", Score: 13100},
		{PlayerID: "8408", Score: 0},
		{PlayerID: "10219", Score: 700},
		{PlayerID: "10222", Score: 5600},
		{PlayerID: "10223", Score: 0},
		{PlayerID: "11370", Score: 0},
		{PlayerID: "11439", Score: -200},
	}

	matchups, scores, err := c.GetMatchupResults(testutils.SleeperLeagueID, 1)
	if err != nil {
		t.Fatalf("unexpected error getting matchup results: %v", err)
	}

	if !reflect.DeepEqual(expectedMatchups, matchups) {
		t.Errorf("matchups were not the expected ones, got: %v", matchups)
	}

	// Sort the scores so they should be in the same order as the expected scores
	slices.SortFunc(scores, func(a, b model.PlayerScore) int {
		idA, e1 := strconv.Atoi(a.PlayerID)
		idB, e2 := strconv.Atoi(b.PlayerID)
		if err := errors.Join(e1, e2); err != nil {
			t.Errorf("error parsing player id when sorting player scores: %v", err)
			return 0
		}
		return idA - idB
	})
	if !reflect.DeepEqual(expectedScores, scores) {
		t.Errorf("player scores were not the expected ones, got: %v", scores)
	}
}

func TestGetRosters(t *testing.T) {
	fakeSleeper := testutils.NewFakeSleeperServer()
	defer fakeSleeper.Close()
	c := NewForTest(fakeSleeper.URL())

	e1 := model.Roster{
		TeamID:    "300638784440004608",
		PlayerIDs: []string{"10226", "10231", "11435", "1166", "1339", "4080"},
	}
	e2 := model.Roster{
		TeamID:    "362744067425296384",
		PlayerIDs: []string{"10871", "1234", "1476", "2078", "2251", "2374"},
	}
	e3 := model.Roster{
		TeamID:    "300368913101774848",
		PlayerIDs: []string{"10219", "10222", "10225", "10444", "1992", "3214"},
	}
	e4 := model.Roster{
		TeamID:    "325106323354046464",
		PlayerIDs: []string{"11439", "1352", "1466", "2216", "2359", "2449"},
	}

	rosters, err := c.GetRosters(testutils.SleeperLeagueID)
	if err != nil {
		t.Errorf("unexpected error loading rosters: %v", err)
	}
	if len(rosters) != 4 {
		t.Errorf("expected 4 rosters, but got %d instead", len(rosters))
	}
	if !reflect.DeepEqual(e1, rosters[0]) {
		t.Errorf("unexpected value for rosters[0], got: %v", rosters[0])
	}
	if !reflect.DeepEqual(e2, rosters[1]) {
		t.Errorf("unexpected value for rosters[1], got: %v", rosters[1])
	}
	if !reflect.DeepEqual(e3, rosters[2]) {
		t.Errorf("unexpected value for rosters[2], got: %v", rosters[2])
	}
	if !reflect.DeepEqual(e4, rosters[3]) {
		t.Errorf("unexpected value for rosters[3], got: %v", rosters[3])
	}
}

func TestGetStarters(t *testing.T) {
	fakeSleeper := testutils.NewFakeSleeperServer()
	defer fakeSleeper.Close()
	c := NewForTest(fakeSleeper.URL())

	starters, err := c.GetStarters(testutils.SleeperLeagueID)
	if err != nil {
		t.Errorf("unexpected error returned: %v", err)
	}

	expected := []model.RosterSpot{
		model.GetRosterSpot("QB"),
		model.GetRosterSpot("RB"),
		model.GetRosterSpot("RB"),
		model.GetRosterSpot("WR"),
		model.GetRosterSpot("WR"),
		model.GetRosterSpot("TE"),
		model.GetRosterSpot("FLEX"),
		model.GetRosterSpot("FLEX"),
		model.GetRosterSpot("FLEX"),
	}

	if !reflect.DeepEqual(expected, starters) {
		t.Errorf("starters do not match expected, got: %v", starters)
	}
}

func TestGetLeagueStandings(t *testing.T) {
	fakeSleeper := testutils.NewFakeSleeperServer()
	defer fakeSleeper.Close()
	c := NewForTest(fakeSleeper.URL())

	standings, err := c.GetLeagueStandings(testutils.SleeperLeagueID)
	if err != nil {
		t.Fatalf("error getting league standings: %v", err)
	}

	expected := []model.LeagueStanding{
		{
			TeamID: "325106323354046464",
			Rank:   1,
			Wins:   24,
			Losses: 4,
			Draws:  0,
			Scored: "1825.98",
		},
		{
			TeamID: "300368913101774848",
			Rank:   2,
			Wins:   20,
			Losses: 8,
			Draws:  0,
			Scored: "1554.16",
		},
		{
			TeamID: "300638784440004608",
			Rank:   3,
			Wins:   19,
			Losses: 9,
			Draws:  0,
			Scored: "1516.56",
		},
		{
			TeamID: "362744067425296384",
			Rank:   4,
			Wins:   18,
			Losses: 10,
			Draws:  0,
			Scored: "1525.08",
		},
	}

	if !reflect.DeepEqual(expected, standings) {
		t.Errorf("standings are not expected values, got: %v", standings)
	}
}
