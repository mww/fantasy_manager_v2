package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/itbasis/go-clock"
	"github.com/mww/fantasy_manager_v2/containers"
	"github.com/mww/fantasy_manager_v2/model"
	"golang.org/x/oauth2"
)

var (
	// A test global db instance to use for all of the tests instead of setting up a new one each time.
	testDB DB

	// a counter to generate new player ids for each test. To help keep them separated.
	idCtr = int32(0)
)

// TestMain controls the main for the tests and allows for setup and shutdown of the tests
func TestMain(m *testing.M) {
	container := containers.NewDBContainer()

	clock := clock.New()

	defer func() {
		// Catch all panics to make sure the shutdown is successfully run
		if r := recover(); r != nil {
			if container != nil {
				container.Shutdown()
			}
			fmt.Println("panic")
		}
	}()

	var err error
	testDB, err = New(context.Background(), container.ConnectionString(), clock)
	if err != nil {
		fmt.Printf("error connecting to db: %v", err)
		os.Exit(-1)
	}

	code := m.Run()
	container.Shutdown()
	os.Exit(code)
}

// Test all of the various ranking operations, adding, getting, listing, and deleting
func TestRankings(t *testing.T) {
	p1 := getPlayerWithName("Justin", "Jefferson")
	p2 := getPlayerWithName("Ja'Marr", "Chase")
	p3 := getPlayerWithName("Tyreek", "Hill")
	p4 := getPlayerWithName("Stefon", "Diggs")
	p5 := getPlayerWithName("A.J.", "Brown")

	ctx := context.Background()
	e1 := testDB.SavePlayer(ctx, p1)
	e2 := testDB.SavePlayer(ctx, p2)
	e3 := testDB.SavePlayer(ctx, p3)
	e4 := testDB.SavePlayer(ctx, p4)
	e5 := testDB.SavePlayer(ctx, p5)
	if err := errors.Join(e1, e2, e3, e4, e5); err != nil {
		t.Fatalf("error inserting players: %v", err)
	}

	// The rankings to insert into the database
	rankings := []struct {
		date     string
		rankings map[string]int32
	}{
		{
			date:     "2023-09-07",
			rankings: map[string]int32{p1.ID: 1, p2.ID: 2, p3.ID: 3, p4.ID: 4, p5.ID: 5},
		},
		{
			date:     "2023-09-13",
			rankings: map[string]int32{p1.ID: 2, p2.ID: 1, p3.ID: 5, p4.ID: 4, p5.ID: 3},
		},
		{
			date:     "2023-09-27",
			rankings: map[string]int32{p1.ID: 1, p2.ID: 2, p3.ID: 5, p4.ID: 4, p5.ID: 3},
		},
		{
			// Out of order on purpose to ensure that results are sorted by date correctly in ListRankings()
			date:     "2023-09-20",
			rankings: map[string]int32{p1.ID: 1, p2.ID: 2, p3.ID: 3, p4.ID: 4, p5.ID: 5},
		},
		{
			date:     "2023-10-04",
			rankings: map[string]int32{p1.ID: 5, p2.ID: 4, p3.ID: 3, p4.ID: 2, p5.ID: 1},
		},
	}

	var rankingID int32
	for _, r := range rankings {
		d, err := time.ParseInLocation(time.DateOnly, r.date, time.UTC)
		if err != nil {
			t.Fatalf("error parsing ranking date: %v", err)
		}

		ranking, err := testDB.AddRanking(ctx, d, r.rankings)
		if err != nil {
			t.Fatalf("error adding ranking for test: %v", err)
		}
		rankingID = ranking.ID
	}

	listResults, err := testDB.ListRankings(ctx)
	if err != nil {
		t.Fatalf("err was expected to be nil: %v", err)
	}

	// Make sure all of the expected dates are in the results
	expectedDates := []string{"2023-10-04", "2023-09-27", "2023-09-20", "2023-09-13", "2023-09-07"}
	for _, d := range expectedDates {
		found := false
		for _, r := range listResults {
			if r.Date.Format(time.DateOnly) == d {
				found = true
				continue
			}
		}

		if !found {
			t.Errorf("did not find expected date %s in listResult", d)
		}
	}

	// Get the first ranking
	getResult, err := testDB.GetRanking(ctx, rankingID)
	if err != nil {
		t.Fatalf("error getting ranking by id: %v", err)
	}
	assertEquals(t, "getResult.Date", "2023-10-04", getResult.Date.Format(time.DateOnly))
	expectedRankings := map[string]model.RankingPlayer{
		p5.ID: {Rank: 1, ID: p5.ID, FirstName: p5.FirstName, LastName: p5.LastName, Position: p5.Position, Team: p5.Team},
		p4.ID: {Rank: 2, ID: p4.ID, FirstName: p4.FirstName, LastName: p4.LastName, Position: p4.Position, Team: p4.Team},
		p3.ID: {Rank: 3, ID: p3.ID, FirstName: p3.FirstName, LastName: p3.LastName, Position: p3.Position, Team: p3.Team},
		p2.ID: {Rank: 4, ID: p2.ID, FirstName: p2.FirstName, LastName: p2.LastName, Position: p2.Position, Team: p2.Team},
		p1.ID: {Rank: 5, ID: p1.ID, FirstName: p1.FirstName, LastName: p1.LastName, Position: p1.Position, Team: p1.Team},
	}
	if !reflect.DeepEqual(expectedRankings, getResult.Players) {
		t.Errorf("expectedRanking != getResult.Players, got: %v", getResult.Players)
	}
}

func TestAddRanking_negativeCases(t *testing.T) {
	p1 := getPlayerWithName("Justin", "Jefferson")
	p2 := getPlayerWithName("Ja'Marr", "Chase")
	p3 := getPlayerWithName("Tyreek", "Hill")
	p4 := getPlayerWithName("Stefon", "Diggs")
	p5 := getPlayerWithName("A.J.", "Brown")

	ctx := context.Background()
	e1 := testDB.SavePlayer(ctx, p1)
	e2 := testDB.SavePlayer(ctx, p2)
	e3 := testDB.SavePlayer(ctx, p3)
	e4 := testDB.SavePlayer(ctx, p4)
	e5 := testDB.SavePlayer(ctx, p5)
	if err := errors.Join(e1, e2, e3, e4, e5); err != nil {
		t.Fatalf("error inserting players: %v", err)
	}

	tests := []struct {
		name     string
		date     string
		rankings map[string]int32
		err      error
	}{
		{
			name:     "zero date",
			date:     "",
			rankings: map[string]int32{p1.ID: 4, p2.ID: 10, p3.ID: 27, p4.ID: 99, p5.ID: 132},
			err:      errors.New("rankings date must be provided"),
		},
		{
			name:     "nil rankings",
			date:     "2023-09-01",
			rankings: nil,
			err:      errors.New("rankings cannot be empty"),
		},
		{
			name:     "empty rankings",
			date:     "2023-09-01",
			rankings: map[string]int32{},
			err:      errors.New("rankings cannot be empty"),
		},
		{
			name:     "invalid player id",
			date:     "2023-09-01",
			rankings: map[string]int32{p1.ID: 4, p2.ID: 10, p3.ID: 27, "9999": 99, p5.ID: 132},
			err:      errors.New("no player with id: 9999"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var rankingDate time.Time
			var err error

			if tc.date != "" {
				rankingDate, err = time.ParseInLocation(time.DateOnly, tc.date, time.UTC)
				if err != nil {
					t.Errorf("error parsing date in test %s: %v", tc.name, err)
				}
			}

			res, err := testDB.AddRanking(ctx, rankingDate, tc.rankings)
			assertError(t, tc.name, tc.err, err)
			if res != nil {
				t.Error("expected res to be nil")
			}
		})
	}
}

func TestLeagues(t *testing.T) {
	ctx := context.Background()

	l1 := model.League{
		Platform:   model.PlatformSleeper,
		ExternalID: "1",
		Name:       "League 1",
		Year:       "2024",
	}

	l2 := model.League{
		Platform:   model.PlatformSleeper,
		ExternalID: "2",
		Name:       "League 2",
		Year:       "2024",
	}
	// Clean up after the test
	defer func() {
		testDB.ArchiveLeague(ctx, l1.ID)
		testDB.ArchiveLeague(ctx, l2.ID)
	}()

	err := testDB.AddLeague(ctx, &l1)
	if err != nil {
		t.Fatalf("unexpected error adding league: %v", err)
	}

	err = testDB.AddLeague(ctx, &l2)
	if err != nil {
		t.Fatalf("unexpected error adding league: %v", err)
	}

	leagues, err := testDB.ListLeagues(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing leagues: %v", err)
	}
	if len(leagues) != 2 {
		t.Fatalf("expected to list 2 leagues, but got %d", len(leagues))
	}
	if leagues[0].ExternalID != "1" {
		t.Errorf("unexpected external id, wanted 1 got: %s", leagues[0].ExternalID)
	}
	if leagues[1].ExternalID != "2" {
		t.Errorf("unexpected external id, wanted 1 got: %s", leagues[1].ExternalID)
	}

	r1, err := testDB.GetLeague(ctx, l1.ID)
	if err != nil {
		t.Fatalf("error getting league by id: %v", err)
	}
	if !reflect.DeepEqual(&l1, r1) {
		t.Errorf("league values not as expected - wanted: %v, got: %v", &l1, r1)
	}

	e1 := testDB.ArchiveLeague(ctx, l1.ID)
	e2 := testDB.ArchiveLeague(ctx, l2.ID)
	if err := errors.Join(e1, e2); err != nil {
		t.Errorf("expected no errors but was: %v", err)
	}

	leagues, err = testDB.ListLeagues(ctx)
	if err != nil {
		t.Errorf("error getting leagues: %v", err)
	}
	if len(leagues) != 0 {
		t.Errorf("expected 0 leagues, instead got: %d", len(leagues))
	}
}

func TestTokens(t *testing.T) {
	ctx := context.Background()

	l := getLeague()
	if err := testDB.AddLeague(ctx, l); err != nil {
		t.Fatalf("error adding league: %v", err)
	}
	// Clean up after the test
	defer func() {
		testDB.ArchiveLeague(ctx, l.ID)
	}()

	expiry, err := time.ParseInLocation(time.DateTime, "2024-09-09 14:45:01", time.UTC)
	if err != nil {
		t.Fatalf("error parsing date time: %v", err)
	}

	token := &oauth2.Token{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		Expiry:       expiry,
	}

	if err := testDB.SaveToken(ctx, l.ID, token); err != nil {
		t.Fatalf("error saving token: %v", err)
	}

	t2, err := testDB.GetToken(ctx, l.ID)
	if err != nil {
		t.Fatalf("error getting token by league id: %v", err)
	}
	if !reflect.DeepEqual(token, t2) {
		t.Errorf("expected token to be: %v, but got: %v", token, t2)
	}

	t2.AccessToken = "access-token-2"
	t2.RefreshToken = "refresh-token-2"
	if err := testDB.SaveToken(ctx, l.ID, t2); err != nil {
		t.Fatalf("error saving updating token: %v", err)
	}

	t3, err := testDB.GetToken(ctx, l.ID)
	if err != nil {
		t.Fatalf("error getting updated token: %v", err)
	}

	if t3.AccessToken != t2.AccessToken {
		t.Errorf("access token not updated, expected %s, got %s", t2.AccessToken, t3.AccessToken)
	}
	if t3.RefreshToken != t2.RefreshToken {
		t.Errorf("refresh token not updated, expected %s, got %s", t2.RefreshToken, t3.RefreshToken)
	}
}

func TestLeagueManagers(t *testing.T) {
	ctx := context.Background()
	// A league to add managers to
	l := getLeague()

	if err := testDB.AddLeague(ctx, l); err != nil {
		t.Fatalf("error adding league: %v", err)
	}
	// Clean up after the test
	defer func() {
		testDB.ArchiveLeague(ctx, l.ID)
	}()

	m1 := getLeagueManager()
	m2 := getLeagueManager()
	m3 := getLeagueManager()
	for _, m := range []*model.LeagueManager{m1, m2, m3} {
		if err := testDB.SaveLeagueManager(ctx, l.ID, m); err != nil {
			t.Fatalf("error adding manager to league: %v", err)
		}
	}

	found, err := testDB.GetLeagueManagers(ctx, l.ID)
	if err != nil {
		t.Fatalf("error getting league managers: %v", err)
	}
	expected := []model.LeagueManager{*m1, *m2, *m3}
	if !reflect.DeepEqual(expected, found) {
		t.Fatalf("expected leagues not found, got: %v", found)
	}

	// Update a record
	m2.TeamName = "New team name"
	if err := testDB.SaveLeagueManager(ctx, l.ID, m2); err != nil {
		t.Fatalf("error saving updated team name: %v", err)
	}

	found, err = testDB.GetLeagueManagers(ctx, l.ID)
	if err != nil {
		t.Fatalf("error getting league managers: %v", err)
	}
	if len(found) != 3 {
		t.Fatalf("expected to find 3 manager, found: %d", len(found))
	}
	if found[1].TeamName != "New team name" {
		t.Fatal("TeamName for m2 not updated as expected")
	}
}

func TestResults(t *testing.T) {
	ctx := context.Background()
	// A league and managers
	l := getLeague()

	if err := testDB.AddLeague(ctx, l); err != nil {
		t.Fatalf("error adding league: %v", err)
	}
	// Clean up after the test
	defer func() {
		testDB.ArchiveLeague(ctx, l.ID)
	}()

	m1 := getLeagueManager()
	m2 := getLeagueManager()
	m3 := getLeagueManager()
	m4 := getLeagueManager()
	for _, m := range []*model.LeagueManager{m1, m2, m3, m4} {
		if err := testDB.SaveLeagueManager(ctx, l.ID, m); err != nil {
			t.Fatalf("error adding manager to league: %v", err)
		}
	}

	matchups := []model.Matchup{
		{
			MatchupID: 1,
			Week:      2,
			TeamA:     &model.TeamResult{TeamID: m1.ExternalID, Score: 100000},
			TeamB:     &model.TeamResult{TeamID: m2.ExternalID, Score: 101000},
		},
		{
			MatchupID: 2,
			Week:      2,
			TeamA:     &model.TeamResult{TeamID: m3.ExternalID, Score: 99100},
			TeamB:     &model.TeamResult{TeamID: m4.ExternalID, Score: 103550},
		},
		{
			MatchupID: 3,
			Week:      2,
			TeamA:     &model.TeamResult{TeamID: m1.ExternalID, Score: 100000},
			TeamB:     &model.TeamResult{TeamID: m3.ExternalID, Score: 99100},
		},
		{
			MatchupID: 4,
			Week:      2,
			TeamA:     &model.TeamResult{TeamID: m2.ExternalID, Score: 101000},
			TeamB:     &model.TeamResult{TeamID: m4.ExternalID, Score: 103550},
		},
	}

	if err := testDB.SaveResults(ctx, l.ID, matchups); err != nil {
		t.Fatalf("error saving matchup results: %v", err)
	}

	matchups, err := testDB.GetResults(ctx, l.ID, 2)
	if err != nil {
		t.Fatalf("error getting matchup results: %v", err)
	}
	if len(matchups) != 4 {
		t.Errorf("expected 4 matchups, but got: %d", len(matchups))
	}

	t1 := &model.TeamResult{TeamID: m1.ExternalID, TeamName: m1.TeamName, Score: 100000}
	t2 := &model.TeamResult{TeamID: m2.ExternalID, TeamName: m2.TeamName, Score: 101000}
	t3 := &model.TeamResult{TeamID: m3.ExternalID, TeamName: m3.TeamName, Score: 99100}
	t4 := &model.TeamResult{TeamID: m4.ExternalID, TeamName: m4.TeamName, Score: 103550}

	for i, m := range matchups {
		switch i {
		case 0:
			if !reflect.DeepEqual(t1, m.TeamA) || !reflect.DeepEqual(t2, m.TeamB) {
				t.Errorf("matchup 1 (id: %d) expected t1 and t2 got: %v, %v", m.MatchupID, m.TeamA, m.TeamB)
			}
		case 1:
			if !reflect.DeepEqual(t3, m.TeamA) || !reflect.DeepEqual(t4, m.TeamB) {
				t.Errorf("matchup 2 (id: %d) expected t3 and t4 got: %v, %v", m.MatchupID, m.TeamA, m.TeamB)
			}
		case 2:
			if !reflect.DeepEqual(t1, m.TeamA) || !reflect.DeepEqual(t3, m.TeamB) {
				t.Errorf("matchup 2 (id: %d) expected t1 and t3 got: %v, %v", m.MatchupID, m.TeamA, m.TeamB)
			}
		case 3:
			if !reflect.DeepEqual(t2, m.TeamA) || !reflect.DeepEqual(t4, m.TeamB) {
				t.Errorf("matchup 2 (id: %d) expected t2 and t4 got: %v, %v", m.MatchupID, m.TeamA, m.TeamB)
			}
		default:
			t.Fatalf("unexpected matchup result: %d", i)
		}
	}

	// Add another weeks results so the list has a little bit more data
	week5 := []model.Matchup{
		{
			MatchupID: 1,
			Week:      5,
			TeamA:     &model.TeamResult{TeamID: m1.ExternalID, Score: 100000},
			TeamB:     &model.TeamResult{TeamID: m4.ExternalID, Score: 101000},
		},
		{
			MatchupID: 2,
			Week:      5,
			TeamA:     &model.TeamResult{TeamID: m2.ExternalID, Score: 99100},
			TeamB:     &model.TeamResult{TeamID: m3.ExternalID, Score: 103550},
		},
		{
			MatchupID: 3,
			Week:      5,
			TeamA:     &model.TeamResult{TeamID: m2.ExternalID, Score: 100000},
			TeamB:     &model.TeamResult{TeamID: m1.ExternalID, Score: 99100},
		},
		{
			MatchupID: 4,
			Week:      5,
			TeamA:     &model.TeamResult{TeamID: m4.ExternalID, Score: 101000},
			TeamB:     &model.TeamResult{TeamID: m3.ExternalID, Score: 103550},
		},
	}
	if err := testDB.SaveResults(ctx, l.ID, week5); err != nil {
		t.Fatalf("error saving week5 matchup results: %v", err)
	}

	weeks, err := testDB.ListResultWeeks(ctx, l.ID)
	if err != nil {
		t.Fatalf("unexpected error listing result weeks: %v", err)
	}
	if !reflect.DeepEqual(weeks, []int{2, 5}) {
		t.Errorf("result weeks were not expected, got: %v", weeks)
	}
}

func TestPowerRankings(t *testing.T) {
	ctx := context.Background()
	// A league
	l := getLeague()
	if err := testDB.AddLeague(ctx, l); err != nil {
		t.Fatalf("error adding league: %v", err)
	}
	defer func() {
		testDB.ArchiveLeague(ctx, l.ID) // Clean up after the test
	}()

	// And managers
	m1 := getLeagueManager()
	m2 := getLeagueManager()
	for _, m := range []*model.LeagueManager{m1, m2} {
		if err := testDB.SaveLeagueManager(ctx, l.ID, m); err != nil {
			t.Fatalf("error adding manager to league: %v", err)
		}
	}

	// And players
	p1 := getPlayer()
	p2 := getPlayer()
	p3 := getPlayer()
	p4 := getPlayer()
	for _, p := range []*model.Player{p1, p2, p3, p4} {
		if err := testDB.SavePlayer(ctx, p); err != nil {
			t.Fatalf("error adding player: %v", err)
		}
	}

	playerRanks := map[string]int32{
		p1.ID: 1,
		p2.ID: 2,
		p3.ID: 3,
		p4.ID: 4,
	}
	// Make the date before any of the ones in TestRankings() to keep
	// the list order working.
	rankingDate, _ := time.Parse(time.DateOnly, "2022-10-11")
	ranking, err := testDB.AddRanking(ctx, rankingDate, playerRanks)
	if err != nil {
		t.Fatalf("error adding ranking: %v", err)
	}

	pr1 := &model.PowerRanking{
		RankingID: ranking.ID,
		Week:      0,
		Teams: []model.TeamPowerRanking{
			{
				TeamID:      m1.ExternalID,
				Rank:        1,
				TotalScore:  10111,
				RosterScore: 10111,
				Roster: []model.PowerRankingPlayer{
					{
						PlayerID:           p1.ID,
						Rank:               1,
						NFLTeam:            model.TEAM_ARI,
						PowerRankingPoints: 1000,
						IsStarter:          true,
					},
					{
						PlayerID:           p2.ID,
						Rank:               2,
						NFLTeam:            model.TEAM_BUF,
						PowerRankingPoints: 999,
						IsStarter:          false,
					},
				},
			},
			{
				TeamID:      m2.ExternalID,
				Rank:        2,
				TotalScore:  10022,
				RosterScore: 10022,
				Roster: []model.PowerRankingPlayer{
					{
						PlayerID:           p3.ID,
						Rank:               3,
						NFLTeam:            model.TEAM_CAR,
						PowerRankingPoints: 888,
						IsStarter:          true,
					},
					{
						PlayerID:           p4.ID,
						Rank:               4,
						NFLTeam:            model.TEAM_DAL,
						PowerRankingPoints: 777,
						IsStarter:          false,
					},
				},
			},
		},
	}
	id, err := testDB.SavePowerRanking(ctx, l.ID, pr1)
	if err != nil {
		t.Fatalf("error saving power ranking 1: %v", err)
	}

	res, err := testDB.GetPowerRanking(ctx, l.ID, id)
	if err != nil {
		t.Fatalf("error looking up power ranking 1: %v", err)
	}

	if len(res.Teams) != 2 {
		t.Errorf("unexpected number of teams, wanted 2 got %d", len(res.Teams))
	}
	if res.Teams[0].Rank != 1 {
		t.Errorf("Team 0 should have rank 1, not %d", res.Teams[0].Rank)
	}
	if res.Teams[0].TeamID != m1.ExternalID {
		t.Errorf("Unexpected team at top of rankings: %s", res.Teams[0].TeamID)
	}
	if res.Teams[0].Roster[0].PlayerID != p1.ID {
		t.Errorf("Unexpected player at top of roster for team 0 - wanted %s, got %s", p1.ID, res.Teams[0].Roster[0].PlayerID)
	}
	if res.Teams[1].Rank != 2 {
		t.Errorf("Team 1 should have rank 2, not %d", res.Teams[1].Rank)
	}
	if res.Teams[1].TeamID != m2.ExternalID {
		t.Errorf("Unexpected team at top of rankings: %s", res.Teams[1].TeamID)
	}
	if res.Teams[1].Roster[0].PlayerID != p3.ID {
		t.Errorf("Unexpected player at top of roster for team 1 - got %s, got %s", p3.ID, res.Teams[1].Roster[0].PlayerID)
	}

	// Add a second power rankings and then list them.
	pr2 := &model.PowerRanking{
		RankingID: ranking.ID,
		Week:      1,
		Teams: []model.TeamPowerRanking{
			{
				TeamID:      m2.ExternalID,
				Rank:        1,
				TotalScore:  11000,
				RosterScore: 11000,
				Roster: []model.PowerRankingPlayer{
					{
						PlayerID:           p3.ID,
						Rank:               3,
						NFLTeam:            model.TEAM_CAR,
						PowerRankingPoints: 888,
						IsStarter:          true,
					},
					{
						PlayerID:           p4.ID,
						Rank:               4,
						NFLTeam:            model.TEAM_DAL,
						PowerRankingPoints: 777,
						IsStarter:          false,
					},
				},
			},
			{
				TeamID:      m1.ExternalID,
				Rank:        2,
				TotalScore:  10111,
				RosterScore: 10111,
				Roster: []model.PowerRankingPlayer{
					{
						PlayerID:           p1.ID,
						Rank:               1,
						NFLTeam:            model.TEAM_ARI,
						PowerRankingPoints: 1000,
						IsStarter:          true,
					},
					{
						PlayerID:           p2.ID,
						Rank:               2,
						NFLTeam:            model.TEAM_BUF,
						PowerRankingPoints: 999,
						IsStarter:          false,
					},
				},
			},
		},
	}
	if _, err := testDB.SavePowerRanking(ctx, l.ID, pr2); err != nil {
		t.Fatalf("error saving power ranking 2: %v", err)
	}

	rankings, err := testDB.ListPowerRankings(ctx, l.ID)
	if err != nil {
		t.Fatalf("error listing power rankings: %v", err)
	}

	if len(rankings) != 2 {
		t.Errorf("expected 2 power rankings, got %d", len(rankings))
	}
	if rankings[0].Week != 1 {
		t.Errorf("expected first power rankings to have week=1, got: %d", rankings[0].Week)
	}
	if rankings[1].Week != 0 {
		t.Errorf("expected second power rankings to have week=1, got: %d", rankings[1].Week)
	}
}

func TestPowerRankings_leagueWithNoRankings(t *testing.T) {
	ctx := context.Background()
	// A league
	l := getLeague()
	if err := testDB.AddLeague(ctx, l); err != nil {
		t.Fatalf("error adding league: %v", err)
	}
	defer func() {
		testDB.ArchiveLeague(ctx, l.ID) // Clean up after the test
	}()

	rankings, err := testDB.ListPowerRankings(ctx, l.ID)
	if err != nil {
		t.Fatalf("unexpected error listing rankings: %v", err)
	}
	if len(rankings) != 0 {
		t.Fatalf("expected 0 rankings, but got %d", len(rankings))
	}
}

func getPlayer() *model.Player {
	id := atomic.AddInt32(&idCtr, 1)

	return &model.Player{
		ID:              fmt.Sprintf("%d", id),
		YahooID:         "28457",
		FirstName:       "Tyler",
		LastName:        "Lockett",
		Nickname1:       "Hot Locket",
		Position:        model.POS_WR,
		Team:            model.TEAM_SEA,
		Weight:          182,
		Height:          70,
		BirthDate:       time.Date(1992, 9, 28, 0, 0, 0, 0, time.UTC),
		RookieYear:      time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
		YearsExp:        9,
		Jersey:          16,
		DepthChartOrder: 2,
		College:         "Kansas State",
		Active:          true,
	}
}

func getPlayerWithName(first, last string) *model.Player {
	id := atomic.AddInt32(&idCtr, 1)

	return &model.Player{
		ID:        fmt.Sprintf("%d", id),
		FirstName: first,
		LastName:  last,
		Position:  model.POS_WR,
		Team:      model.TEAM_DET,
	}
}

func getLeague() *model.League {
	id := atomic.AddInt32(&idCtr, 1)

	return &model.League{
		Platform:   model.PlatformSleeper,
		ExternalID: fmt.Sprint(id),
		Name:       fmt.Sprintf("League %d", id),
		Year:       "2024",
	}
}

func getLeagueManager() *model.LeagueManager {
	id := atomic.AddInt32(&idCtr, 1)

	return &model.LeagueManager{
		ExternalID:  fmt.Sprint(id),
		TeamName:    fmt.Sprintf("Team %d", id),
		ManagerName: fmt.Sprintf("Manager Name %d", id),
		JoinKey:     fmt.Sprint(id),
	}
}

func assertFatalf(t *testing.T, c bool, f string, args ...any) {
	if !c {
		t.Fatalf(f, args...)
	}
}

func assertEquals(t *testing.T, field string, expected, actual any) {
	if expected != actual {
		t.Errorf("%s - expected: '%s', got: '%s'", field, expected, actual)
	}
}

func assertError(t *testing.T, tcName string, e1, e2 error) {
	if e1 == nil && e2 == nil {
		return
	}
	if (e1 != nil && e2 == nil) || (e1 == nil && e2 != nil) {
		t.Errorf("unexpected error in %s, expected: %v, got: %v", tcName, e1, e2)
		return
	}
	if e1.Error() != e2.Error() {
		t.Errorf("errors are not equal in %s, expected: %v, got: %v", tcName, e1, e2)
	}
}
