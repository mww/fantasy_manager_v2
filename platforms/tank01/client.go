package tank01

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/mww/fantasy_manager_v2/model"
)

const (
	tank01URL = "https://tank01-nfl-live-in-game-real-time-statistics-nfl.p.rapidapi.com"

	headerRapidApiHost = "x-rapidapi-host"
	headerRapidApiKey  = "x-rapidapi-key"

	rapidApiHost = "tank01-nfl-live-in-game-real-time-statistics-nfl.p.rapidapi.com"
)

type Client interface {
	LoadPlayers() ([]model.Player, error)
}

type client struct {
	url        string
	key        string
	httpClient *http.Client
}

func New(key string) (Client, error) {
	c := &client{
		url: tank01URL,
		key: key,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	return c, nil
}

func NewForTest(url string) Client {
	return &client{
		url:        url,
		key:        "not-important",
		httpClient: http.DefaultClient,
	}
}

func (c *client) LoadPlayers() ([]model.Player, error) {
	return nil, errors.New("not yet")
}

func (c *client) tank01Request(res any, path string, args ...any) error {
	p := fmt.Sprintf(path, args...)
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", c.url, p), nil)
	if err != nil {
		return fmt.Errorf("error creating tank01 http request: %w", err)
	}
	req.Header.Add(headerRapidApiHost, rapidApiHost)
	req.Header.Add(headerRapidApiKey, c.key)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending tank01 http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code from tank01: %d", resp.StatusCode)
	}

	err = json.NewDecoder(resp.Body).Decode(res)
	if err != nil {
		return fmt.Errorf("error parsing response from tank01: %w", err)
	}

	return nil
}
