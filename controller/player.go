package controller

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mww/fantasy_manager_v2/model"
)

func (c *C) GetPlayer(ctx context.Context, id string) (*model.Player, error) {
	return c.db.GetPlayer(ctx, id)
}

func (c *C) UpdatePlayers(ctx context.Context) error {
	start := time.Now()
	log.Printf("update players starting at %v", start.Format(time.DateTime))

	players, err := c.sleeper.LoadPlayers()
	if err != nil {
		return err
	}

	for _, p := range players {
		err := c.db.SavePlayer(ctx, &p)
		if err != nil {
			return fmt.Errorf("error saving player (%s %s): %w", p.FirstName, p.LastName, err)
		}
	}

	log.Printf("load players finished, took %v", time.Since(start))
	return nil
}

func (c *C) RunPeriodicPlayerUpdates(shutdown chan bool, wg *sync.WaitGroup) {
	ticker := time.NewTicker(24 * time.Hour) // Make sure we update players once per day
	defer wg.Done()

	for {
		select {
		case <-shutdown:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := c.UpdatePlayers(ctx); err != nil {
				log.Printf("%v", err)
			}
		}
	}
}
