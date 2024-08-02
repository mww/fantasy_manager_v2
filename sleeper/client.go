package sleeper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mww/fantasy_manager_v2/model"
)

const SleeperURL = "https://api.sleeper.app"

type Client interface {
	LoadPlayers() ([]model.Player, error)
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

func (c *client) LoadPlayers() ([]model.Player, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/players/nfl", c.url), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var parsed map[string]sleeperPlayer
	err = json.NewDecoder(resp.Body).Decode(&parsed)
	if err != nil {
		return nil, fmt.Errorf("error parsing response from sleeper: %w", err)
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
