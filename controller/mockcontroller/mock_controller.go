package mockcontroller

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/mww/fantasy_manager_v2/model"
	"github.com/stretchr/testify/mock"
)

type C struct {
	mock.Mock
}

func (c *C) GetPlayer(ctx context.Context, id string) (*model.Player, error) {
	args := c.Called(ctx, id)

	var p *model.Player
	if args.Get(0) != nil {
		p = args.Get(0).(*model.Player)
	}

	return p, args.Error(1)
}

func (c *C) Search(ctx context.Context, query string) ([]model.Player, error) {
	args := c.Called(ctx, query)

	var res []model.Player
	if args.Get(0) != nil {
		res = args.Get(0).([]model.Player)
	}

	return res, args.Error(1)
}

func (c *C) UpdatePlayerNickname(ctx context.Context, id, nickname string) error {
	args := c.Called(ctx, id, nickname)
	return args.Error(0)
}

func (c *C) UpdatePlayers(ctx context.Context) error {
	args := c.Called(ctx)
	return args.Error(0)
}

func (c *C) AddRankings(r io.Reader, date time.Time) (string, error) {
	args := c.Called(r, date)
	return args.String(0), args.Error(1)
}

func (c *C) RunPeriodicPlayerUpdates(frequency time.Duration, shutdown chan bool, wg *sync.WaitGroup) {
	c.Called(frequency, shutdown, wg)
}
