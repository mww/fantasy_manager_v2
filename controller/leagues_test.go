package controller

import (
	"context"
	"reflect"
	"testing"

	"github.com/itbasis/go-clock"
	"github.com/mww/fantasy_manager_v2/model"
	"github.com/mww/fantasy_manager_v2/sleeper"
	"github.com/mww/fantasy_manager_v2/testutils"
)

func TestGetLeaguesFromPlatform(t *testing.T) {
	fakeSleeper := testutils.NewFakeSleeperServer()
	defer testutils.NewFakeSleeperServer()
	sleeper := sleeper.NewForTest(fakeSleeper.URL())

	ctrl, err := New(clock.New(), sleeper, testDB.DB)
	if err != nil {
		t.Fatalf("error creating new controller: %v", err)
	}

	ctx := context.Background()

	tests := map[string]struct {
		username  string
		platform  string
		year      string
		exErrMsg  string
		exLeagues []model.League
	}{
		"success sleeper": {username: "sleeperuser", platform: "sleeper", year: "2024", exLeagues: []model.League{
			{Name: "Footclan & Friends Dynasty", ExternalID: "924039165950484480", Year: "2024", Platform: "sleeper"},
			{Name: "The Megalabowl", ExternalID: "1005178517580746753", Year: "2024", Platform: "sleeper"},
		}},
		"unsupported platform": {username: "sleeperuser", platform: "ESPN", year: "2024",
			exErrMsg: "unsupported platform"},
		"bad year": {username: "sleeperuser", platform: "sleeper", year: "24",
			exErrMsg: "year parameter must be in the YYYY format, got: 24"},
		"unknown username": {username: "unknown", platform: "sleeper", year: "2024",
			exErrMsg: "user not found"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			leagues, err := ctrl.GetLeaguesFromPlatform(ctx, tc.username, tc.platform, tc.year)
			if tc.exErrMsg == "" {
				if !reflect.DeepEqual(tc.exLeagues, leagues) {
					t.Errorf("leagues are not as expected, got: %v", leagues)
				}
			} else {
				if err.Error() != tc.exErrMsg {
					t.Errorf("expected error message: %s, got: %v", tc.exErrMsg, err.Error())
				}
			}
		})
	}
}

func TestAddLeague(t *testing.T) {
	fakeSleeper := testutils.NewFakeSleeperServer()
	defer testutils.NewFakeSleeperServer()
	sleeper := sleeper.NewForTest(fakeSleeper.URL())

	ctrl, err := New(clock.New(), sleeper, testDB.DB)
	if err != nil {
		t.Fatalf("error creating new controller: %v", err)
	}

	ctx := context.Background()

	tests := map[string]struct {
		platform   string
		externalID string
		name       string
		year       string
		exErrMsg   string
	}{
		"success": {platform: "sleeper", externalID: "123", name: "League 1", year: "2024", exErrMsg: ""},
		"unsupported platform": {platform: "MFL", externalID: "123", name: "League 1", year: "2024",
			exErrMsg: "MFL is not a supported platform"},
		"bad external id": {platform: "sleeper", externalID: "    ", name: "League 1", year: "2024",
			exErrMsg: "externalID must be provided"},
		"bad name": {platform: "sleeper", externalID: "123", name: "", year: "2024",
			exErrMsg: "league name must be provided"},
		"bad date": {platform: "sleeper", externalID: "123", name: "League 4", year: "2024-07-01",
			exErrMsg: "year parameter must be in the YYYY format, got: 2024-07-01"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			l, err := ctrl.AddLeague(ctx, tc.platform, tc.externalID, tc.name, tc.year)

			if tc.exErrMsg == "" {
				if err != nil {
					t.Errorf("unexpected error adding league: %v", err)
				}
				if l.ID <= 0 {
					t.Errorf("league ID was not set as expected: %d", l.ID)
				}
				if l.Archived {
					t.Errorf("error league is archived")
				}
				if l.Name != tc.name || l.ExternalID != tc.externalID || l.Platform != tc.platform {
					t.Errorf("parameters for league are not as expected: %v", l)
				}
			} else {
				if err.Error() != tc.exErrMsg {
					t.Errorf("expected error: %s, got: %s", tc.exErrMsg, err.Error())
				}
			}
		})
	}
}

func TestAddLeagueManagers(t *testing.T) {
	ctx := context.Background()

	fakeSleeper := testutils.NewFakeSleeperServer()
	defer testutils.NewFakeSleeperServer()
	sleeper := sleeper.NewForTest(fakeSleeper.URL())

	ctrl, err := New(clock.New(), sleeper, testDB.DB)
	if err != nil {
		t.Fatalf("error creating new controller: %v", err)
	}

	l, err := ctrl.AddLeague(ctx, model.PlatformSleeper, "924039165950484480", "Footclan & Friends Dynasty", "2024")
	if err != nil {
		t.Fatalf("error adding league: %v", err)
	}

	l, err = ctrl.AddLeagueManagers(ctx, l.ID)
	if err != nil {
		t.Fatalf("error adding league managers: %v", err)
	}

	expectedManagers := []model.LeagueManager{
		{ExternalID: "300638784440004608", TeamName: "Puk Nukem", ManagerName: "8thAndFinalRule", JoinKey: "1"},
		{ExternalID: "362744067425296384", TeamName: "No-Bell Prizes", ManagerName: "mww", JoinKey: "4"},
		{ExternalID: "300368913101774848", ManagerName: "gee17", JoinKey: "6"},
		{ExternalID: "325106323354046464", TeamName: "Jolly Roger", ManagerName: "Jollymon", JoinKey: "7"},
	}
	if !reflect.DeepEqual(expectedManagers, l.Managers) {
		t.Errorf("l.Managers does not match expected value, got: %v", l.Managers)
	}
}
