package controller

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mww/fantasy_manager_v2/model"
	"github.com/mww/fantasy_manager_v2/testutils"
)

func TestGetPlayerRankingMap(t *testing.T) {
	tests := map[string]struct {
		csvData  string
		err      error
		expected map[string]int32
	}{
		"good rankings": {csvData: rankingsGood, err: nil, expected: map[string]int32{
			testutils.IDJefferson: 1,
			testutils.IDMcCaffrey: 2,
			testutils.IDChase:     3,
			testutils.IDChubb:     4,
			testutils.IDTucker:    5,
			testutils.IDKelce:     6,
			testutils.IDHill:      7,
		}},
		"different col order": {csvData: rankingsDiffColOrder, err: nil, expected: map[string]int32{
			testutils.IDJefferson: 1,
			testutils.IDMcCaffrey: 2,
		}},
		"bad team name":    {csvData: rankingsBadTeamName, err: errors.New("bad team name for Christian McCaffrey"), expected: nil},
		"missing team col": {csvData: rankingsMissingTeamColumn, err: errors.New("error finding required columns; rank: 0, name: 2, team: -1, pos: 3"), expected: nil},
	}

	ctx := context.Background()

	c1, testCtrl := controllerForTest()
	defer testCtrl.Close()

	if err := c1.UpdatePlayers(ctx); err != nil {
		t.Fatalf("error adding players for test: %v", err)
	}

	ctrl := c1.(*controller)
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := strings.NewReader(tc.csvData)
			playerRanks, err := ctrl.getPlayerRankingMap(ctx, r)
			if tc.err == nil {
				if err != nil {
					t.Fatalf("expected err to be nil, but was: %v", err)
				}
				if !reflect.DeepEqual(tc.expected, playerRanks) {
					t.Errorf("player ranks were not as expected - actual: %v", playerRanks)
				}
			} else {
				if err == nil {
					t.Error("expected an error, got nil instead")
				} else if tc.err.Error() != err.Error() {
					t.Errorf("error was not what was expected - actual: %v", err)
				}
			}
		})
	}
}

func TestRankings(t *testing.T) {
	ctx := context.Background()

	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()

	if err := ctrl.UpdatePlayers(ctx); err != nil {
		t.Fatalf("error getting players: %v", err)
	}

	// Add a ranking
	date, _ := time.ParseInLocation(time.DateOnly, "2023-09-07", time.UTC)
	r := strings.NewReader(rankingsGood)

	id, err := ctrl.AddRanking(ctx, r, date)
	if err != nil {
		t.Fatalf("error adding a ranking: %v", err)
	}
	if id <= 0 {
		t.Fatalf("ranking id is less than 1: %d", id)
	}

	res1, err := ctrl.GetRanking(ctx, id)
	if err != nil {
		t.Fatalf("error getting ranking: %v", err)
	}
	if res1.Date.Format(time.DateOnly) != "2023-09-07" {
		t.Fatalf("rankings date is not expected: %s", res1.Date.Format(time.DateOnly))
	}

	expectedRankings := map[string]model.RankingPlayer{
		testutils.IDJefferson: {Rank: 1, ID: testutils.IDJefferson, FirstName: "Justin", LastName: "Jefferson", Position: model.POS_WR, Team: model.TEAM_MIN},
		testutils.IDMcCaffrey: {Rank: 2, ID: testutils.IDMcCaffrey, FirstName: "Christian", LastName: "McCaffrey", Position: model.POS_RB, Team: model.TEAM_SFO},
		testutils.IDChase:     {Rank: 3, ID: testutils.IDChase, FirstName: "Ja'Marr", LastName: "Chase", Position: model.POS_WR, Team: model.TEAM_CIN},
		testutils.IDChubb:     {Rank: 4, ID: testutils.IDChubb, FirstName: "Nick", LastName: "Chubb", Position: model.POS_RB, Team: model.TEAM_CLE},
		testutils.IDTucker:    {Rank: 5, ID: testutils.IDTucker, FirstName: "Justin", LastName: "Tucker", Position: model.POS_K, Team: model.TEAM_BAL},
		testutils.IDKelce:     {Rank: 6, ID: testutils.IDKelce, FirstName: "Travis", LastName: "Kelce", Position: model.POS_TE, Team: model.TEAM_KCC},
		testutils.IDHill:      {Rank: 7, ID: testutils.IDHill, FirstName: "Tyreek", LastName: "Hill", Position: model.POS_WR, Team: model.TEAM_MIA},
	}
	if len(expectedRankings) != len(res1.Players) {
		t.Errorf("wrong number of players, expected %d, got %d", len(expectedRankings), len(res1.Players))
	}
	for id, e := range expectedRankings {
		a, ok := res1.Players[id]
		if !ok {
			t.Errorf("no player with id %s found in actual rankings", id)
		}
		if !reflect.DeepEqual(e, a) {
			t.Errorf("expected: %v, got %v", e, a)
		}
	}

	rankings, err := ctrl.ListRankings(ctx)
	if err != nil {
		t.Fatalf("error listing rankings: %v", err)
	}
	if len(rankings) <= 0 {
		t.Fatalf("expected 1 or more results, got: %d", len(rankings))
	}
	idFound := false
	for _, r := range rankings {
		if r.ID == res1.ID {
			idFound = true
		}
	}
	if !idFound {
		t.Fatal("expected ranking id not found in list operation")
	}

	if err := ctrl.DeleteRanking(ctx, id); err != nil {
		t.Fatalf("error deleting ranking: %v", err)
	}

	res2, err := ctrl.GetRanking(ctx, id)
	if err == nil {
		t.Fatal("expected an error getting a deleting ranking but got none")
	}
	if res2 != nil {
		t.Fatal("expected res2 to be nil")
	}
}

func TestGetPosition(t *testing.T) {
	tests := []struct {
		input    string
		expected model.Position
	}{
		{input: "WR1", expected: model.POS_WR},
		{input: "RB17", expected: model.POS_RB},
		{input: "TE10", expected: model.POS_TE},
		{input: "QB2", expected: model.POS_QB},
		{input: "K1", expected: model.POS_K},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			a := getPosition(tc.input)
			if a != tc.expected {
				t.Errorf("expected '%s', got '%s'", tc.expected, a)
			}
		})
	}
}

var (
	// A kicker is included to ensure we are filtering them as expected.
	rankingsGood = `"RK",TIERS,"PLAYER NAME",TEAM,"POS","BYE WEEK","SOS SEASON","ECR VS. ADP"
"1",1,"Justin Jefferson",MIN,"WR1","13","3 out of 5 stars","+1"
"2",1,"Christian McCaffrey",SF,"RB1","9","4 out of 5 stars","-1"
"3",1,"Ja'Marr Chase",CIN,"WR2","7","4 out of 5 stars","+1"
"4",1,"Nick Chubb",CLE,"RB2","5","3 out of 5 stars","+5"
"5",1,"Justin Tucker",BAL,"K1","13","5 out of 5 stars","-68"
"6",2,"Travis Kelce",KC,"TE1","10","4 out of 5 stars","0"
"7",2,"Tyreek Hill",MIA,"WR3","10","3 out of 5 stars","0"`

	rankingsBadTeamName = `"RK",TIERS,"PLAYER NAME",TEAM,"POS","BYE WEEK","SOS SEASON","ECR VS. ADP"
"1",1,"Justin Jefferson",MIN,"WR1","13","3 out of 5 stars","+1"
"2",1,"Christian McCaffrey",XXX,"RB1","9","4 out of 5 stars","-1"
"3",1,"Ja'Marr Chase",CIN,"WR2","7","4 out of 5 stars","+1"`

	rankingsMissingTeamColumn = `"RK",TIERS,"PLAYER NAME","POS","BYE WEEK","SOS SEASON","ECR VS. ADP"
"1",1,"Justin Jefferson","WR1","13","3 out of 5 stars","+1"
"2",1,"Christian McCaffrey","RB1","9","4 out of 5 stars","-1"`

	rankingsDiffColOrder = `"POS","RK",TIERS,"BYE WEEK",TEAM,"SOS SEASON","ECR VS. ADP","PLAYER NAME"
"WR1","1",1,"13",MIN,"3 out of 5 stars","+1","Justin Jefferson"
"RB1","2",1,"9",SF,"4 out of 5 stars","-1","Christian McCaffrey"`
)
