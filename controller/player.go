package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mww/fantasy_manager_v2/model"
)

func (c *controller) GetPlayer(ctx context.Context, id string) (*model.Player, error) {
	return c.db.GetPlayer(ctx, id)
}

func (c *controller) Search(ctx context.Context, query string) ([]model.Player, error) {
	q, pos := getPositionFromQuery(query)
	q, team := getTeamFromQuery(q)

	if pos == model.POS_UNKNOWN && team == nil && q == "" {
		return nil, fmt.Errorf("error not a valid query: '%s'", query)
	}
	return c.db.Search(ctx, q, pos, team)
}

// Updates a player's nickname, or deletes it if the nickname == ""
// Returns an error if not successful, nil otherwise.
func (c *controller) UpdatePlayerNickname(ctx context.Context, id, nickname string) error {
	p, err := c.db.GetPlayer(ctx, id)
	if err != nil {
		return err
	}

	if p.Nickname1 == nickname {
		return errors.New("no updated needed")
	}

	// Delete the nickname
	if nickname == "" {
		return c.db.DeleteNickname(ctx, id, p.Nickname1)
	}

	p.Nickname1 = nickname
	return c.db.SavePlayer(ctx, p)
}

func (c *controller) UpdatePlayers(ctx context.Context) error {
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

// Add a new rankings for players. This will parse the data from the reader (in CSV format) and
// create a new rankings data point. Returns the id of the new rankings and an error if there
// was one.
func (c *controller) AddRankings(r io.Reader, date time.Time) (string, error) {
	// TODO
	log.Printf("in AddRankings()")
	return "0", nil
}

func (c *controller) RunPeriodicPlayerUpdates(frequency time.Duration, shutdown chan bool, wg *sync.WaitGroup) {
	ticker := time.NewTicker(frequency)
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
