package controller

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/mww/fantasy_manager_v2/model"
)

func TestCalculatePlayerValue(t *testing.T) {
	tests := []int32{1, 100, 250, 500, 1000, 1001, 1002}

	for _, tc := range tests {
		t.Run(fmt.Sprint(tc), func(t *testing.T) {
			val := calculatePlayerValue(tc)
			if val < 1 {
				t.Errorf("rank %d did not generate a value greater than 1: %d", tc, val)
			}
		})
	}
}

func TestInitalizePowerRankings(t *testing.T) {
	r1 := model.Roster{
		TeamID:    "1",
		PlayerIDs: []string{"1", "3", "5", "7", "9"},
	}
	r2 := model.Roster{
		TeamID:    "2",
		PlayerIDs: []string{"2", "4", "6", "8", "10"},
	}
	rosters := []model.Roster{r1, r2}

	ranking := &model.Ranking{
		ID: 1,
		Players: map[string]model.RankingPlayer{
			"1":  {Rank: 9, ID: "1"},
			"2":  {Rank: 4, ID: "2"},
			"3":  {Rank: 3, ID: "3"},
			"4":  {Rank: 1, ID: "4"},
			"5":  {Rank: 2, ID: "5"},
			"6":  {Rank: 6, ID: "6"},
			"7":  {Rank: 5, ID: "7"},
			"8":  {Rank: 7, ID: "8"},
			"10": {Rank: 10, ID: "10"},
		},
	}

	pr := initializePowerRankings(rosters, ranking)
	if len(pr.Teams) != len(rosters) {
		t.Errorf("expected result to have %d teams, but was %d", len(rosters), len(pr.Teams))
	}

	expected := &model.PowerRanking{
		RankingID: ranking.ID,
		Teams: []model.TeamPowerRanking{
			{
				TeamID: r1.TeamID,
				Roster: []model.PowerRankingPlayer{
					{PlayerID: "5", Rank: 2},
					{PlayerID: "3", Rank: 3},
					{PlayerID: "7", Rank: 5},
					{PlayerID: "1", Rank: 9},
					{PlayerID: "9", Rank: 1000},
				},
			},
			{
				TeamID: r2.TeamID,
				Roster: []model.PowerRankingPlayer{
					{PlayerID: "4", Rank: 1},
					{PlayerID: "2", Rank: 4},
					{PlayerID: "6", Rank: 6},
					{PlayerID: "8", Rank: 7},
					{PlayerID: "10", Rank: 10},
				},
			},
		},
	}

	if !reflect.DeepEqual(expected, pr) {
		t.Errorf("error with power ranking, expected:\n%v\ngot:\n%v", expected, pr)
	}
}

func TestCalculateRosterScores(t *testing.T) {
	pr := &model.PowerRanking{
		Teams: []model.TeamPowerRanking{
			{
				TeamID: "1",
				Roster: []model.PowerRankingPlayer{
					{PlayerID: "1", Rank: 1, Position: model.POS_RB},
					{PlayerID: "2", Rank: 2, Position: model.POS_TE},
					{PlayerID: "3", Rank: 3, Position: model.POS_RB},
					{PlayerID: "4", Rank: 4, Position: model.POS_WR},
					{PlayerID: "5", Rank: 5, Position: model.POS_WR},
					{PlayerID: "6", Rank: 6, Position: model.POS_WR},
					{PlayerID: "10", Rank: 10, Position: model.POS_RB},
					{PlayerID: "15", Rank: 15, Position: model.POS_WR},
					{PlayerID: "21", Rank: 21, Position: model.POS_QB},
				},
			},
		},
	}
	starters := []model.RosterSpot{
		model.GetRosterSpot("QB"),
		model.GetRosterSpot("RB"),
		model.GetRosterSpot("WR"),
		model.GetRosterSpot("FLEX"),
	}

	calculateRosterScores(pr, starters)
	if len(pr.Teams) != 1 {
		t.Fatalf("wrong number of results returned, expected 1 got: %d", len(pr.Teams))
	}
	team := pr.Teams[0]

	expectedStarters := map[string]bool{
		"21": true, // QB
		"1":  true, // RB
		"4":  true, // WR
		"2":  true, // FLEX
	}
	for _, p := range team.Roster {
		_, isStarter := expectedStarters[p.PlayerID]
		if isStarter != p.IsStarter {
			t.Errorf("player %s has isStarter: %v, but expected: %v - %v", p.PlayerID, p.IsStarter, isStarter, team)
		}
		if p.PowerRankingPoints <= 0 {
			t.Errorf("player %s has power ranking points <= 0: %d", p.PlayerID, p.PowerRankingPoints)
		}
	}
	if team.RosterScore <= 0 {
		t.Errorf("expected roster to have a score > 0, got: %d", team.RosterScore)
	}
}
