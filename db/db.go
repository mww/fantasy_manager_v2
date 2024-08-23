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

	SavePlayerScores(ctx context.Context, leagueID int32, week int, scores []model.PlayerScore) error
	// Look up the scores for a specific player regardless of league or week.
	GetPlayerScores(ctx context.Context, playerID string) ([]model.PlayerScore, error)

	// Lists the 20 most recent rankings in the system. The most recent ranking is returned first.
	// Only the ranking metadata, the ID and date, are returned. The actual ranking data is returned
	// with GetRanking().
	ListRankings(ctx context.Context) ([]model.Ranking, error)
	GetRanking(ctx context.Context, id int32) (*model.Ranking, error)
	AddRanking(ctx context.Context, date time.Time, rankings map[string]int32) (*model.Ranking, error)
	DeleteRanking(ctx context.Context, id int32) error

	ListLeagues(ctx context.Context) ([]model.League, error)
	GetLeague(ctx context.Context, id int32) (*model.League, error)
	GetLeagueManagers(ctx context.Context, leagueID int32) ([]model.LeagueManager, error)
	SaveLeagueManager(ctx context.Context, leagueID int32, managers *model.LeagueManager) error
	AddLeague(ctx context.Context, league *model.League) error
	ArchiveLeague(ctx context.Context, id int32) error

	SaveResults(ctx context.Context, leagueID int32, matchups []model.Matchup) error
	GetResults(ctx context.Context, leagueID int32, week int) ([]model.Matchup, error)
}
