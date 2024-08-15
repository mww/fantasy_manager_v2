package sleeper

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
