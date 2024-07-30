package testutils

import (
	"context"
	"log"
	"time"

	"github.com/itbasis/go-clock"
	"github.com/mww/fantasy_manager_v2/containers"
	"github.com/mww/fantasy_manager_v2/db"
	"github.com/mww/fantasy_manager_v2/model"
)

var (
	TylerLockett = &model.Player{
		ID:        "2374",
		FirstName: "Tyler",
		LastName:  "Lockett",
		Position:  model.POS_WR,
		Team:      model.TEAM_SEA,
	}
	JalenHurts = &model.Player{
		ID:        "6904",
		FirstName: "Jalen",
		LastName:  "Hurts",
		Position:  model.POS_QB,
		Team:      model.TEAM_PHI,
	}
	CeeDeeLamb = &model.Player{
		ID:        "6786",
		FirstName: "CeeDee",
		LastName:  "Lamb",
		Position:  model.POS_WR,
		Team:      model.TEAM_DAL,
	}
	TJHockenson = &model.Player{
		ID:        "5844",
		FirstName: "T.J.",
		LastName:  "Hockenson",
		Position:  model.POS_TE,
		Team:      model.TEAM_MIN,
	}
	BreeceHall = &model.Player{
		ID:        "8155",
		FirstName: "Breece",
		LastName:  "Hall",
		Position:  model.POS_RB,
		Team:      model.TEAM_NYJ,
	}
)

type TestDB struct {
	container *containers.DBContainer
	DB        db.DB
	Clock     clock.Clock
}

func NewTestDB() *TestDB {
	container := containers.NewDBContainer()
	clock := clock.New()

	db, err := db.New(context.Background(), container.ConnectionString(), clock)
	if err != nil {
		log.Fatalf("error connecting to db in test container: %v", err)
	}

	if err := InsertTestPlayers(db); err != nil {
		log.Fatalf("error populating db in test container: %v", err)
	}

	return &TestDB{
		container: container,
		DB:        db,
		Clock:     clock,
	}
}

func (db *TestDB) Shutdown() {
	db.container.Shutdown()
}

func InsertTestPlayers(db db.DB) error {
	players := []*model.Player{
		TylerLockett,
		JalenHurts,
		CeeDeeLamb,
		TJHockenson,
		BreeceHall,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, p := range players {
		err := db.SavePlayer(ctx, p)
		if err != nil {
			return err
		}
	}

	return nil
}
