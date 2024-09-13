package yahoo

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/mww/fantasy_manager_v2/model"
	"github.com/mww/fantasy_manager_v2/testutils"
)

func TestGetLeagueMetadata(t *testing.T) {
	fakeYahoo := testutils.NewFakeYahooServer()
	defer fakeYahoo.Close()

	c := NewForTest(fakeYahoo.URL())

	name, err := c.GetLeagueName(http.DefaultClient, testutils.YahooLeagueID)
	if err != nil {
		t.Fatalf("unexpected error getting league name: %v", err)
	}
	if name != "Y! Friends and Family League" {
		t.Errorf("league name was not expected value, got: %s", name)
	}
}

func TestGetLeagueMetadata_badLeagueId(t *testing.T) {
	fakeYahoo := testutils.NewFakeYahooServer()
	defer fakeYahoo.Close()

	c := NewForTest(fakeYahoo.URL())

	_, err := c.GetLeagueName(http.DefaultClient, "987")
	if err == nil {
		t.Fatal("expected an error, but got none")
	}
}

func TestGetStarters(t *testing.T) {
	fakeYahoo := testutils.NewFakeYahooServer()
	defer fakeYahoo.Close()

	c := NewForTest(fakeYahoo.URL())

	starters, err := c.GetStarters(http.DefaultClient, testutils.YahooLeagueID)
	if err != nil {
		t.Fatalf("unexpected error getting starters: %v", err)
	}

	expected := []model.RosterSpot{
		model.GetRosterSpot("QB"),
		model.GetRosterSpot("WR"),
		model.GetRosterSpot("WR"),
		model.GetRosterSpot("WR"),
		model.GetRosterSpot("RB"),
		model.GetRosterSpot("RB"),
		model.GetRosterSpot("TE"),
		model.GetRosterSpot("FLEX"),
		model.GetRosterSpot("K"),
		model.GetRosterSpot("DEF"),
	}

	if !reflect.DeepEqual(expected, starters) {
		t.Errorf("wanted %v but got %v", expected, starters)
	}
}

func TestGetTeams(t *testing.T) {
	fakeYahoo := testutils.NewFakeYahooServer()
	defer fakeYahoo.Close()

	c := NewForTest(fakeYahoo.URL())

	managers, err := c.GetManagers(http.DefaultClient, testutils.YahooLeagueID)
	if err != nil {
		t.Fatalf("unexpected error getting managers: %v", err)
	}

	expected := []model.LeagueManager{
		{
			ExternalID:  "223.l.431.t.10",
			TeamName:    "Gehlken",
			ManagerName: "Mark",
		},
		{
			ExternalID:  "223.l.431.t.5",
			TeamName:    "RotoExperts",
			ManagerName: "James",
		},
		{
			ExternalID:  "223.l.431.t.8",
			TeamName:    "Y! - Pianowski",
			ManagerName: "George",
		},
		{
			ExternalID:  "223.l.431.t.12",
			TeamName:    "Y! - Behrens",
			ManagerName: "James",
		},
	}

	if !reflect.DeepEqual(expected, managers) {
		t.Errorf("expected: %v, but got %v", expected, managers)
	}
}

func TestGetScoreboard(t *testing.T) {
	fakeYahoo := testutils.NewFakeYahooServer()
	defer fakeYahoo.Close()

	c := NewForTest(fakeYahoo.URL())

	matchups, err := c.GetScoreboard(http.DefaultClient, testutils.YahooLeagueID, 1)
	if err != nil {
		t.Fatalf("unexpected error getting yahoo scoreboard: %v", err)
	}

	expected := []model.Matchup{
		{
			Week: 1,
			TeamA: &model.TeamResult{
				TeamID: "223.l.431.t.10",
				Score:  142780,
			},
			TeamB: &model.TeamResult{
				TeamID: "223.l.431.t.5",
				Score:  88840,
			},
		},
		{
			Week: 1,
			TeamA: &model.TeamResult{
				TeamID: "223.l.431.t.8",
				Score:  122780,
			},
			TeamB: &model.TeamResult{
				TeamID: "223.l.431.t.12",
				Score:  87740,
			},
		},
	}

	if !reflect.DeepEqual(expected, matchups) {
		t.Errorf("expected %v, got %v", expected, matchups)
	}
}

func TestGetRoster(t *testing.T) {
	fakeYahoo := testutils.NewFakeYahooServer()
	defer fakeYahoo.Close()

	c := NewForTest(fakeYahoo.URL())

	roster, err := c.GetRoster(http.DefaultClient, testutils.YahooTeam10ID)
	if err != nil {
		t.Fatalf("unexpected error getting roster: %v", err)
	}

	expected := []model.YahooPlayer{
		{YahooID: "29288", FirstName: "Tyler", LastName: "Boyd", Pos: model.POS_WR},
		{YahooID: "30150", FirstName: "Zay", LastName: "Jones", Pos: model.POS_WR},
		{YahooID: "31012", FirstName: "Mike", LastName: "Gesicki", Pos: model.POS_TE},
	}

	if !reflect.DeepEqual(expected, roster) {
		t.Errorf("expected: %v, got: %v", expected, roster)
	}
}
