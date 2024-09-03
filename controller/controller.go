package controller

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/itbasis/go-clock"
	"github.com/mww/fantasy_manager_v2/db"
	"github.com/mww/fantasy_manager_v2/model"
	"github.com/mww/fantasy_manager_v2/sleeper"
)

// C encapsulates business logic without worrying about any web layers
type C interface {
	GetPlayer(ctx context.Context, id string) (*model.Player, error)
	Search(ctx context.Context, query string) ([]model.Player, error)
	// Updates a player's nickname, or deletes it if the nickname == ""
	// Returns an error if not successful, nil otherwise.
	UpdatePlayerNickname(ctx context.Context, id, nickname string) error
	UpdatePlayers(ctx context.Context) error
	// Look up the scores for a specific player for all leagues and weeks.
	GetPlayerScores(ctx context.Context, playerID string) ([]model.PlayerScore, error)
	RunPeriodicPlayerUpdates(frequency time.Duration, shutdown chan bool, wg *sync.WaitGroup)

	// Add a new rankings for players. This will parse the data from the reader (in CSV format) and
	// create a new rankings data point. Returns the id of the new rankings and an error if there
	// was one.
	AddRanking(ctx context.Context, r io.Reader, date time.Time) (int32, error)
	GetRanking(ctx context.Context, id int32) (*model.Ranking, error)
	DeleteRanking(ctx context.Context, id int32) error
	ListRankings(ctx context.Context) ([]model.Ranking, error)

	GetLeaguesFromPlatform(ctx context.Context, username, platform, year string) ([]model.League, error)
	AddLeague(ctx context.Context, platform, externalID, name, year string) (*model.League, error)
	AddLeagueManagers(ctx context.Context, leagueID int32) (*model.League, error) // Will also update the list
	GetLeague(ctx context.Context, id int32) (*model.League, error)
	ListLeagues(ctx context.Context) ([]model.League, error)
	SyncResultsFromPlatform(ctx context.Context, leagueID int32, week int) error
	GetLeagueResults(ctx context.Context, leagueID int32, week int) ([]model.Matchup, error)
}

type controller struct {
	clock   clock.Clock
	sleeper sleeper.Client
	db      db.DB
}

func New(clock clock.Clock, sleeper sleeper.Client, db db.DB) (C, error) {
	c := &controller{
		clock:   clock,
		sleeper: sleeper,
		db:      db,
	}
	return c, nil
}

// When we need to make calls that are specific to a platform, grab a platform
// adapter and it will do it. This is internal to the controller package.
type platformAdpater interface {
	getLeagues(user, year string) ([]model.League, error)
	getManagers(l *model.League) ([]model.LeagueManager, error)
	sortManagers(m []model.LeagueManager)
	getMatchupResults(l *model.League, week int) ([]model.Matchup, []model.PlayerScore, error)
}

func getPlatformAdapter(platform string, c *controller) platformAdpater {
	switch platform {
	case model.PlatformSleeper:
		return &sleeperAdapter{c}
	default:
		return &nilPlatformAdapter{err: fmt.Errorf("%s is not a supported platform", platform)}
	}
}

// nilPlatformAdapter exists so that we can always return an adapter and simply the usage.
// It eliminates the need for an extra error check.
type nilPlatformAdapter struct {
	err error
}

func (a *nilPlatformAdapter) getLeagues(user, year string) ([]model.League, error) {
	return nil, a.err
}

func (a *nilPlatformAdapter) getManagers(l *model.League) ([]model.LeagueManager, error) {
	return nil, a.err
}

func (a *nilPlatformAdapter) sortManagers(m []model.LeagueManager) {
}

func (a *nilPlatformAdapter) getMatchupResults(l *model.League, week int) ([]model.Matchup, []model.PlayerScore, error) {
	return nil, nil, a.err
}
