package db

import (
	"context"
	"errors"
	"log"
	"reflect"
	"testing"

	"github.com/mww/fantasy_manager_v2/model"
)

func TestPlayer_saveAndLoad(t *testing.T) {
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

	// Remove the yahoo id from res2, save the player and make sure that
	// the yahoo wasn't actually removed from the db. This tests when we
	// set the yahoo id and sleeper doesn't have the data. We want to make
	// sure the change isn't lost
	res2.YahooID = ""
	if err := testDB.SavePlayer(ctx, res2); err != nil {
		t.Fatalf("error saving player: %v", err)
	}
	res4, err := testDB.GetPlayer(ctx, res2.ID)
	if err != nil {
		t.Fatalf("error getting player: %v", err)
	}
	if res4.YahooID != p.YahooID {
		t.Errorf("expected yahoo id to be '%s', but was '%s'", p.YahooID, res4.YahooID)
	}
}

func TestPlayer_search(t *testing.T) {
	ctx := context.Background()

	// Change the player name since default player returned by getPlayer is used in several places
	// and may be in the DB multiple times.
	p := getPlayer()
	p.ID = "9998" // Set a static ID since we only ever want one player with this name in the DB
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

func TestPlayer_nicknames(t *testing.T) {
	ctx := context.Background()
	p := getPlayer()
	p.Nickname1 = ""
	p.DepthChartOrder = 1

	if err := testDB.SavePlayer(ctx, p); err != nil {
		t.Fatalf("error saving player: %v", err)
	}

	r1, err := testDB.GetPlayer(ctx, p.ID)
	if err != nil {
		t.Fatalf("error looking up player: %v", err)
	}
	if r1.Nickname1 != "" {
		t.Errorf("nickname was expected to not be set, got: '%s'", r1.Nickname1)
	}
	if r1.DepthChartOrder != 1 {
		t.Errorf("depth chart order is wrong, got: %d", r1.DepthChartOrder)
	}

	// Make a few changes to the player including setting a nickname
	p.DepthChartOrder = 2
	p.College = "University of Washington"
	p.Nickname1 = "superhero"
	if err := testDB.SavePlayer(ctx, p); err != nil {
		t.Fatalf("error saving player after updates: %v", err)
	}

	r2, err := testDB.GetPlayer(ctx, p.ID)
	if err != nil {
		t.Fatalf("error looking up player after updates: %v", err)
	}
	if r2.DepthChartOrder != 2 {
		t.Errorf("depth chart order is wrong, got: %d", r2.DepthChartOrder)
	}
	if r2.College != "University of Washington" {
		t.Errorf("college is not expected value: '%s'", r2.College)
	}
	if r2.Nickname1 != "superhero" {
		t.Errorf("nickname is not expected value: '%s'", r2.Nickname1)
	}

	// Make a few more changes, clearing out the nickname. Clearing the
	// nickname is to simulate getting a player with updates from sleeper
	// where the nickname is not provided.
	p.DepthChartOrder = 3
	p.College = "Bama"
	p.Nickname1 = ""
	if err := testDB.SavePlayer(ctx, p); err != nil {
		t.Fatalf("error saving player: %v", err)
	}

	r3, err := testDB.GetPlayer(ctx, p.ID)
	if err != nil {
		t.Fatalf("error getting player: %v", err)
	}
	if r3.DepthChartOrder != 3 {
		t.Errorf("depth chart order is not expected value: %d", r3.DepthChartOrder)
	}
	if r3.College != "Bama" {
		t.Errorf("college is not expected value: '%s'", r3.College)
	}
	// Nickname should not have been deleted
	if r3.Nickname1 != "superhero" {
		t.Errorf("nickname is not the expected value: '%s'", r3.Nickname1)
	}

	if err := testDB.DeletePlayerNickname(ctx, p.ID, "superhero"); err != nil {
		t.Fatalf("error deleting player nickname: %v", err)
	}

	r4, err := testDB.GetPlayer(ctx, p.ID)
	if err != nil {
		t.Fatalf("error getting player: %v", err)
	}
	if r4.Nickname1 != "" {
		t.Errorf("nickname is expected to be empty, but was: '%s'", r4.Nickname1)
	}
	if r4.DepthChartOrder != 3 {
		t.Errorf("depth chart order is not expected value: %d", r4.DepthChartOrder)
	}
	if r4.College != "Bama" {
		t.Errorf("college is not expected value: '%s'", r4.College)
	}
	nicknameChanges := 0
	for i, c := range r4.Changes {
		if c.PropertyName == "Nickname1" {
			nicknameChanges++
			log.Printf("nickname change: %d - old: '%s', new: '%s'", i, c.OldValue, c.NewValue)
		}
	}
	if nicknameChanges != 2 {
		t.Errorf("expected 2 nickname changes but found %d", nicknameChanges)
	}
}

func TestConvertYahooPlayerIDs(t *testing.T) {
	ctx := context.Background()

	// Insert several players with specific IDs so that they are always consistent and we can count on
	// unique results when searching for them. These are also all players found in dleeperdata/players.json file
	players := []model.Player{
		{ID: "1166", YahooID: "", FirstName: "Kirk", LastName: "Cousins", Position: model.POS_QB, Team: model.TEAM_ATL},
		{ID: "1339", YahooID: "26658", FirstName: "Zach", LastName: "Ertz", Position: model.POS_TE, Team: model.TEAM_WAS},
		{ID: "1352", YahooID: "26664", FirstName: "Robert", LastName: "Woods", Position: model.POS_WR, Team: model.TEAM_HOU},
		{ID: "1992", YahooID: "27589", FirstName: "Allen", LastName: "Robinson", Position: model.POS_WR, Team: model.TEAM_DET},
		{ID: "2216", YahooID: "", FirstName: "Mike", LastName: "Evans", Position: model.POS_WR, Team: model.TEAM_TBB},
		{ID: "2359", YahooID: "", FirstName: "Ameer", LastName: "Abdullah", Position: model.POS_RB, Team: model.TEAM_LVR},
		{ID: "3225", YahooID: "29288", FirstName: "Tyler", LastName: "Boyd", Position: model.POS_WR, Team: model.TEAM_TEN},
		{ID: "4080", YahooID: "", FirstName: "Zay", LastName: "Jones", Position: model.POS_WR, Team: model.TEAM_ARI},
		{ID: "4993", YahooID: "31012", FirstName: "Mike", LastName: "Gesicki", Position: model.POS_TE, Team: model.TEAM_CIN},
		{ID: "7601", YahooID: "", FirstName: "Rondale", LastName: "Moore", Position: model.POS_WR, Team: model.TEAM_ATL},
		{ID: "8154", YahooID: "", FirstName: "Brian", LastName: "Robinson", Position: model.POS_RB, Team: model.TEAM_WAS},
		{ID: "10219", YahooID: "", FirstName: "Chris", LastName: "Rodriguez", Position: model.POS_RB, Team: model.TEAM_WAS},
		{ID: "SEA", YahooID: "", FirstName: "Seattle", Position: model.POS_DEF, Team: model.TEAM_SEA},

		// These players have duplicate yahoo ids and are not used in the main test
		{ID: "10225", YahooID: "40073", FirstName: "Jonathan", LastName: "Mingo", Position: model.POS_WR, Team: model.TEAM_CAR},
		{ID: "10444", YahooID: "40073", FirstName: "Cedric", LastName: "Tillman", Position: model.POS_WR, Team: model.TEAM_CLE},
	}

	for _, p := range players {
		if err := testDB.SavePlayer(ctx, &p); err != nil {
			t.Fatalf("error saving player: %v", err)
		}
	}

	input := []model.YahooPlayer{
		{YahooID: "25812", FirstName: "Kirk", LastName: "Cousins", Pos: model.POS_QB},
		{YahooID: "26658", FirstName: "Zach", LastName: "Ertz", Pos: model.POS_TE},
		{YahooID: "26664", FirstName: "Robert", LastName: "Woods", Pos: model.POS_WR},
		{YahooID: "27589", FirstName: "Allen", LastName: "Robinson", Pos: model.POS_WR},
		{YahooID: "27535", FirstName: "Mike", LastName: "Evans", Pos: model.POS_WR},
		{YahooID: "28442", FirstName: "Ameer", LastName: "Abdullah", Pos: model.POS_RB},
		{YahooID: "29288", FirstName: "Tyler", LastName: "Boyd", Pos: model.POS_WR},
		{YahooID: "30150", FirstName: "Zay", LastName: "Jones", Pos: model.POS_WR},
		{YahooID: "31012", FirstName: "Mike", LastName: "Gesicki", Pos: model.POS_TE},
		{YahooID: "33437", FirstName: "Rondale", LastName: "Moore", Pos: model.POS_WR},
		{YahooID: "34054", FirstName: "Brian", LastName: "Robinson", Pos: model.POS_RB},
		{YahooID: "40231", FirstName: "Chris", LastName: "Rodriguez", Pos: model.POS_RB},
		{YahooID: "100026", FirstName: "Seattle", Pos: model.POS_DEF},
	}

	expected := []string{"1166", "1339", "1352", "1992", "2216", "2359", "3225", "4080", "4993", "7601", "8154", "10219", "SEA"}

	results, err := testDB.ConvertYahooPlayerIDs(ctx, input)
	if err != nil {
		t.Fatalf("error converting yahoo player ids: %v", err)
	}
	if !reflect.DeepEqual(expected, results) {
		t.Errorf("expected: %v, got: %v", expected, results)
	}

	// Verify that players where a named match was found had the yahoo id saved back to their record.
	updatedPlayers := []struct {
		playerID string
		yahooID  string
	}{
		{playerID: "1166", yahooID: "25812"},
		{playerID: "2216", yahooID: "27535"},
		{playerID: "2359", yahooID: "28442"},
		{playerID: "4080", yahooID: "30150"},
		{playerID: "7601", yahooID: "33437"},
		{playerID: "8154", yahooID: "34054"},
		{playerID: "10219", yahooID: "40231"},
		{playerID: "SEA", yahooID: "100026"},
	}
	for _, up := range updatedPlayers {
		p, err := testDB.GetPlayer(ctx, up.playerID)
		if err != nil {
			t.Errorf("error finding player with id: %s: %v", up.playerID, err)
			continue
		}
		if p.YahooID != up.yahooID {
			t.Errorf("player yahoo id does not match expected for player: %s (%s %s), wanted: '%s', got: '%s'",
				up.playerID, p.FirstName, p.LastName, up.yahooID, p.YahooID)
		}
	}

	// Search for a player that won't be found
	notFound := []model.YahooPlayer{
		{YahooID: "999999", FirstName: "Captain", LastName: "America", Pos: model.POS_RB},
	}
	if _, err := testDB.ConvertYahooPlayerIDs(ctx, notFound); err == nil {
		t.Errorf("expected an error but there wasn't one when looking up Captain America")
	}

	multipleFound := []model.YahooPlayer{
		{YahooID: "40073", FirstName: "Cedric", LastName: "Tillman", Pos: model.POS_WR},
	}
	if _, err := testDB.ConvertYahooPlayerIDs(ctx, multipleFound); err == nil {
		t.Errorf("expected an error but there wasn't one when looking up a duplicated ID")
	}
}
