package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mww/fantasy_manager_v2/model"
)

const yearOnlyFormat = "2006"

func (c *controller) GetLeaguesFromPlatform(ctx context.Context, username, platform, year string) ([]model.League, error) {
	if _, err := time.Parse(yearOnlyFormat, year); err != nil {
		return nil, fmt.Errorf("year parameter must be in the YYYY format, got: %s", year)
	}

	return getPlatformAdapter(platform, c).getLeagues(username, year)
}

func (c *controller) AddLeague(ctx context.Context, platform, externalID, name, year string) (*model.League, error) {
	if !model.IsPlatformSupported(platform) {
		return nil, fmt.Errorf("%s is not a supported platform", platform)
	}

	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return nil, errors.New("externalID must be provided")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("league name must be provided")
	}

	if _, err := time.Parse(yearOnlyFormat, year); err != nil {
		return nil, fmt.Errorf("year parameter must be in the YYYY format, got: %s", year)
	}

	l := &model.League{
		Platform:   platform,
		ExternalID: externalID,
		Name:       name,
		Year:       year,
	}

	if err := c.db.AddLeague(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (c *controller) AddLeagueManagers(ctx context.Context, leagueID int32) (*model.League, error) {
	l, err := c.GetLeague(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("error getting league from DB: %w", err)
	}

	l.Managers, err = getPlatformAdapter(l.Platform, c).getManagers(l)
	if err != nil {
		return nil, fmt.Errorf("error getting managers: %w", err)
	}

	for _, m := range l.Managers {
		if err := c.db.SaveLeagueManager(ctx, leagueID, &m); err != nil {
			return nil, fmt.Errorf("error saving league manager: %w", err)
		}
	}

	return c.GetLeague(ctx, leagueID)
}

func (c *controller) GetLeague(ctx context.Context, id int32) (*model.League, error) {
	l, err := c.db.GetLeague(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error looking up league: %w", err)
	}

	l.Managers, err = c.db.GetLeagueManagers(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error looking up league manages: %w", err)
	}

	getPlatformAdapter(l.Platform, c).sortManagers(l.Managers)

	return l, nil
}

func (c *controller) ListLeagues(ctx context.Context) ([]model.League, error) {
	return c.db.ListLeagues(ctx)
}

func (c *controller) ArchiveLeague(ctx context.Context, id int32) error {
	return c.db.ArchiveLeague(ctx, id)
}

func (c *controller) SyncResultsFromPlatform(ctx context.Context, leagueID int32, week int) error {
	l, err := c.db.GetLeague(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("error looking up league: %w", err)
	}
	l.Managers, err = c.db.GetLeagueManagers(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("error loading league managers: %w", err)
	}

	matchups, scores, err := getPlatformAdapter(l.Platform, c).getMatchupResults(l, week)
	if err != nil {
		return fmt.Errorf("error getting matchup results: %w", err)
	}

	if err := c.db.SaveResults(ctx, l.ID, matchups); err != nil {
		return fmt.Errorf("error saving matchup results: %w", err)
	}

	if err := c.db.SavePlayerScores(ctx, l.ID, week, scores); err != nil {
		return fmt.Errorf("error saving player scores: %w", err)
	}

	return nil
}

func (c *controller) GetLeagueResults(ctx context.Context, leagueID int32, week int) ([]model.Matchup, error) {
	return c.db.GetResults(ctx, leagueID, week)
}
