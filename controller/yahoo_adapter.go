package controller

import (
	"context"
	"errors"
	"fmt"

	"github.com/mww/fantasy_manager_v2/model"
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
}

func (a *yahooAdapter) getMatchupResults(l *model.League, week int) ([]model.Matchup, []model.PlayerScore, error) {
	return nil, nil, errors.New("not implemented")
}

func (a *yahooAdapter) getRosters(l *model.League) ([]model.Roster, error) {
	return nil, errors.New("not implemented")
}

func (a *yahooAdapter) getStarters(l *model.League) ([]model.RosterSpot, error) {
	return nil, errors.New("not implemented")
}
