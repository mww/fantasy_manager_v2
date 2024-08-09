package db

import (
	"context"
	"time"

	"github.com/mww/fantasy_manager_v2/model"
)

type DB interface {
	GetPlayer(ctx context.Context, id string) (*model.Player, error)
	SavePlayer(ctx context.Context, p *model.Player) error
	DeleteNickname(ctx context.Context, id string, oldNickname string) error
	Search(ctx context.Context, query string, pos model.Position, team *model.NFLTeam) ([]model.Player, error)

	// Lists the 20 most recent rankings in the system. The most recent ranking is returned first.
	// Only the ranking metadata, the ID and date, are returned. The actual ranking data is returned
	// with GetRanking().
	ListRankings(ctx context.Context) ([]model.Ranking, error)
	GetRanking(ctx context.Context, id int32) (*model.Ranking, error)
	AddRanking(ctx context.Context, date time.Time, rankings map[string]int32) (*model.Ranking, error)
	DeleteRanking(ctx context.Context, id int32) error
}
