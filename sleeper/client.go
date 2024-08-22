package sleeper

import (
	"encoding/json"
	"errors"
	"fmt"
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

	// Get all of the league managers for a specific league.
	GetLeagueManagers(leagueID string) ([]model.LeagueManager, error)

	// Sort the managers in a stable and logical order.
	SortManagers(m []model.LeagueManager)
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

func (c *client) GetLeagueManagers(leagueID string) ([]model.LeagueManager, error) {
	// TODO - call rosters first, then get the owners, that way we can more easily ignore co-owners
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
