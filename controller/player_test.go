package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/mww/fantasy_manager_v2/db"
	"github.com/mww/fantasy_manager_v2/db/mockdb"
	"github.com/mww/fantasy_manager_v2/model"
	"github.com/mww/fantasy_manager_v2/sleeper"
	"github.com/mww/fantasy_manager_v2/sleeper/mocksleeper"
	"github.com/stretchr/testify/mock"
)

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

func TestSearch(t *testing.T) {
	sleeper, err := sleeper.New()
	if err != nil {
		t.Fatalf("error getting sleeper client: %v", err)
	}

	mockResults := []model.Player{
		{ID: "1", FirstName: "Player1", LastName: "Last1"},
		{ID: "2", FirstName: "Player2", LastName: "Last2"},
	}

	tests := map[string]struct {
		q   string
		res []model.Player
		err error
		// The expected arguments to the db call
		exQ string
		exP model.Position
		exT *model.NFLTeam
	}{
		"positive plain":     {q: "Christian McCaffrey", res: mockResults, exQ: "Christian McCaffrey", exP: model.POS_UNKNOWN, exT: nil},
		"positive both":      {q: "AJ Brown team:PHI pos:WR", res: mockResults, exQ: "AJ Brown", exP: model.POS_WR, exT: model.TEAM_PHI},
		"positive just team": {q: "CeeDee Lamb team:cowboys", res: mockResults, exQ: "CeeDee Lamb", exP: model.POS_UNKNOWN, exT: model.TEAM_DAL},
		"positive just pos":  {q: "Ken Walker pos:RB", res: mockResults, exQ: "Ken Walker", exP: model.POS_RB, exT: nil},
		"empty":              {q: "", exQ: "", res: nil, err: fmt.Errorf("error not a valid query: ''"), exP: model.POS_UNKNOWN},
		"db error":           {q: "Jalen Hurts", res: nil, err: errors.New("db error"), exQ: "Jalen Hurts", exP: model.POS_UNKNOWN, exT: nil},
	}

	for name, tc := range tests {
		mockDB := &mockdb.DB{}
		ctrl, err := New(sleeper, mockDB)
		if err != nil {
			t.Fatalf("error constructing controller: %v", err)
		}

		t.Run(name, func(t *testing.T) {
			if tc.exQ != "" || tc.exP != model.POS_UNKNOWN || tc.exT != nil {
				mockDB.On("Search", mock.Anything, tc.exQ, tc.exP, tc.exT).Return(tc.res, tc.err)
			}

			res, err := ctrl.Search(context.Background(), tc.q)
			if !reflect.DeepEqual(res, tc.res) {
				t.Errorf("result was not the expected value")
			}
			if !errorsEqual(err, tc.err) {
				t.Errorf("unexpected err value, wanted: '%v', got: '%v'", tc.err, err)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestUpdatePlayerNickname(t *testing.T) {
	sleeper, err := sleeper.New()
	if err != nil {
		t.Fatalf("error getting sleeper client: %v", err)
	}

	// These are modified by the tests, so don't reuse them between tests
	p1 := &model.Player{ID: "1", FirstName: "Tyler", LastName: "Lockett"}
	p2 := &model.Player{ID: "2", FirstName: "Tyler", LastName: "Lockett", Nickname1: "Hot Locket"}
	p3 := &model.Player{ID: "3", FirstName: "Josh", LastName: "Jacobs", Nickname1: "Fat Thor"}
	p4 := &model.Player{ID: "4", FirstName: "TJ", LastName: "Hockenson"}

	saveErr := errors.New("some error saving a player")

	tests := map[string]struct {
		id      string
		p       *model.Player
		nn      string
		err     error
		saveEx  bool // if the save call is expected or not
		saveErr error
	}{
		"simple add":      {id: p1.ID, p: p1, nn: "nickname", err: nil, saveEx: true, saveErr: nil},
		"no player found": {id: "20", p: nil, nn: "nickname", err: db.ErrPlayerNotFound, saveEx: false},
		"nn already set":  {id: p2.ID, p: p2, nn: p2.Nickname1, err: errors.New("no updated needed"), saveEx: false},
		"delete nn":       {id: p3.ID, p: p3, nn: "", err: nil, saveEx: true, saveErr: nil},
		"save error":      {id: p4.ID, p: p4, nn: "The HockStrap", err: saveErr, saveEx: true, saveErr: saveErr},
	}

	for name, tc := range tests {
		mockDB := &mockdb.DB{}
		ctrl, err := New(sleeper, mockDB)
		if err != nil {
			t.Fatalf("error constructing controller: %v", err)
		}

		t.Run(name, func(t *testing.T) {
			if tc.p != nil {
				mockDB.On("GetPlayer", mock.Anything, tc.id).Return(tc.p, nil)
			} else {
				mockDB.On("GetPlayer", mock.Anything, tc.id).Return(nil, db.ErrPlayerNotFound)
			}

			if tc.saveEx {
				if tc.nn == "" {
					mockDB.On("DeleteNickname", mock.Anything, tc.id, tc.p.Nickname1).Return(tc.saveErr)
				} else {
					mockDB.On("SavePlayer", mock.Anything, tc.p).Return(tc.saveErr)
				}
			}

			err = ctrl.UpdatePlayerNickname(context.Background(), tc.id, tc.nn)
			if !errorsEqual(tc.err, err) {
				t.Errorf("expected err '%v', got '%v'", tc.err, err)
			}

			mockDB.AssertExpectations(t)
			if !tc.saveEx {
				mockDB.AssertNotCalled(t, "SavePlayer", mock.Anything, tc.p)
			}
			if tc.nn != "" {
				mockDB.AssertNotCalled(t, "DeleteNickname", mock.Anything, tc.id)
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

	// TODO: flesh out tests more
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockSleeper := mocksleeper.Client{}
			mockDB := mockdb.DB{}

			ctrl, err := New(&mockSleeper, &mockDB)
			if err != nil {
				t.Fatalf("error constructing controller: %v", err)
			}

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
	db := &mockdb.DB{}

	ctrl, err := New(sleeper, db)
	if err != nil {
		t.Fatalf("error creating controller: %v", err)
	}

	players := []model.Player{
		{ID: "1", FirstName: "One", LastName: "LastOne", Position: model.POS_QB},
		{ID: "2", FirstName: "Two", LastName: "LastTwo", Position: model.POS_WR},
		{ID: "3", FirstName: "Three", LastName: "LastThree", Position: model.POS_RB},
		{ID: "4", FirstName: "Four", LastName: "LastFour", Position: model.POS_TE},
	}

	sleeper.On("LoadPlayers").Return(players, nil)
	db.On("SavePlayer", mock.Anything, &players[0]).Return(nil)
	db.On("SavePlayer", mock.Anything, &players[1]).Return(nil)
	db.On("SavePlayer", mock.Anything, &players[2]).Return(nil)
	db.On("SavePlayer", mock.Anything, &players[3]).Return(nil)

	err = ctrl.UpdatePlayers(context.Background())
	if err != nil {
		t.Errorf("error updating players: %v", err)
	}

	sleeper.AssertExpectations(t)
	db.AssertExpectations(t)
}

func TestUpdatePlayers_sleeperError(t *testing.T) {
	sleeper := &mocksleeper.Client{}
	db := &mockdb.DB{}

	ctrl, err := New(sleeper, db)
	if err != nil {
		t.Fatalf("error creating controller: %v", err)
	}

	sleeper.On("LoadPlayers").Return(nil, errors.New("error from sleeper"))

	err = ctrl.UpdatePlayers(context.Background())
	if !errorsEqual(err, errors.New("error from sleeper")) {
		t.Errorf("not the expected error: '%v'", err)
	}

	sleeper.AssertExpectations(t)
	db.AssertNotCalled(t, "SavePlayer", mock.Anything, mock.Anything)
}

func TestUpdatePlayers_dbError(t *testing.T) {
	sleeper := &mocksleeper.Client{}
	db := &mockdb.DB{}

	ctrl, err := New(sleeper, db)
	if err != nil {
		t.Fatalf("error creating controller: %v", err)
	}

	players := []model.Player{
		{ID: "1", FirstName: "One", LastName: "LastOne", Position: model.POS_QB},
		{ID: "2", FirstName: "Two", LastName: "LastTwo", Position: model.POS_WR},
		{ID: "3", FirstName: "Three", LastName: "LastThree", Position: model.POS_RB},
		{ID: "4", FirstName: "Four", LastName: "LastFour", Position: model.POS_TE},
	}

	sleeper.On("LoadPlayers").Return(players, nil)
	db.On("SavePlayer", mock.Anything, &players[0]).Return(nil)
	db.On("SavePlayer", mock.Anything, &players[1]).Return(errors.New("this error"))

	err = ctrl.UpdatePlayers(context.Background())
	if !errorsEqual(err, errors.New("error saving player (Two LastTwo): this error")) {
		t.Errorf("not the expected error: '%v'", err)
	}

	sleeper.AssertExpectations(t)
	db.AssertExpectations(t)
}

func TestRunPeriodicPlayerUpdates(t *testing.T) {
	sleeper := &mocksleeper.Client{}
	db := &mockdb.DB{}

	ctrl, err := New(sleeper, db)
	if err != nil {
		t.Fatalf("error creating controller: %v", err)
	}

	players := []model.Player{
		{ID: "1", FirstName: "One", LastName: "LastOne", Position: model.POS_QB},
		{ID: "2", FirstName: "Two", LastName: "LastTwo", Position: model.POS_WR},
		{ID: "3", FirstName: "Three", LastName: "LastThree", Position: model.POS_RB},
		{ID: "4", FirstName: "Four", LastName: "LastFour", Position: model.POS_TE},
	}

	sleeper.On("LoadPlayers").Return(players, nil).Times(3)
	db.On("SavePlayer", mock.Anything, &players[0]).Return(nil).Times(3)
	db.On("SavePlayer", mock.Anything, &players[1]).Return(nil).Times(3)
	db.On("SavePlayer", mock.Anything, &players[2]).Return(nil).Times(3)
	db.On("SavePlayer", mock.Anything, &players[3]).Return(nil).Times(3)

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
	db.AssertExpectations(t)
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
