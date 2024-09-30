package controller

import (
	"context"
	"fmt"

	"github.com/mww/fantasy_manager_v2/model"
)

type sleeperAdapter struct {
	c *controller
}

func (a *sleeperAdapter) getLeagues(user, year string) ([]model.League, error) {
	userID, err := a.c.sleeper.GetUserID(user)
	if err != nil {
		return nil, err
	}

	return a.c.sleeper.GetLeaguesForUser(userID, year)
}

func (a *sleeperAdapter) getLeagueName(ctx context.Context, leagueID, stateToken string) (string, error) {
	return a.c.sleeper.GetLeagueName(leagueID)
}

func (a *sleeperAdapter) getManagers(ctx context.Context, l *model.League) ([]model.LeagueManager, error) {
	managers, err := a.c.sleeper.GetLeagueManagers(l.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("error loading managers from sleeper for %s: %w", l.ExternalID, err)
	}
	return managers, nil
}

func (a *sleeperAdapter) sortManagers(m []model.LeagueManager) {
	a.c.sleeper.SortManagers(m)
}

func (a *sleeperAdapter) getMatchupResults(ctx context.Context, l *model.League, week int) ([]model.Matchup, []model.PlayerScore, error) {
	matchups, scores, err := a.c.sleeper.GetMatchupResults(l.ExternalID, week)
	if err != nil {
		return nil, nil, err
	}

	// Fill in the TeamID fields based on the join key
	owners := make(map[string]string)
	for _, manager := range l.Managers {
		owners[manager.JoinKey] = manager.ExternalID
	}

	for i, m := range matchups {
		matchups[i].TeamA.TeamID = owners[m.TeamA.JoinKey]
		matchups[i].TeamB.TeamID = owners[m.TeamB.JoinKey]
	}

	return matchups, scores, nil
}

func (a *sleeperAdapter) getRosters(ctx context.Context, l *model.League) ([]model.Roster, error) {
	return a.c.sleeper.GetRosters(l.ExternalID)
}

func (a *sleeperAdapter) getStarters(ctx context.Context, l *model.League) ([]model.RosterSpot, error) {
	return a.c.sleeper.GetStarters(l.ExternalID)
}

func (a *sleeperAdapter) getLeagueStandings(ctx context.Context, leagueID string) ([]model.LeagueStanding, error) {
	return a.c.sleeper.GetLeagueStandings(leagueID)
}
