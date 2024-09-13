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
	"github.com/mww/fantasy_manager_v2/platforms/sleeper"
	"github.com/mww/fantasy_manager_v2/platforms/yahoo"
	"golang.org/x/oauth2"
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
	GetPlayerScores(ctx context.Context, playerID string) ([]model.SeasonScores, error)
	RunPeriodicPlayerUpdates(frequency time.Duration, shutdown chan bool, wg *sync.WaitGroup)

	// Add a new rankings for players. This will parse the data from the reader (in CSV format) and
	// create a new rankings data point. Returns the id of the new rankings and an error if there
	// was one.
	AddRanking(ctx context.Context, r io.Reader, date time.Time) (int32, error)
	GetRanking(ctx context.Context, id int32) (*model.Ranking, error)
	DeleteRanking(ctx context.Context, id int32) error
	ListRankings(ctx context.Context) ([]model.Ranking, error)

	GetLeaguesFromPlatform(ctx context.Context, username, platform, year string) ([]model.League, error)
	AddLeague(ctx context.Context, platform, externalID, year, stateToken string) (*model.League, error)
	AddLeagueManagers(ctx context.Context, leagueID int32) (*model.League, error) // Will also update the list
	GetLeague(ctx context.Context, id int32) (*model.League, error)
	ListLeagues(ctx context.Context) ([]model.League, error)
	ArchiveLeague(ctx context.Context, id int32) error
	SyncResultsFromPlatform(ctx context.Context, leagueID int32, week int) error
	// Return a slice of weeks for which there are results for the league
	ListLeagueResultWeeks(ctx context.Context, leagueID int32) ([]int, error)
	GetLeagueResults(ctx context.Context, leagueID int32, week int) ([]model.Matchup, error)

	ListPowerRankings(ctx context.Context, leagueID int32) ([]model.PowerRanking, error)
	GetPowerRanking(ctx context.Context, leagueID, powerRankingID int32) (*model.PowerRanking, error)
	// Calculates the power ranking and returns the id of the saved rankings
	CalculatePowerRanking(ctx context.Context, leagueID, rankingID int32, week int) (int32, error)

	// These methods are all for OAuth linking. Start creates a state token and
	// saves it for 5 minutes, returning the auth code URL.
	// Exchange makes sure that the state token exists and is valid before exchanging
	// the code for an oauth token.
	// and retrieved by the same random state value until the new league can be
	// created and the token can be properly saved in the DB.
	// In all cases the state token expires after the 5 minutes and then stash
	// and retrieve stop working for it.
	// Once a league has been created the Token can be fully commited to the DB
	// by calling OAuthSave(). After that the token can only be retrieved with
	// GetToken().
	OAuthStart(platform string) (string, error)
	OAuthExchange(ctx context.Context, state, code string) error
	OAuthRetrieve(state string) (*oauth2.Token, error)
	OAuthSave(ctx context.Context, state string, leagueID int32) error

	GetToken(ctx context.Context, leagueID int32) (*oauth2.Token, error)
}

type controller struct {
	clock       clock.Clock
	db          db.DB
	sleeper     sleeper.Client
	yahoo       *yahoo.Client
	yahooConfig *oauth2.Config
	oauthStates map[string]*oauthState
}

type oauthState struct {
	platform string
	expiry   time.Time
	token    *oauth2.Token
}

func New(clock clock.Clock, db db.DB, sleeper sleeper.Client, yahoo *yahoo.Client, yahooConfig *oauth2.Config) (C, error) {
	c := &controller{
		clock:       clock,
		db:          db,
		sleeper:     sleeper,
		yahoo:       yahoo,
		yahooConfig: yahooConfig,
		oauthStates: make(map[string]*oauthState),
	}
	return c, nil
}

// When we need to make calls that are specific to a platform, grab a platform
// adapter and it will do it. This is internal to the controller package.
type platformAdpater interface {
	getLeagues(user, year string) ([]model.League, error)
	getLeagueName(ctx context.Context, leagueID, stateToken string) (string, error)
	getManagers(ctx context.Context, l *model.League) ([]model.LeagueManager, error)
	sortManagers(m []model.LeagueManager)
	getMatchupResults(ctx context.Context, l *model.League, week int) ([]model.Matchup, []model.PlayerScore, error)
	getRosters(ctx context.Context, l *model.League) ([]model.Roster, error)
	// Get all the starting roster spots. This is used in the power rankings calculations.
	getStarters(ctx context.Context, l *model.League) ([]model.RosterSpot, error)
}

func getPlatformAdapter(platform string, c *controller) platformAdpater {
	switch platform {
	case model.PlatformSleeper:
		return &sleeperAdapter{c}
	case model.PlatformYahoo:
		return &yahooAdapter{c}
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

func (a *nilPlatformAdapter) getLeagueName(ctx context.Context, leagueID, stateToken string) (string, error) {
	return "", a.err
}

func (a *nilPlatformAdapter) getManagers(ctx context.Context, l *model.League) ([]model.LeagueManager, error) {
	return nil, a.err
}

func (a *nilPlatformAdapter) sortManagers(m []model.LeagueManager) {
}

func (a *nilPlatformAdapter) getMatchupResults(ctx context.Context, l *model.League, week int) ([]model.Matchup, []model.PlayerScore, error) {
	return nil, nil, a.err
}

func (a *nilPlatformAdapter) getRosters(ctx context.Context, l *model.League) ([]model.Roster, error) {
	return nil, a.err
}

func (a *nilPlatformAdapter) getStarters(ctx context.Context, l *model.League) ([]model.RosterSpot, error) {
	return nil, a.err
}
