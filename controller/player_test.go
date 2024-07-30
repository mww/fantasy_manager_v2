package controller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/mww/fantasy_manager_v2/db"
	"github.com/mww/fantasy_manager_v2/model"
	"github.com/mww/fantasy_manager_v2/sleeper"
	"github.com/mww/fantasy_manager_v2/sleeper/mocksleeper"
	"github.com/mww/fantasy_manager_v2/testutils"
)

// A global testDB instance to use for all of the tests instead of setting up a new one each time.
var testDB *testutils.TestDB

// TestMain controls the main for the tests and allows for setup and shutdown of the tests
func TestMain(m *testing.M) {
	defer func() {
		// Catch all panics to make sure the shutdown is successfully run
		if r := recover(); r != nil {
			if testDB != nil {
				testDB.Shutdown()
			}
			fmt.Println("panic")
		}
	}()

	// Setup the global testDB variable
	testDB = testutils.NewTestDB()
	defer testDB.Shutdown()
	code := m.Run()
	os.Exit(code)
}

func TestGetPositionFromQuery(t *testing.T) {
	tests := map[string]struct {
		input     string
		wantQuery string
		wantPos   model.Position
	}{
		"position at end":    {input: "Tom Brady pos:QB", wantQuery: "Tom Brady", wantPos: model.POS_QB},
		"upper case POS":     {input: "Tom Brady POS:QB", wantQuery: "Tom Brady", wantPos: model.POS_QB},
		"position at start":  {input: "pos:QB Tom Brady", wantQuery: "Tom Brady", wantPos: model.POS_QB},
		"lower case pos":     {input: "DK Metcalf pos:wr", wantQuery: "DK Metcalf", wantPos: model.POS_WR},
		"position only":      {input: "pos:RB", wantQuery: "", wantPos: model.POS_RB},
		"no position":        {input: "TJ Hockenson", wantQuery: "TJ Hockenson", wantPos: model.POS_UNKNOWN},
		"unknown position":   {input: "Russell Wilson pos:QR", wantQuery: "Russell Wilson", wantPos: model.POS_UNKNOWN},
		"write out position": {input: "Tom Brady position:QB", wantQuery: "Tom Brady", wantPos: model.POS_QB},
		"space before :":     {input: "Tom Brady pos :QB", wantQuery: "Tom Brady", wantPos: model.POS_QB},
		"space after :":      {input: "Tom Brady pos: QB", wantQuery: "Tom Brady", wantPos: model.POS_QB},
		"spaces around :":    {input: "Tom Brady pos : QB", wantQuery: "Tom Brady", wantPos: model.POS_QB},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			q, pos := getPositionFromQuery(tc.input)
			if tc.wantQuery != q {
				t.Errorf("query incorrect, wanted: '%s', got: '%s'", tc.wantQuery, q)
			}
			if tc.wantPos != pos {
				t.Errorf("position incorrect, wanted: '%s', got: '%s'", tc.wantPos, pos)
			}
		})
	}
}

func TestGetTeamFromQuery(t *testing.T) {
	tests := map[string]struct {
		input     string
		wantQuery string
		wantTeam  *model.NFLTeam
	}{
		"team at end":     {input: "AJ Brown team:PHI", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"team at start":   {input: "team:PHI AJ Brown", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"uppercase TEAM":  {input: "TEAM:PHI AJ Brown", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"mascot":          {input: "team:eagles AJ Brown", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"city":            {input: "AJ Brown team:Philadelphia", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"nickname":        {input: "AJ Brown team:Philly", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"space before :":  {input: "AJ Brown team :PHI", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"space after :":   {input: "AJ Brown team: PHI", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"spaces around :": {input: "AJ Brown team : PHI", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"no team":         {input: "CeeDee Lamb", wantQuery: "CeeDee Lamb", wantTeam: nil},
		"bad team":        {input: "CeeDee Lamb team:puyallup", wantQuery: "CeeDee Lamb", wantTeam: nil},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			q, team := getTeamFromQuery(tc.input)
			if tc.wantQuery != q {
				t.Errorf("query incorrect, wanted: '%s', got: '%s'", tc.wantQuery, q)
			}
			if tc.wantTeam != team {
				t.Errorf("team incorrect, wanted: '%s', got: '%s'", tc.wantTeam, team)
			}
		})
	}
}

func TestGetPlayerSearchQuery(t *testing.T) {
	tests := map[string]struct {
		q     string
		exQ   string
		exP   model.Position
		exT   *model.NFLTeam
		exErr bool
	}{
		"positive plain":     {q: "Christian McCaffrey", exQ: "Christian McCaffrey", exP: model.POS_UNKNOWN, exT: nil, exErr: false},
		"positive both":      {q: "AJ Brown team:PHI pos:WR", exQ: "AJ Brown", exP: model.POS_WR, exT: model.TEAM_PHI, exErr: false},
		"positive just team": {q: "CeeDee Lamb team:cowboys", exQ: "CeeDee Lamb", exP: model.POS_UNKNOWN, exT: model.TEAM_DAL, exErr: false},
		"positive just pos":  {q: "Ken Walker pos:RB", exQ: "Ken Walker", exP: model.POS_RB, exT: nil, exErr: false},
		"empty":              {q: "", exQ: "", exP: model.POS_UNKNOWN, exT: nil, exErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			term, pos, team, err := getPlayerSearchQuery(tc.q)
			if tc.exErr && err == nil {
				t.Errorf("expected error, got none")
			}
			if !tc.exErr && err != nil {
				t.Errorf("did not expect error, but got: %v", err)
			}

			if term != tc.exQ {
				t.Errorf("expected term: '%s', got: '%s'", tc.exQ, term)
			}
			if pos != tc.exP {
				t.Errorf("expected position: '%v', got: '%v'", tc.exP, pos)
			}
			if team != tc.exT {
				t.Errorf("expected team: '%v', got '%v'", tc.exT, team)
			}
		})
	}
}

func TestUpdatePlayerNickname(t *testing.T) {
	ctx := context.Background()
	sleeper, err := sleeper.New()
	if err != nil {
		t.Fatalf("error getting sleeper client: %v", err)
	}

	// Using a slice to enforce test ordering.
	// Some tests rely on other tests being run first.
	tests := []struct {
		name string
		id   string
		nn   string
		err  error  // the expected error
		exNN string // the expected nickname after running UpdatePlayerNickname()
	}{
		{
			name: "simple add",
			id:   testutils.TylerLockett.ID,
			nn:   "Hot Locket",
			err:  nil,
			exNN: "Hot Locket",
		},
		{
			name: "nn already set",
			id:   testutils.TylerLockett.ID,
			nn:   "Hot Locket",
			err:  errors.New("no updated needed"),
			exNN: "Hot Locket",
		},
		{
			name: "no player found",
			id:   "111",
			nn:   "nickname",
			err:  db.ErrPlayerNotFound,
			exNN: "skip",
		},
		{
			name: "delete nickname",
			id:   testutils.TylerLockett.ID,
			nn:   "",
			err:  nil,
			exNN: "",
		},
	}

	ctrl, err := New(sleeper, testDB.DB)
	if err != nil {
		t.Fatalf("error constructing controller: %v", err)
	}

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {
			err = ctrl.UpdatePlayerNickname(ctx, tc.id, tc.nn)
			if !errorsEqual(tc.err, err) {
				t.Errorf("expected err '%v', got '%v'", tc.err, err)
			}

			if tc.exNN != "skip" {
				p, err := ctrl.GetPlayer(ctx, tc.id)
				if err != nil {
					t.Errorf("error looking up player to validate nickname: %v", err)
				}
				if p.Nickname1 != tc.exNN {
					t.Errorf("expected nickname: '%s', got: '%s'", tc.exNN, p.Nickname1)
				}
			}
		})
	}
}

func TestAddRankings(t *testing.T) {
	tests := map[string]struct {
		date          string
		expectedID    string
		expectedError error
	}{
		"simple add": {date: "2024-06-26", expectedID: "0", expectedError: nil},
	}

	mockSleeper := mocksleeper.Client{}
	ctrl, err := New(&mockSleeper, testDB.DB)
	if err != nil {
		t.Fatalf("error constructing controller: %v", err)
	}

	// TODO: flesh out tests more
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			date, _ := time.Parse(time.DateOnly, tc.date)

			id, err := ctrl.AddRankings(nil, date)
			if !errorsEqual(err, tc.expectedError) {
				t.Errorf("error not the same as expected. wanted: %v, got: %v", tc.expectedError, err)
			}
			if id != tc.expectedID {
				t.Errorf("id not the same as expected. wanted: %s, got: %s", tc.expectedID, id)
			}
		})
	}
}

func TestUpdatePlayers_success(t *testing.T) {
	sleeper := &mocksleeper.Client{}
	ctrl, err := New(sleeper, testDB.DB)
	if err != nil {
		t.Fatalf("error creating controller: %v", err)
	}

	players := []model.Player{
		*testutils.TylerLockett,
		*testutils.JalenHurts,
		*testutils.CeeDeeLamb,
		*testutils.TJHockenson,
		*testutils.BreeceHall,
	}

	sleeper.On("LoadPlayers").Return(players, nil)

	err = ctrl.UpdatePlayers(context.Background())
	if err != nil {
		t.Errorf("error updating players: %v", err)
	}

	sleeper.AssertExpectations(t)
}

func TestUpdatePlayers_sleeperError(t *testing.T) {
	sleeper := &mocksleeper.Client{}
	ctrl, err := New(sleeper, testDB.DB)
	if err != nil {
		t.Fatalf("error creating controller: %v", err)
	}

	sleeper.On("LoadPlayers").Return(nil, errors.New("error from sleeper"))

	err = ctrl.UpdatePlayers(context.Background())
	if !errorsEqual(err, errors.New("error from sleeper")) {
		t.Errorf("not the expected error: '%v'", err)
	}

	sleeper.AssertExpectations(t)
}

func TestRunPeriodicPlayerUpdates(t *testing.T) {
	sleeper := &mocksleeper.Client{}
	ctrl, err := New(sleeper, testDB.DB)
	if err != nil {
		t.Fatalf("error creating controller: %v", err)
	}

	players := []model.Player{
		*testutils.TylerLockett,
		*testutils.JalenHurts,
		*testutils.CeeDeeLamb,
		*testutils.TJHockenson,
		*testutils.BreeceHall,
	}

	sleeper.On("LoadPlayers").Return(players, nil).Times(3)

	shutdown := make(chan bool, 1)
	go func() {
		time.Sleep(160 * time.Millisecond) // enough time to run 3 times, but not 4
		close(shutdown)
	}()
	var wg sync.WaitGroup

	wg.Add(1)
	ctrl.RunPeriodicPlayerUpdates(50*time.Millisecond, shutdown, &wg)
	wg.Wait()

	sleeper.AssertExpectations(t)
}

func errorsEqual(e1, e2 error) bool {
	if e1 == nil && e2 == nil {
		return true
	}
	if (e1 != nil && e2 == nil) || (e1 == nil && e2 != nil) {
		return false
	}
	return e1.Error() == e2.Error()
}
