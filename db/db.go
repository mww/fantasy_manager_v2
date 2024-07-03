package db

import (
	"context"

	"github.com/mww/fantasy_manager_v2/model"
)

type DB interface {
	GetPlayer(ctx context.Context, id string) (*model.Player, error)
	SavePlayer(ctx context.Context, p *model.Player) error
	Search(ctx context.Context, query string, pos model.Position, team *model.NFLTeam) ([]model.Player, error)
}
