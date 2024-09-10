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
