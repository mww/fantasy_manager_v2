package controller

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mww/fantasy_manager_v2/model"
	"github.com/mww/fantasy_manager_v2/testutils"
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

func TestCalculateAndGetPowerRanking(t *testing.T) {
	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()

	ctx := context.Background()

	if err := ctrl.UpdatePlayers(ctx); err != nil {
		t.Fatalf("error updating players: %v", err)
	}

	l, err := ctrl.AddLeague(ctx, model.PlatformSleeper, testutils.SleeperLeagueID, "2024", "" /* state */)
	if err != nil {
		t.Fatalf("error adding a new league: %v", err)
	}
	defer func() {
		if err := ctrl.ArchiveLeague(ctx, l.ID); err != nil {
			t.Fatalf("error archiving league: %v", err)
		}
	}()

	if _, err := ctrl.AddLeagueManagers(ctx, l.ID); err != nil {
		t.Fatalf("error adding league managers: %v", err)
	}

	rankingDate, err := time.ParseInLocation(time.DateOnly, "2018-09-01", time.UTC)
	if err != nil {
		t.Fatalf("error parsing ranking date: %v", err)
	}
	rankingID, err := ctrl.AddRanking(ctx, getRankingsData(), rankingDate)
	if err != nil {
		t.Fatalf("error adding ranking: %v", err)
	}

	// Now that all of the setup is done, calculate and verify the power rankings.
	prID, err := ctrl.CalculatePowerRanking(ctx, l.ID, rankingID, 0)
	if err != nil {
		t.Fatalf("error calculating power ranking: %v", err)
	}

	pr, err := ctrl.GetPowerRanking(ctx, l.ID, prID)
	if err != nil {
		t.Fatalf("error getting power ranking: %v", err)
	}

	expected := model.PowerRanking{
		Teams: []model.TeamPowerRanking{
			{
				TeamID:   "325106323354046464",
				TeamName: "Jolly Roger",
				Rank:     1,
				Roster: []model.PowerRankingPlayer{
					{FirstName: "Mike", LastName: "Evans"},
					{FirstName: "Travis", LastName: "Kelce"},
					{FirstName: "Stefon", LastName: "Diggs"},
					{FirstName: "Jaleel", LastName: "McLaughlin"},
					{FirstName: "Robert", LastName: "Woods"},
					{FirstName: "Ameer", LastName: "Abdullah"},
				},
			},
			{
				TeamID:   "300368913101774848",
				TeamName: "gee17",
				Rank:     2,
				Roster: []model.PowerRankingPlayer{
					{FirstName: "Jayden", LastName: "Reed"},
					{FirstName: "Hunter", LastName: "Henry"},
					{FirstName: "Jonathan", LastName: "Mingo"},
					{FirstName: "Cedric", LastName: "Tillman"},
					{FirstName: "Chris", LastName: "Rodriguez"},
					{FirstName: "Allen", LastName: "Robinson"},
				},
			},
			{
				TeamID:   "362744067425296384",
				TeamName: "No-Bell Prizes",
				Rank:     3,
				Roster: []model.PowerRankingPlayer{
					{FirstName: "Tyler", LastName: "Lockett"},
					{FirstName: "Russell", LastName: "Wilson"},
					{FirstName: "Odell", LastName: "Beckham"},
					{FirstName: "Luke", LastName: "Schoonmaker"},
					{FirstName: "Logan", LastName: "Thomas"},
					{FirstName: "Latavius", LastName: "Murray"},
				},
			},
			{
				TeamID:   "300638784440004608",
				TeamName: "Puk Nukem",
				Rank:     4,
				Roster: []model.PowerRankingPlayer{
					{FirstName: "Kirk", LastName: "Cousins"},
					{FirstName: "Andrei", LastName: "Iosivas"},
					{FirstName: "Zach", LastName: "Ertz"},
					{FirstName: "Zay", LastName: "Jones"},
					{FirstName: "Emanuel", LastName: "Wilson"},
					{FirstName: "Elijah", LastName: "Higgins"},
				},
			},
		},
	}

	for i := range expected.Teams {
		e := expected.Teams[i]
		a := pr.Teams[i]

		if e.TeamID != a.TeamID {
			t.Errorf("expected TeamID to be %s, but was %s", e.TeamID, a.TeamID)
		}
		if e.TeamName != a.TeamName {
			t.Errorf("expected TeamName to be %s, but was %s", e.TeamName, a.TeamName)
		}
		if e.Rank != a.Rank {
			t.Errorf("expected Rank to be %d, but was %d", e.Rank, a.Rank)
		}

		for j := range e.Roster {
			ep := e.Roster[j]
			ap := a.Roster[j]

			if ep.FirstName != ap.FirstName {
				t.Errorf("For roster spot %d, FirstName expected to be %s but was %s", j, ep.FirstName, ap.FirstName)
			}
			if ep.LastName != ap.LastName {
				t.Errorf("For roster spot %d, LastName expected to be %s but was %s", j, ep.LastName, ap.LastName)
			}
		}
	}
}

func getRankingsData() io.Reader {
	// Rondale Moore is intentionally missing from this list, to be someone
	// without a ranking.
	const rankings = `"RK","PLAYER NAME",TEAM,"POS"
"1","Justin Jefferson",MIN,"WR1"
"2","Christian McCaffrey",SF,"RB1"
"3","Ja'Marr Chase",CIN,"WR2"
"4","Nick Chubb",CLE,"RB2"
"5","Bijan Robinson",ATL,"RB3"
"7","Tyreek Hill",MIA,"WR3"
"21","Mike Evans",TB,"WR13"
"29","Jalen Hurts",PHI,"QB2"
"35","Travis Kelce",KC,"TE2"
"51","Stefon Diggs",HOU,"WR27"
"80","Jayden Reed",GB,"WR37"
"84","Brian Robinson Jr.",WAS,"RB28"
"116","Tyler Lockett",SEA,"WR51"
"126","Kirk Cousins",ATL,"QB18"
"135","Jaleel McLaughlin",DEN,"RB45"
"151","Hunter Henry",NE,"TE17"
"177","Jordan Mason",SF,"RB56"
"197","Tyler Boyd",TEN,"WR76"
"199","Russell Wilson",PIT,"QB30"
"208","Andrei Iosivas",CIN,"WR77"
"215","Ben Sinnott",WAS,"TE26"
"224","Mike Gesicki",CIN,"TE27"
"255","Zach Ertz",WAS,"TE30"
"266","Jonathan Mingo",CAR,"WR94"
"267","Odell Beckham Jr.",MIA,"WR95"
"275","Eric Gray",NYG,"RB77"
"283","Zay Jones",ARI,"WR100"
"285","Cedric Tillman",CLE,"WR101"
"310","Chris Rodriguez Jr.",WAS,"RB90"
"312","Robert Woods",HOU,"WR109"
"319","Emanuel Wilson",GB,"RB92"
"395","Ameer Abdullah",LV,"RB107"
"396","Kyle Juszczyk",SF,"RB108"
"411","Luke Schoonmaker",DAL,"TE60"
"434","Logan Thomas",FA,"TE64"
"435","Elijah Higgins",ARI,"TE65"
"456","Latavius Murray",FA,"RB123"
"519","Allen Robinson II",DET,"WR173"
"600","Jamal Agnew",CAR,"WR203"
"671","Chris Brooks",MIA,"RB172"`

	return strings.NewReader(rankings)
}
