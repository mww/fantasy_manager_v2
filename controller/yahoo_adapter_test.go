package controller

import (
	"context"
	"testing"

	"github.com/mww/fantasy_manager_v2/model"
	"github.com/mww/fantasy_manager_v2/testutils"
)

func TestYahooAdapter(t *testing.T) {
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

	l, err := ctrl.AddLeague(ctx, model.PlatformYahoo, testutils.YahooLeagueID, "2024", state)
	if err != nil {
		t.Fatalf("unexpected error adding league: %v", err)
	}
	if err := ctrl.OAuthSave(ctx, state, l.ID); err != nil {
		t.Fatalf("unexpected error saving oauth token: %v", err)
	}

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

	// Test methods that aren't implemented
	if _, err := adapter.getLeagues("user", "2024"); err == nil {
		t.Errorf("expected getLeagues to return an error")
	}

	if _, _, err := adapter.getMatchupResults(nil, 1); err == nil {
		t.Errorf("expected getMatchupResults to return an error")
	}

	if _, err := adapter.getRosters(nil); err == nil {
		t.Errorf("expected getRosters to return an error")
	}

	if _, err := adapter.getStarters(nil); err == nil {
		t.Errorf("expected getStarters to return an error")
	}
}
