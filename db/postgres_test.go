package db

import (
	"context"
	"errors"
	"log"
	"path/filepath"
	"testing"
	"time"

	"github.com/itbasis/go-clock"
	"github.com/mww/fantasy_manager_v2/model"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	image      = "postgres:16.3-alpine"
	dbName     = "fantasy_manager"
	dbUser     = "ffuser"
	dbPassword = "secret"
)

type dbContainer struct {
	container *postgres.PostgresContainer
	db        DB
}

func NewContainer() *dbContainer {
	ctx := context.Background()

	container, err := postgres.RunContainer(ctx,
		testcontainers.WithImage(image),
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		postgres.WithInitScripts(filepath.Join("..", "schema", "schema.sql")),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		log.Fatalf("error starting container: %v", err)
	}

	// explicitly set sslmode=disable because the container is not configured to use TLS
	connStr, err := container.ConnectionString(context.Background(), "sslmode=disable")
	if err != nil {
		log.Fatalf("error getting connection string: %v", err)
	}

	clock := clock.New()
	db, err := New(ctx, connStr, clock)
	if err != nil {
		log.Fatalf("error creating db instance: %s", err)
	}

	return &dbContainer{
		container: container,
		db:        db,
	}
}

func (c *dbContainer) Shutdown() {
	err := c.container.Terminate(context.Background())
	if err != nil {
		log.Fatalf("error terminating container: %v", err)
	}
}

func TestDB_saveAndLoad(t *testing.T) {
	ctx := context.Background()

	c := NewContainer()
	defer c.Shutdown()

	p := getPlayer()

	err := c.db.SavePlayer(ctx, p)
	assertFatalf(t, err == nil, "error saving player: %v", err)

	res, err := c.db.GetPlayer(ctx, "2374")
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
	err = c.db.SavePlayer(ctx, p)
	assertFatalf(t, err == nil, "error saving player after update: %v", err)

	res2, err := c.db.GetPlayer(ctx, "2374")
	assertFatalf(t, err == nil, "error saving updated player: %v", err)

	assertEquals(t, "Weight", p.Weight, res2.Weight)
	assertEquals(t, "Changes", 1, len(p.Changes))
	// Now updated should not be zero
	if res2.Updated.IsZero() {
		t.Errorf("expected res2 updated time to not be zero")
	}

	// Lookup a player that doesn't exist
	res3, err := c.db.GetPlayer(ctx, "1111")
	assertFatalf(t, err != nil, "should have had an error searching for player")
	assertEquals(t, "error type", true, errors.Is(err, ErrPlayerNotFound))
	if res3 != nil {
		t.Errorf("expected res3 to be nil, but was %v", res3)
	}
}

func TestDB_search(t *testing.T) {
	ctx := context.Background()

	c := NewContainer()
	defer c.Shutdown()

	p := getPlayer()

	err := c.db.SavePlayer(ctx, p)
	assertFatalf(t, err == nil, "error saving player: %v", err)

	players, err := c.db.Search(ctx, "Tyler")
	assertFatalf(t, err == nil, "error searching for player: %v", err)
	assertEquals(t, "num players found", 1, len(players))

	players, err = c.db.Search(ctx, "Frank")
	assertFatalf(t, err == nil, "error searching for players: %v", err)
	assertEquals(t, "num players found 2", 0, len(players))
}

func getPlayer() *model.Player {
	return &model.Player{
		ID:              "2374",
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
		t.Errorf("%s - expected: %s, got: %s", field, expected, actual)
	}
}
