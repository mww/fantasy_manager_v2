package yahoo

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"

	"github.com/mww/fantasy_manager_v2/model"
	"github.com/mww/fantasy_manager_v2/platforms/yahoo/internal"
)

const YahooURL = "https://fantasysports.yahooapis.com"

type Client struct {
	url string
}

func New() (*Client, error) {
	return &Client{url: YahooURL}, nil
}

func NewForTest(url string) *Client {
	return &Client{url: url}
}

func (c *Client) GetStarters(httpClient *http.Client, leagueID string) ([]model.RosterSpot, error) {
	content, err := c.yahooRequest(httpClient, "/fantasy/v2/league/nfl.l.%s/settings", leagueID)
	if err != nil {
		return nil, err
	}

	if content == nil ||
		content.League == nil ||
		content.League.Settings == nil ||
		content.League.Settings.RosterPositions == nil ||
		content.League.Settings.RosterPositions.Positions == nil {
		return nil, errors.New("settings has no roster positions")
	}
	resp := make([]model.RosterSpot, 0, 9)
	for _, p := range content.League.Settings.RosterPositions.Positions {
		pos := p.Position
		if p.Position == "BN" {
			continue
		}
		if p.Position == "W/R/T" {
			pos = "FLEX"
		}
		for range p.Count {
			resp = append(resp, model.GetRosterSpot(pos))
		}
	}

	if len(resp) == 0 {
		return nil, errors.New("no roster positions found")
	}
	return resp, nil
}

func (c *Client) GetManagers(httpClient *http.Client, leagueID string) ([]model.LeagueManager, error) {
	content, err := c.yahooRequest(httpClient, "/fantasy/v2/league/nfl.l.%s/standings", leagueID)
	if err != nil {
		return nil, err
	}

	if content == nil ||
		content.League == nil ||
		content.League.Standings == nil ||
		content.League.Standings.Teams == nil ||
		content.League.Standings.Teams.Teams == nil {
		return nil, errors.New("league has no teams")
	}

	resp := make([]model.LeagueManager, 0, 12)
	for _, t := range content.League.Standings.Teams.Teams {
		var m model.LeagueManager

		m.ExternalID = t.Key
		m.TeamName = t.Name
		if t.Managers != nil && t.Managers.Managers != nil {
			m.ManagerName = t.Managers.Managers[0].Nickname
		}
		resp = append(resp, m)
	}

	return resp, nil
}

func (c *Client) GetLeagueName(httpClient *http.Client, leagueID string) (string, error) {
	content, err := c.yahooRequest(httpClient, "/fantasy/v2/league/nfl.l.%s", leagueID)
	if err != nil {
		return "", err
	}

	if content == nil || content.League == nil || content.League.Name == "" {
		return "", errors.New("league name not found")
	}

	return content.League.Name, nil
}

func (c *Client) GetScoreboard(httpClient *http.Client, leagueID string, week int) ([]model.Matchup, error) {
	content, err := c.yahooRequest(httpClient, "/fantasy/v2/league/nfl.l.%s/scoreboard;week=%d", leagueID, week)
	if err != nil {
		return nil, err
	}

	if content == nil ||
		content.League == nil ||
		content.League.Scoreboard == nil ||
		content.League.Scoreboard.Matchups == nil ||
		content.League.Scoreboard.Matchups.Matchups == nil {
		return nil, errors.New("league scoreboard not found")
	}

	results := make([]model.Matchup, 0, 6)
	for _, m := range content.League.Scoreboard.Matchups.Matchups {
		if err := validateTeams(m.Teams); err != nil {
			return nil, err
		}

		matchup := model.Matchup{
			Week: week,
			TeamA: &model.TeamResult{
				TeamID: m.Teams.Teams[0].Key,
				Score:  int32(m.Teams.Teams[0].TeamPoints.Total * 1000),
			},
			TeamB: &model.TeamResult{
				TeamID: m.Teams.Teams[1].Key,
				Score:  int32(m.Teams.Teams[1].TeamPoints.Total * 1000),
			},
		}

		results = append(results, matchup)
	}

	return results, nil
}

func validateTeams(teams *internal.Teams) error {
	if teams == nil || len(teams.Teams) != 2 {
		return errors.New("invalid teams in result")
	}
	for _, t := range teams.Teams {
		if t.Key == "" || t.TeamPoints == nil {
			return errors.New("invalid team in results")
		}
	}
	return nil
}

func (c *Client) GetRoster(httpClient *http.Client, teamID string) ([]model.YahooPlayer, error) {
	content, err := c.yahooRequest(httpClient, "/fantasy/v2/team/%s/roster", teamID)
	if err != nil {
		return nil, err
	}

	if content == nil ||
		content.Team == nil ||
		content.Team.Roster == nil ||
		content.Team.Roster.Players == nil ||
		content.Team.Roster.Players.Players == nil {
		return nil, errors.New("team roster not found")
	}

	results := make([]model.YahooPlayer, 0, 15)
	for _, p := range content.Team.Roster.Players.Players {
		y := model.YahooPlayer{
			YahooID: p.ID,
			Pos:     model.ParsePosition(p.Position),
		}
		if p.Name != nil {
			y.FirstName = p.Name.First
			y.LastName = p.Name.Last
		}
		if y.Pos == model.POS_DEF {
			y.FirstName = p.TeamFullName
		}
		results = append(results, y)
	}

	return results, nil
}

func (c *Client) yahooRequest(httpClient *http.Client, path string, args ...any) (*internal.FantasyContent, error) {
	p := fmt.Sprintf(path, args...)
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", c.url, p), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating yahoo http request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending yahoo http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from yahoo: %d", resp.StatusCode)
	}

	var res internal.FantasyContent
	err = xml.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, fmt.Errorf("error parsing response from yahoo: %w", err)
	}

	return &res, nil
}
