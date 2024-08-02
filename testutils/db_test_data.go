package testutils

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/itbasis/go-clock"
	"github.com/mww/fantasy_manager_v2/containers"
	"github.com/mww/fantasy_manager_v2/db"
	"github.com/mww/fantasy_manager_v2/model"
)

var (
	IDLockett   = "2374"
	IDHurts     = "6904"
	IDLamb      = "6786"
	IDHockenson = "5844"
	IDHall      = "8155"
	IDJefferson = "6794"
	IDMcCaffrey = "4034"
	IDChase     = "7564"
	IDChubb     = "4988"
	IDKelce     = "1466"
	IDHill      = "3321"
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	players, err := GetPlayersForTest()
	if err != nil {
		return err
	}
	for _, p := range players {
		err := db.SavePlayer(ctx, &p)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetPlayersForTest() ([]model.Player, error) {
	players := []struct {
		id string
		f  string // first name
		l  string // last name
		p  string // position
		t  string // team
	}{
		{id: IDLockett, f: "Tyler", l: "Lockett", p: "WR", t: "SEA"},
		{id: IDHurts, f: "Jalen", l: "Hurts", p: "QB", t: "PHI"},
		{id: IDLamb, f: "CeeDee", l: "Lamb", p: "WR", t: "DAL"},
		{id: IDHockenson, f: "T.J.", l: "Hockenson", p: "TE", t: "MIN"},
		{id: IDHall, f: "Breece", l: "Hall", p: "RB", t: "NYJ"},
		{id: IDJefferson, f: "Justin", l: "Jefferson", p: "WR", t: "MIN"},
		{id: IDMcCaffrey, f: "Christian", l: "McCaffrey", p: "RB", t: "SFO"},
		{id: IDChase, f: "Ja'Marr", l: "Chase", p: "WR", t: "CIN"},
		{id: IDChubb, f: "Nick", l: "Chubb", p: "RB", t: "CLE"},
		{id: IDKelce, f: "Travis", l: "Kelce", p: "TE", t: "KCC"},
		{id: IDHill, f: "Tyreek", l: "Hill", p: "WR", t: "MIA"},
	}

	result := make([]model.Player, len(players))
	for i, p := range players {
		pos := model.ParsePosition(p.p)
		if pos == model.POS_UNKNOWN {
			return nil, fmt.Errorf("unknown position for player %s", p.id)
		}
		team := model.ParseTeam(p.t)
		if team == model.TEAM_FA {
			return nil, fmt.Errorf("unknown team for player %s", p.id)
		}

		result[i].ID = p.id
		result[i].FirstName = p.f
		result[i].LastName = p.l
		result[i].Position = pos
		result[i].Team = team
	}

	return result, nil
}
