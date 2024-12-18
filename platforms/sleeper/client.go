package sleeper

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/mww/fantasy_manager_v2/model"
)

const SleeperURL = "https://api.sleeper.app"

type Client interface {
	LoadPlayers() ([]model.Player, error)

	// Take the username and return the sleeper user id or an error.
	GetUserID(username string) (string, error)

	// Get all of the leagues for the user and year.
	GetLeaguesForUser(userID, year string) ([]model.League, error)

	GetLeagueName(leagueID string) (string, error)

	// Get all of the league managers for a specific league.
	GetLeagueManagers(leagueID string) ([]model.LeagueManager, error)

	// Sort the managers in a stable and logical order.
	SortManagers(m []model.LeagueManager)

	// Get the matchup for a specific week for a league
	// Also returns the individual scores for all the players.
	GetMatchupResults(leagueID string, week int) ([]model.Matchup, []model.PlayerScore, error)

	// Load the rosters for all users.
	GetRosters(leagueID string) ([]model.Roster, error)

	// Get a list of all the positions a user needs to start in the league.
	// This is used to select a starting lineup in the power rankings.
	GetStarters(leagueID string) ([]model.RosterSpot, error)

	GetLeagueStandings(leagueID string) ([]model.LeagueStanding, error)
}

type client struct {
	url        string
	httpClient *http.Client
}

func New() (Client, error) {
	c := &client{
		url: SleeperURL,
		httpClient: &http.Client{
			Timeout: 1 * time.Minute,
		},
	}
	return c, nil
}

func NewForTest(url string) Client {
	return &client{
		url:        url,
		httpClient: http.DefaultClient,
	}
}

func (c *client) LoadPlayers() ([]model.Player, error) {
	var parsed map[string]sleeperPlayer
	if err := c.sleeperRequest(&parsed, "/v1/players/nfl"); err != nil {
		return nil, err
	}

	// Convert the players into model.Players
	result := make([]model.Player, 0, len(parsed))
	for _, p := range parsed {
		pos := model.ParsePosition(p.Position)
		if pos == model.POS_UNKNOWN || (p.FirstName == "Player" && p.LastName == "Invalid") {
			continue
		}
		if p.ID == "" {
			log.Printf("player without an ID set: %s %s", p.FirstName, p.LastName)
			continue
		}
		result = append(result, *p.toPlayer())
	}

	return result, nil
}

func (c *client) GetUserID(username string) (string, error) {
	var resp struct {
		UserID string `json:"user_id"`
	}
	if err := c.sleeperRequest(&resp, "/v1/user/%s", username); err != nil {
		return "", err
	}

	if resp.UserID == "" {
		return "", errors.New("user not found")
	}

	return resp.UserID, nil
}

func (c *client) GetLeaguesForUser(userID, year string) ([]model.League, error) {
	var resp []struct {
		LeagueID string `json:"league_id"`
		Name     string `json:"name"`
	}
	if err := c.sleeperRequest(&resp, "/v1/user/%s/leagues/nfl/%s", userID, year); err != nil {
		return nil, err
	}

	if len(resp) == 0 {
		return nil, errors.New("no leagues found")
	}

	res := make([]model.League, len(resp))
	for i, r := range resp {
		res[i].ExternalID = r.LeagueID
		res[i].Name = r.Name
		res[i].Year = year
		res[i].Archived = false
		res[i].Platform = model.PlatformSleeper
	}

	return res, nil
}

func (c *client) GetLeagueName(leagueID string) (string, error) {
	var league struct {
		Name string `json:"name"`
	}
	if err := c.sleeperRequest(&league, "/v1/league/%s", leagueID); err != nil {
		return "", err
	}
	if league.Name == "" {
		return "", errors.New("league name not found")
	}
	return league.Name, nil
}

func (c *client) GetLeagueManagers(leagueID string) ([]model.LeagueManager, error) {
	var rosters []struct {
		OwnerID  string `json:"owner_id"`
		RosterID int    `json:"roster_id"`
	}
	if err := c.sleeperRequest(&rosters, "/v1/league/%s/rosters", leagueID); err != nil {
		return nil, err
	}
	if len(rosters) == 0 {
		return nil, errors.New("no managers found")
	}

	managerMap := make(map[string]*model.LeagueManager)
	for _, r := range rosters {
		managerMap[r.OwnerID] = &model.LeagueManager{
			ExternalID: r.OwnerID,
			JoinKey:    fmt.Sprint(r.RosterID),
		}
	}

	type metadata struct {
		TeamName string `json:"team_name"`
	}
	var users []struct {
		DisplayName string    `json:"display_name"`
		UserID      string    `json:"user_id"`
		Metadata    *metadata `json:"metadata"`
	}
	if err := c.sleeperRequest(&users, "/v1/league/%s/users", leagueID); err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, errors.New("no managers found")
	}

	for _, u := range users {
		if m, found := managerMap[u.UserID]; found {
			m.ManagerName = u.DisplayName
			if u.Metadata != nil && u.Metadata.TeamName != "" {
				m.TeamName = u.Metadata.TeamName
			}
		}
	}

	resp := make([]model.LeagueManager, 0, len(managerMap))
	for _, v := range managerMap {
		resp = append(resp, *v)
	}
	c.SortManagers(resp)
	return resp, nil
}

func (c *client) SortManagers(m []model.LeagueManager) {
	// Sort by the JoinKey value
	slices.SortFunc(m, func(a, b model.LeagueManager) int {
		ai, e1 := strconv.Atoi(a.JoinKey)
		bi, e2 := strconv.Atoi(b.JoinKey)
		if err := errors.Join(e1, e2); err != nil {
			return 0
		}
		return ai - bi
	})
}

func (c *client) GetMatchupResults(leagueID string, week int) ([]model.Matchup, []model.PlayerScore, error) {
	var res []struct {
		Points       float64            `json:"points"`
		RosterID     int                `json:"roster_id"`
		MatchupID    int32              `json:"matchup_id"`
		PlayerPoints map[string]float64 `json:"players_points"`
	}
	if err := c.sleeperRequest(&res, "/v1/league/%s/matchups/%d", leagueID, week); err != nil {
		return nil, nil, err
	}

	playerScores := make([]model.PlayerScore, 0, 128)
	// map key is matchup_id which allows us to join the matches
	matchMap := make(map[int32]*model.Matchup)
	for _, r := range res {
		tr := &model.TeamResult{
			JoinKey: fmt.Sprint(r.RosterID),
			Score:   int32(r.Points * 1000),
		}
		m := matchMap[r.MatchupID]
		if m == nil {
			// This is the first team we've found for the matchup
			matchMap[r.MatchupID] = &model.Matchup{
				TeamA:     tr,
				MatchupID: r.MatchupID,
				Week:      week,
			}
		} else {
			// The first team in the matchup has already been added
			m.TeamB = tr
		}

		for id, score := range r.PlayerPoints {
			ps := model.PlayerScore{
				PlayerID: id,
				Score:    int32(score * 1000),
			}
			playerScores = append(playerScores, ps)
		}
	}

	matches := make([]model.Matchup, 0, len(matchMap))
	for _, m := range matchMap {
		if m.TeamA == nil || m.TeamB == nil {
			return nil, nil, errors.New("at least one matchup is not complete with 2 teams")
		}
		matches = append(matches, *m)
	}
	slices.SortFunc(matches, func(a, b model.Matchup) int {
		return int(a.MatchupID - b.MatchupID)
	})
	return matches, playerScores, nil
}

func (c *client) GetRosters(leagueID string) ([]model.Roster, error) {
	var rosters []struct {
		OwnerID string   `json:"owner_id"`
		Players []string `json:"players"`
	}
	if err := c.sleeperRequest(&rosters, "/v1/league/%s/rosters", leagueID); err != nil {
		return nil, err
	}

	results := make([]model.Roster, 0, len(rosters))
	for _, r := range rosters {
		roster := model.Roster{
			TeamID:    r.OwnerID,
			PlayerIDs: r.Players,
		}
		results = append(results, roster)
	}
	return results, nil
}

func (c *client) GetStarters(leagueID string) ([]model.RosterSpot, error) {
	var league struct {
		RosterPositions []string `json:"roster_positions"`
	}
	if err := c.sleeperRequest(&league, "/v1/league/%s", leagueID); err != nil {
		return nil, err
	}

	response := make([]model.RosterSpot, 0, 10)
	for _, p := range league.RosterPositions {
		if p == "BN" {
			break
		}
		response = append(response, model.GetRosterSpot(p))
	}
	return response, nil
}

func (c *client) GetLeagueStandings(leagueID string) ([]model.LeagueStanding, error) {
	type settings struct {
		FPts        int `json:"fpts"`
		FPtsDecimal int `json:"fpts_decimal"`
		Wins        int `json:"wins"`
		Losses      int `json:"losses"`
		Ties        int `json:"ties"`
	}
	var data []struct {
		OwnerID  string   `json:"owner_id"`
		Settings settings `json:"settings"`
	}
	if err := c.sleeperRequest(&data, "/v1/league/%s/rosters", leagueID); err != nil {
		return nil, err
	}

	results := make([]model.LeagueStanding, 0, len(data))
	for _, t := range data {
		s := model.LeagueStanding{
			TeamID: t.OwnerID,
			Wins:   t.Settings.Wins,
			Losses: t.Settings.Losses,
			Draws:  t.Settings.Ties,
			Scored: fmt.Sprintf("%d.%02d", t.Settings.FPts, t.Settings.FPtsDecimal),
		}
		results = append(results, s)
	}

	// Sort in order:
	// - Highest number of wins
	// - Smaller number of losses
	// - Fantasy points scored
	slices.SortFunc(results, func(a, b model.LeagueStanding) int {
		if a.Wins == b.Wins {
			if a.Losses == b.Losses {
				fa, e1 := strconv.ParseFloat(a.Scored, 64)
				fb, e2 := strconv.ParseFloat(b.Scored, 64)
				if err := errors.Join(e1, e2); err != nil {
					log.Printf("error parsing points scored: %v", err)
					return 0
				}
				if fa > fb {
					return -1
				} else if fb > fa {
					return 1
				}
				return 0
			}
			return a.Losses - b.Losses
		}
		return b.Wins - a.Wins
	})
	for i := range results {
		results[i].Rank = i + 1
	}
	return results, nil
}

// Sends the request to sleeper and uses a JSON parser to read the result into res.
// Returns an error if any or if the status code of the result is not 200.
func (c *client) sleeperRequest(res any, path string, args ...any) error {
	p := fmt.Sprintf(path, args...)
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", c.url, p), nil)
	if err != nil {
		return fmt.Errorf("error creating sleeper http request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending sleeper http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code from sleeper: %d", resp.StatusCode)
	}

	err = json.NewDecoder(resp.Body).Decode(res)
	if err != nil {
		return fmt.Errorf("error parsing response from sleeper: %w", err)
	}

	return nil
}
