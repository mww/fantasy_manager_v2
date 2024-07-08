package controller

import (
	"github.com/mww/fantasy_manager_v2/db"
	"github.com/mww/fantasy_manager_v2/sleeper"
)

// C encapsulates business logic without worrying about any web layers
type C struct {
	sleeper sleeper.Client
	db      db.DB
}

func New(sleeper sleeper.Client, db db.DB) (*C, error) {
	c := &C{
		sleeper: sleeper,
		db:      db,
	}
	return c, nil
}
