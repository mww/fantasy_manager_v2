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
