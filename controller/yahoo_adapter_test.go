package controller

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/itbasis/go-clock"
	"github.com/mww/fantasy_manager_v2/model"
	"github.com/mww/fantasy_manager_v2/testutils"
	"golang.org/x/oauth2"
)

func TestGetLeagueName(t *testing.T) {
	ctx := context.Background()

	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()

	adapter := &yahooAdapter{ctrl.(*controller)}

	authURL, err := ctrl.OAuthStart(model.PlatformYahoo)
	state := validateOAuthStart(t, authURL, err)

	if err := ctrl.OAuthExchange(ctx, state, "code"); err != nil {
		t.Fatalf("error exchanging oauth token: %v", err)
	}

	name, err := adapter.getLeagueName(ctx, testutils.YahooLeagueID, state)
	if err != nil {
		t.Fatalf("unexpected error getting league name: %v", err)
	}
	if name != "Y! Friends and Family League" {
		t.Fatalf("league name was not expected value: %s", name)
	}
}

func TestSortManagers(t *testing.T) {
	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()
	adapter := &yahooAdapter{ctrl.(*controller)}

	input := []model.LeagueManager{
		{ExternalID: "449.l.149976.t.1"},
		{ExternalID: "449.l.149976.t.7"},
		{ExternalID: "449.l.149976.t.5"},
		{ExternalID: "449.l.149976.t.4"},
		{ExternalID: "449.l.149976.t.9"},
		{ExternalID: "449.l.149976.t.12"},
		{ExternalID: "449.l.149976.t.2"},
		{ExternalID: "449.l.149976.t.8"},
		{ExternalID: "449.l.149976.t.10"},
		{ExternalID: "449.l.149976.t.6"},
		{ExternalID: "449.l.149976.t.3"},
		{ExternalID: "449.l.149976.t.11"},
	}

	expected := []model.LeagueManager{
		{ExternalID: "449.l.149976.t.1"},
		{ExternalID: "449.l.149976.t.2"},
		{ExternalID: "449.l.149976.t.3"},
		{ExternalID: "449.l.149976.t.4"},
		{ExternalID: "449.l.149976.t.5"},
		{ExternalID: "449.l.149976.t.6"},
		{ExternalID: "449.l.149976.t.7"},
		{ExternalID: "449.l.149976.t.8"},
		{ExternalID: "449.l.149976.t.9"},
		{ExternalID: "449.l.149976.t.10"},
		{ExternalID: "449.l.149976.t.11"},
		{ExternalID: "449.l.149976.t.12"},
	}

	adapter.sortManagers(input)
	if !reflect.DeepEqual(input, expected) {
		t.Errorf("expected: %v, got: %v", expected, input)
	}
}

func TestGetManagers(t *testing.T) {
	ctx := context.Background()

	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()

	l := setupTest(t, ctx, testCtrl.Clock)
	defer func() {
		testDB.DB.ArchiveLeague(ctx, l.ID)
	}()

	adapter := &yahooAdapter{ctrl.(*controller)}
	managers, err := adapter.getManagers(ctx, l)
	if err != nil {
		t.Fatalf("unexpected error getting managers: %v", err)
	}
	expectedNames := []string{
		"Gehlken",
		"RotoExperts",
		"Y! - Pianowski",
		"Y! - Behrens",
	}
	for i := range managers {
		if managers[i].TeamName != expectedNames[i] {
			t.Errorf("expected managers[%d].TeamName to be %s, but was %s", i, managers[i].TeamName, expectedNames[i])
		}
	}
}

func TestGetStarters(t *testing.T) {
	ctx := context.Background()

	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()

	l := setupTest(t, ctx, testCtrl.Clock)
	defer func() {
		testDB.DB.ArchiveLeague(ctx, l.ID)
	}()

	adapter := &yahooAdapter{ctrl.(*controller)}
	starters, err := adapter.getStarters(ctx, l)
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
		t.Errorf("expected: %v, got: %v", expected, starters)
	}
}

func TestGetLeagues(t *testing.T) {
	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()
	adapter := &yahooAdapter{ctrl.(*controller)}
	if _, err := adapter.getLeagues("user", "2024"); err == nil {
		t.Errorf("expected an error in getLeague(), but got none")
	}
}

func TestGetMatchupResults(t *testing.T) {
	ctx := context.Background()

	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()

	l := setupTest(t, ctx, testCtrl.Clock)
	defer func() {
		testDB.DB.ArchiveLeague(ctx, l.ID)
	}()

	adapter := &yahooAdapter{ctrl.(*controller)}
	matchups, players, err := adapter.getMatchupResults(ctx, l, 1)
	if err != nil {
		t.Fatalf("unexpected error in getMatchupResults: %v", err)
	}
	if players == nil || len(players) > 0 {
		t.Errorf("players was nil or had results when expected to be empty: %v", players)
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

func TestGetRosters(t *testing.T) {
	ctx := context.Background()

	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()

	l := setupTest(t, ctx, testCtrl.Clock)
	defer func() {
		testDB.DB.ArchiveLeague(ctx, l.ID)
	}()
	l, err := ctrl.AddLeagueManagers(ctx, l.ID)
	if err != nil {
		t.Fatalf("error adding league managers: %v", err)
	}
	if err := ctrl.UpdatePlayers(ctx); err != nil {
		t.Fatalf("error updating players: %v", err)
	}

	adapter := &yahooAdapter{ctrl.(*controller)}
	rosters, err := adapter.getRosters(ctx, l)
	if err != nil {
		t.Fatalf("unexpected error getting yahoo rosters: %v", err)
	}

	expected := []model.Roster{
		{TeamID: "223.l.431.t.5", PlayerIDs: []string{"1166", "1339", "1352"}},
		{TeamID: "223.l.431.t.8", PlayerIDs: []string{"1992", "2216", "2359"}},
		{TeamID: "223.l.431.t.10", PlayerIDs: []string{"3225", "4080", "4993"}},
		{TeamID: "223.l.431.t.12", PlayerIDs: []string{"7601", "8154", "10219"}},
	}

	if !reflect.DeepEqual(expected, rosters) {
		t.Errorf("expected: %v, got: %v", expected, rosters)
	}
}

func setupTest(t *testing.T, ctx context.Context, clock clock.Clock) *model.League {
	l := &model.League{
		Platform:   model.PlatformYahoo,
		ExternalID: testutils.YahooLeagueID,
		Name:       "Fake Yahoo League",
		Year:       "2024",
		Archived:   false,
	}
	err := testDB.DB.AddLeague(ctx, l)
	if err != nil {
		t.Fatalf("unexpected error adding league: %v", err)
	}

	token := &oauth2.Token{
		AccessToken:  "access",
		RefreshToken: "refresh",
		Expiry:       clock.Now().Add(1 * time.Hour),
	}
	if err := testDB.DB.SaveToken(ctx, l.ID, token); err != nil {
		t.Fatalf("error saving token: %v", err)
	}

	return l
}
