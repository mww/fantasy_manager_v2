package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"slices"
	"strconv"

	"github.com/mww/fantasy_manager_v2/model"
)

var (
	// Format looks like 449.l.149976.t.1
	teamIDRegex = regexp.MustCompile(`.+\.t\.(?P<id>\d+)`)
)

type yahooAdapter struct {
	c *controller
}

func (a *yahooAdapter) getLeagues(user, year string) ([]model.League, error) {
	return nil, errors.New("not implemented")
}

func (a *yahooAdapter) getLeagueName(ctx context.Context, leagueID, stateToken string) (string, error) {
	t, err := a.c.OAuthRetrieve(stateToken)
	if err != nil {
		return "", fmt.Errorf("error getting oauth token when getting league name: %w", err)
	}

	httpClient := a.c.yahooConfig.Client(ctx, t)
	return a.c.yahoo.GetLeagueName(httpClient, leagueID)
}

func (a *yahooAdapter) getManagers(ctx context.Context, l *model.League) ([]model.LeagueManager, error) {
	t, err := a.c.GetToken(ctx, l.ID)
	if err != nil {
		return nil, err
	}

	httpClient := a.c.yahooConfig.Client(ctx, t)
	return a.c.yahoo.GetManagers(httpClient, l.ExternalID)
}

func (a *yahooAdapter) sortManagers(m []model.LeagueManager) {
	parsedIDs := make(map[string]int)
	slices.SortFunc(m, func(a, b model.LeagueManager) int {
		idA, found := parsedIDs[a.ExternalID]
		if !found {
			idA = parseID(a.ExternalID)
			parsedIDs[a.ExternalID] = idA
		}

		idB, found := parsedIDs[b.ExternalID]
		if !found {
			idB = parseID(b.ExternalID)
			parsedIDs[b.ExternalID] = idB
		}

		return idA - idB
	})
}

func (a *yahooAdapter) getMatchupResults(ctx context.Context, l *model.League, week int) ([]model.Matchup, []model.PlayerScore, error) {
	t, err := a.c.GetToken(ctx, l.ID)
	if err != nil {
		return nil, nil, err
	}

	httpClient := a.c.yahooConfig.Client(ctx, t)
	matchups, err := a.c.yahoo.GetScoreboard(httpClient, l.ExternalID, week)
	if err != nil {
		return nil, nil, err
	}

	playerScores := make([]model.PlayerScore, 0) // Yahoo isn't providing this data
	return matchups, playerScores, nil
}

func (a *yahooAdapter) getRosters(ctx context.Context, l *model.League) ([]model.Roster, error) {
	t, err := a.c.GetToken(ctx, l.ID)
	if err != nil {
		return nil, err
	}

	results := make([]model.Roster, 0, len(l.Managers))
	httpClient := a.c.yahooConfig.Client(ctx, t)
	for _, t := range l.Managers {
		roster, err := a.c.yahoo.GetRoster(httpClient, t.ExternalID)
		if err != nil {
			return nil, err
		}

		ids, err := a.c.db.ConvertYahooPlayerIDs(ctx, roster)
		if err != nil {
			return nil, err
		}

		results = append(results, model.Roster{TeamID: t.ExternalID, PlayerIDs: ids})
	}

	return results, nil
}

func (a *yahooAdapter) getStarters(ctx context.Context, l *model.League) ([]model.RosterSpot, error) {
	t, err := a.c.GetToken(ctx, l.ID)
	if err != nil {
		return nil, err
	}

	httpClient := a.c.yahooConfig.Client(ctx, t)
	return a.c.yahoo.GetStarters(httpClient, l.ExternalID)
}

func (a *yahooAdapter) getLeagueStandings(ctx context.Context, leagueID string) ([]model.LeagueStanding, error) {
	return nil, errors.New("getLeagueStanding not supported for yahoo leagues")
}

func parseID(id string) int {
	result := 0
	m := teamIDRegex.FindStringSubmatch(id)
	if m != nil {
		part := m[teamIDRegex.SubexpIndex("id")]

		var err error
		result, err = strconv.Atoi(part)
		if err != nil {
			log.Printf("unable to parse yahoo team id '%s': %v", id, err)
		}
	} else {
		log.Printf("yahoo team id does not match regexp: '%s'", id)
	}
	return result
}
