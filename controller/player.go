package controller

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mww/fantasy_manager_v2/model"
)

func (c *C) GetPlayer(ctx context.Context, id string) (*model.Player, error) {
	return c.db.GetPlayer(ctx, id)
}

func (c *C) Search(ctx context.Context, query string) ([]model.Player, error) {
	q, pos := getPositionFromQuery(query)
	q, team := getTeamFromQuery(q)

	if pos == model.POS_UNKNOWN && team == nil && q == "" {
		return nil, fmt.Errorf("error not a valid query: '%s", query)
	}
	return c.db.Search(ctx, q, pos, team)
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

var positionRegex = regexp.MustCompile(`(?i)(pos|position)\s*:\s*(?P<pos>\w+)`)

// Parse out the position from the query, returning the same query without the position.
// So if the query is "Tom pos:QB" this will return "Tom" and model.POS_QB.
// If the input query does not have a `pos:` argument then the function will return the
// input string and model.POS_UNKNOWN.
// Allowed tags for the position are `pos` and `positions` case insensitive.
func getPositionFromQuery(q string) (string, model.Position) {
	pos := model.POS_UNKNOWN
	m := positionRegex.FindStringSubmatch(q)
	if m != nil {
		p := m[positionRegex.SubexpIndex("pos")]
		pos = model.ParsePosition(p)
		q = strings.Replace(q, m[0], "", 1) // Remove the position match from the query
		q = strings.TrimSpace(q)            // Remove any remaining whitespace
	}

	return q, pos
}

var teamRegex = regexp.MustCompile(`(?i)team\s*:\s*(?P<team>\w+)`)

// Parse out the team from the query, returning the same query without the team.
// So if the query is "Brown team:PHI" this will return "Brown" and model.TEAM_PHI.
// If the input query does not have a `team:` argument then the function will return the
// input string and nil.
// The only allowed tag is `team` case insensitive.
func getTeamFromQuery(q string) (string, *model.NFLTeam) {
	var team *model.NFLTeam
	m := teamRegex.FindStringSubmatch(q)
	if m != nil {
		t := m[teamRegex.SubexpIndex("team")]
		team = model.ParseTeam(t)
		if team == model.TEAM_FA {
			team = nil
		}
		q = strings.Replace(q, m[0], "", 1) // Remove the team match from the query
		q = strings.TrimSpace(q)            // Remove any remaining whitespace
	}

	return q, team
}
