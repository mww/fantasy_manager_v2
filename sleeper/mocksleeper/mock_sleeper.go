package mocksleeper

import (
	"github.com/mww/fantasy_manager_v2/model"
	"github.com/stretchr/testify/mock"
)

type Client struct {
	mock.Mock
}

func (c *Client) LoadPlayers() ([]model.Player, error) {
	args := c.Called()

	var res []model.Player
	if args.Get(0) != nil {
		res = args.Get(0).([]model.Player)
	}

	return res, args.Error(1)
}

func (c *Client) GetUserID(username string) (string, error) {
	args := c.Called(username)
	return args.String(0), args.Error(1)
}

func (c *Client) GetLeaguesForUser(userID, year string) ([]model.League, error) {
	args := c.Called(userID, year)

	var res []model.League
	if args.Get(0) != nil {
		res = args.Get(0).([]model.League)
	}

	return res, args.Error(1)
}
