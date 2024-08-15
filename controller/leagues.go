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

	switch platform {
	case model.PlatformSleeper:
		return c.getSleeperLeagues(ctx, username, year)
	default:
		return nil, errors.New("unsupported platform")
	}
}

func (c *controller) getSleeperLeagues(_ context.Context, username, year string) ([]model.League, error) {
	userID, err := c.sleeper.GetUserID(username)
	if err != nil {
		return nil, err
	}

	return c.sleeper.GetLeaguesForUser(userID, year)
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

func (c *controller) GetLeague(ctx context.Context, id int32) (*model.League, error) {
	return c.db.GetLeague(ctx, id)
}

func (c *controller) ListLeagues(ctx context.Context) ([]model.League, error) {
	return c.db.ListLeagues(ctx)
}
