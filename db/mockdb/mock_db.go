package mockdb

import (
	"context"

	"github.com/mww/fantasy_manager_v2/model"
	"github.com/stretchr/testify/mock"
)

type DB struct {
	mock.Mock
}

func (db *DB) GetPlayer(ctx context.Context, id string) (*model.Player, error) {
	args := db.Called(ctx, id)

	var p *model.Player
	if args.Get(0) != nil {
		p = args.Get(0).(*model.Player)
	}

	return p, args.Error(1)
}

func (db *DB) SavePlayer(ctx context.Context, p *model.Player) error {
	args := db.Called(ctx, p)
	return args.Error(0)
}

func (db *DB) DeleteNickname(ctx context.Context, id, oldNickname string) error {
	args := db.Called(ctx, id, oldNickname)
	return args.Error(0)
}

func (db *DB) Search(ctx context.Context, query string, pos model.Position, team *model.NFLTeam) ([]model.Player, error) {
	args := db.Called(ctx, query, pos, team)

	var r []model.Player
	if args.Get(0) != nil {
		r = args.Get(0).([]model.Player)
	}
	return r, args.Error(1)
}
