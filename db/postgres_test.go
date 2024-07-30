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

func TestDB_saveAndLoad(t *testing.T) {
	ctx := context.Background()
	p := getPlayer()

	err := testDB.SavePlayer(ctx, p)
	assertFatalf(t, err == nil, "error saving player: %v", err)

	res, err := testDB.GetPlayer(ctx, p.ID)
	assertFatalf(t, err == nil, "error retreiving player: %v", err)

	// Make sure that the after saving and retreiving the player, all the fields
	// are the same.
	assertEquals(t, "ID", p.ID, res.ID)
	assertEquals(t, "YahooID", p.YahooID, res.YahooID)
	assertEquals(t, "FirstName", p.FirstName, res.FirstName)
	assertEquals(t, "LastName", p.LastName, res.LastName)
	assertEquals(t, "Nickname1", p.Nickname1, res.Nickname1)
	assertEquals(t, "Position", p.Position, res.Position)
	assertEquals(t, "Team", p.Team, res.Team)
	assertEquals(t, "Weight", p.Weight, res.Weight)
	assertEquals(t, "Height", p.Height, res.Height)
	assertEquals(t, "BirthDate", p.BirthDate, res.BirthDate)
	assertEquals(t, "RookieYear", p.RookieYear, res.RookieYear)
	assertEquals(t, "YearsExp", p.YearsExp, res.YearsExp)
	assertEquals(t, "Jersey", p.Jersey, res.Jersey)
	assertEquals(t, "DepthChartOrder", p.DepthChartOrder, res.DepthChartOrder)
	assertEquals(t, "College", p.College, res.College)
	assertEquals(t, "Active", p.Active, res.Active)
	assertEquals(t, "player changes", 0, len(res.Changes))

	// The originals should not have their created or updated times set.
	if !p.Created.IsZero() {
		t.Errorf("expected created time to be zero")
	}
	if !p.Updated.IsZero() {
		t.Errorf("expected updated time to be zero")
	}

	// The result should have a created time, but not an updated time.
	if res.Created.IsZero() {
		t.Errorf("expected res created time to not be zero")
	}
	if !res.Updated.IsZero() {
		t.Errorf("expected res updated time to be zero")
	}

	// Now update a field and make sure it persists as expected.
	p.Weight = p.Weight - 5
	err = testDB.SavePlayer(ctx, p)
	assertFatalf(t, err == nil, "error saving player after update: %v", err)

	res2, err := testDB.GetPlayer(ctx, p.ID)
	assertFatalf(t, err == nil, "error getting updated player: %v", err)

	assertEquals(t, "Weight", p.Weight, res2.Weight)
	assertEquals(t, "Changes", 1, len(p.Changes))
	// Now updated should not be zero
	if res2.Updated.IsZero() {
		t.Errorf("expected res2 updated time to not be zero")
	}

	// Lookup a player that doesn't exist
	res3, err := testDB.GetPlayer(ctx, "1111")
	assertFatalf(t, err != nil, "should have had an error searching for player")
	assertEquals(t, "error type", true, errors.Is(err, ErrPlayerNotFound))
	if res3 != nil {
		t.Errorf("expected res3 to be nil, but was %v", res3)
	}
}

func TestDB_search(t *testing.T) {
	ctx := context.Background()

	// Change the player name since default player returned by getPlayer is used in several places
	// and may be in the DB multiple times.
	p := getPlayer()
	p.ID = "9999" // Set a static ID since we only ever want one player with this name in the DB
	p.FirstName = "DK"
	p.LastName = "Metcalf"
	p.Nickname1 = ""

	err := testDB.SavePlayer(ctx, p)
	assertFatalf(t, err == nil, "error saving player: %v", err)

	players, err := testDB.Search(ctx, "Metcalf", model.POS_UNKNOWN, nil)
	assertFatalf(t, err == nil, "error searching for player: %v", err)
	assertEquals(t, "num players found", 1, len(players))

	players, err = testDB.Search(ctx, "Frank", model.POS_UNKNOWN, nil)
	assertFatalf(t, err == nil, "error searching for players: %v", err)
	assertEquals(t, "num players found when searching for Frank", 0, len(players))

	// TODO: add tests for searching by position and team
}

func TestNicknames(t *testing.T) {
	ctx := context.Background()
	p := getPlayer()
	p.Nickname1 = "" // Make sure no nickname to start

	err := testDB.SavePlayer(ctx, p)
	assertFatalf(t, err == nil, "error saving player: %v", err)

	p1, err := testDB.GetPlayer(ctx, p.ID)
	assertFatalf(t, err == nil, "error fetching player: %v", err)
	assertEquals(t, "Nickname1", "", p1.Nickname1)
	if len(p1.Changes) != 0 {
		t.Errorf("should be 0 changes, but instead there are %d", len(p1.Changes))
	}

	p1.Nickname1 = "nickname"
	err = testDB.SavePlayer(ctx, p1)
	assertFatalf(t, err == nil, "error saving player: %v", err)

	// Verify the nickname has been saved
	p2, err := testDB.GetPlayer(ctx, p.ID)
	assertFatalf(t, err == nil, "error fetching player: %v", err)
	assertEquals(t, "Nickname1", "nickname", p2.Nickname1)
	if len(p2.Changes) != 1 {
		t.Errorf("should be 1 changes, but instead there are %d", len(p2.Changes))
	}
	assertPlayerChange(t, "change[0]", "Nickname1", "", "nickname", &p2.Changes[0])

	// Update the nickname to a new value
	p2.Nickname1 = "updated nickname"
	err = testDB.SavePlayer(ctx, p2)
	assertFatalf(t, err == nil, "error saving player: %v", err)

	// Verify the nickname has been updated and saved correctly
	p3, err := testDB.GetPlayer(ctx, p.ID)
	assertFatalf(t, err == nil, "error fetching player: %v", err)
	assertEquals(t, "Nickname1", "updated nickname", p3.Nickname1)
	if len(p3.Changes) != 2 {
		t.Errorf("should be 2 changes, but instead there are %d", len(p3.Changes))
	}
	assertPlayerChange(t, "change[0]", "Nickname1", "nickname", "updated nickname", &p3.Changes[0])
	assertPlayerChange(t, "change[1]", "Nickname1", "", "nickname", &p3.Changes[1])

	// Save the player with no nickname to make sure it isn't accidently deleted
	// This simulates getting an update from sleeper that doesn't contain the nickname.
	pNoNick := getPlayer()
	pNoNick.Nickname1 = ""
	err = testDB.SavePlayer(ctx, pNoNick)
	assertFatalf(t, err == nil, "error saving player: %v", err)
	pAfterUpdate, err := testDB.GetPlayer(ctx, p.ID)
	assertFatalf(t, err == nil, "error fetching player: %v", err)
	if !reflect.DeepEqual(p3, pAfterUpdate) {
		t.Fatalf("players are not equal after saving an empty nickname")
	}

	// Now delete the nickname
	err = testDB.DeleteNickname(ctx, p.ID, p3.Nickname1)
	assertFatalf(t, err == nil, "error deleting player nickname")

	// Verify the nickname has been deleted
	p4, err := testDB.GetPlayer(ctx, p.ID)
	assertFatalf(t, err == nil, "error fetching player: %v", err)
	assertEquals(t, "Nickname1", "", p4.Nickname1)
	if len(p4.Changes) != 3 {
		t.Errorf("should be 3 changes, but instead there are %d", len(p4.Changes))
	}
	assertPlayerChange(t, "change[0]", "Nickname1", "updated nickname", "", &p4.Changes[0])
	assertPlayerChange(t, "change[1]", "Nickname1", "nickname", "updated nickname", &p4.Changes[1])
	assertPlayerChange(t, "change[2]", "Nickname1", "", "nickname", &p4.Changes[2])
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

func assertPlayerChange(t *testing.T, key, exProp, exOld, exNew string, c *model.Change) {
	if exProp != c.PropertyName {
		t.Errorf("%s.PropertyName - expected: '%s', got: '%s'", key, exProp, c.PropertyName)
	}
	if exOld != c.OldValue {
		t.Errorf("%s.OldValue - expected: '%s', got: '%s'", key, exOld, c.OldValue)
	}
	if exNew != c.NewValue {
		t.Errorf("%s.NewValue - expected: '%s', got: '%s'", key, exNew, c.NewValue)
	}
}
