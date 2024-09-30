package controller

import (
	"context"
	"reflect"
	"testing"

	"github.com/mww/fantasy_manager_v2/model"
	"github.com/mww/fantasy_manager_v2/testutils"
)

func TestGetLeaguesFromPlatform(t *testing.T) {
	ctx := context.Background()

	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()

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
			exErrMsg: "ESPN is not a supported platform"},
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
	ctx := context.Background()

	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()

	tests := map[string]struct {
		platform   string
		externalID string
		year       string
		exErrMsg   string
	}{
		"success": {platform: "sleeper", externalID: testutils.SleeperLeagueID, year: "2024", exErrMsg: ""},
		"unsupported platform": {platform: "MFL", externalID: testutils.SleeperLeagueID, year: "2024",
			exErrMsg: "MFL is not a supported platform"},
		"bad external id": {platform: "sleeper", externalID: "    ", year: "2024",
			exErrMsg: "externalID must be provided"},
		"bad date": {platform: "sleeper", externalID: testutils.SleeperLeagueID, year: "2024-07-01",
			exErrMsg: "year parameter must be in the YYYY format, got: 2024-07-01"},
		"missing external id": {platform: "sleeper", externalID: "123", year: "2024",
			exErrMsg: "league name not found: unexpected status code from sleeper: 404"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			l, err := ctrl.AddLeague(ctx, tc.platform, tc.externalID, tc.year, "" /* state */)

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
				if l.Name == "" || l.ExternalID != tc.externalID || l.Platform != tc.platform {
					t.Errorf("parameters for league are not as expected: %v", l)
				}

				// Clean up
				if err := ctrl.ArchiveLeague(ctx, l.ID); err != nil {
					t.Fatalf("error archiving league: %v", err)
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

	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()

	l, err := ctrl.AddLeague(ctx, model.PlatformSleeper, "924039165950484480", "2024", "" /* state */)
	if err != nil {
		t.Fatalf("error adding league: %v", err)
	}

	l, err = ctrl.AddLeagueManagers(ctx, l.ID)
	if err != nil {
		t.Fatalf("error adding league managers: %v", err)
	}

	l2, err := ctrl.GetLeague(ctx, l.ID)
	if err != nil {
		t.Fatalf("error loading league: %v", err)
	}

	expectedManagers := []model.LeagueManager{
		{ExternalID: "300638784440004608", TeamName: "Puk Nukem", ManagerName: "8thAndFinalRule", JoinKey: "1"},
		{ExternalID: "362744067425296384", TeamName: "No-Bell Prizes", ManagerName: "mww", JoinKey: "4"},
		{ExternalID: "300368913101774848", ManagerName: "gee17", JoinKey: "6"},
		{ExternalID: "325106323354046464", TeamName: "Jolly Roger", ManagerName: "Jollymon", JoinKey: "7"},
	}
	if !reflect.DeepEqual(expectedManagers, l2.Managers) {
		t.Errorf("l.Managers does not match expected value, got: %v", l.Managers)
	}
}

func TestSyncResultsFromPlatform(t *testing.T) {
	ctx := context.Background()

	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()

	if err := ctrl.UpdatePlayers(ctx); err != nil {
		t.Fatalf("error adding players: %v", err)
	}

	l, err := ctrl.AddLeague(ctx, model.PlatformSleeper, testutils.SleeperLeagueID, "2024", "" /* state */)
	if err != nil {
		t.Fatalf("error adding league: %v", err)
	}

	l, err = ctrl.AddLeagueManagers(ctx, l.ID)
	if err != nil {
		t.Fatalf("error adding league managers: %v", err)
	}

	if err := ctrl.SyncResultsFromPlatform(ctx, l.ID, 1); err != nil {
		t.Fatalf("error syncing league results: %v", err)
	}

	matchups, err := ctrl.GetLeagueResults(ctx, l.ID, 1)
	if err != nil {
		t.Fatalf("error loading matchups: %v", err)
	}

	expectedMatchups := []model.Matchup{
		{
			TeamA: &model.TeamResult{
				TeamID: "300638784440004608", TeamName: "Puk Nukem", Score: 107540,
			},
			TeamB: &model.TeamResult{
				TeamID: "362744067425296384", TeamName: "No-Bell Prizes", Score: 84300,
			},
			Week: 1,
		},
		{
			TeamA: &model.TeamResult{
				TeamID: "300368913101774848", TeamName: "gee17", Score: 85060,
			},
			TeamB: &model.TeamResult{
				TeamID: "325106323354046464", TeamName: "Jolly Roger", Score: 114240,
			},
			Week: 1,
		},
	}

	if len(matchups) != len(expectedMatchups) {
		t.Errorf("expected %d matchups, got %d", len(expectedMatchups), len(matchups))
	}

	for i, a := range matchups {
		e := expectedMatchups[i]
		if e.Week != a.Week {
			t.Errorf("expected week to be %d, but was %d", e.Week, a.Week)
		}
		if !reflect.DeepEqual(e.TeamA, a.TeamA) {
			t.Errorf("expected TeamA to be %v, got: %v, id: %d", e.TeamA, a.TeamA, a.MatchupID)
		}
	}

	scores, err := ctrl.GetPlayerScores(ctx, "8154")
	if err != nil {
		t.Fatalf("error getting player scores for id 8154: %v", err)
	}

	verified := false
	for _, s := range scores {
		if s.LeagueID != l.ID {
			continue
		}
		if verified {
			t.Errorf("score in league already verified, unexpected value: %v", s)
			continue
		}
		expectedScore := model.SeasonScores{
			LeagueID:   l.ID,
			LeagueName: l.Name,
			LeagueYear: l.Year,
			PlayerID:   "8154",
			Scores:     []int32{0, 13100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		}
		if !reflect.DeepEqual(expectedScore, s) {
			t.Errorf("player score not expected, got: %v", s)
		}
	}
}

func TestGetLeagueStandings(t *testing.T) {
	ctx := context.Background()

	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()

	if err := ctrl.UpdatePlayers(ctx); err != nil {
		t.Fatalf("error adding players: %v", err)
	}

	l, err := ctrl.AddLeague(ctx, model.PlatformSleeper, testutils.SleeperLeagueID, "2024", "" /* state */)
	if err != nil {
		t.Fatalf("error adding league: %v", err)
	}
	defer func() {
		ctrl.ArchiveLeague(ctx, l.ID)
	}()

	l, err = ctrl.AddLeagueManagers(ctx, l.ID)
	if err != nil {
		t.Fatalf("error adding league managers: %v", err)
	}

	standings, err := ctrl.GetLeagueStandings(ctx, l.ID)
	if err != nil {
		t.Fatalf("unexpected error getting league standings: %v", err)
	}

	expected := []model.LeagueStanding{
		{TeamID: "325106323354046464", TeamName: "Jolly Roger", Rank: 1, Wins: 24, Losses: 4, Draws: 0, Scored: "1825.98"},
		{TeamID: "300368913101774848", TeamName: "gee17", Rank: 2, Wins: 20, Losses: 8, Draws: 0, Scored: "1554.16"},
		{TeamID: "300638784440004608", TeamName: "Puk Nukem", Rank: 3, Wins: 19, Losses: 9, Draws: 0, Scored: "1516.56"},
		{TeamID: "362744067425296384", TeamName: "No-Bell Prizes", Rank: 4, Wins: 18, Losses: 10, Draws: 0, Scored: "1525.08"},
	}

	if !reflect.DeepEqual(expected, standings) {
		t.Errorf("expected: %v, got: %v", expected, standings)
	}
}
