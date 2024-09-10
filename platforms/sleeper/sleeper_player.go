package sleeper

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/mww/fantasy_manager_v2/model"
)

var (
	heightRegex = regexp.MustCompile(`(?P<feet>\d+)'(?P<inches>\d+)"`)
	zeroTime    = time.Time{}
)

type sleeperPlayer struct {
	ID              string    `json:"player_id"`
	YahooID         int       `json:"yahoo_id"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	Position        string    `json:"position"`
	Team            string    `json:"team"`
	Weight          string    `json:"weight"`
	Height          string    `json:"height"`
	BirthDate       string    `json:"birth_date"`
	YearsExp        int       `json:"years_exp"`
	JerseyNumber    int       `json:"number"`
	DepthChartOrder int       `json:"depth_chart_order"`
	College         string    `json:"college"`
	Active          bool      `json:"active"`
	Metadata        *metadata `json:"metadata"`
}

type metadata struct {
	RookieYear string `json:"rookie_year"`
}

func (p *sleeperPlayer) toPlayer() *model.Player {
	return &model.Player{
		ID:              p.ID,
		YahooID:         formatYahooID(p.YahooID),
		FirstName:       p.FirstName,
		LastName:        p.LastName,
		Position:        model.ParsePosition(p.Position),
		Team:            model.ParseTeam(p.Team),
		Weight:          parseInt(p.Weight, p.ID),
		Height:          parseHeight(p.Height, p.ID),
		BirthDate:       parseBirthdate(p.BirthDate, p.ID),
		RookieYear:      parseRookieYear(p.Metadata, p.ID),
		YearsExp:        p.YearsExp,
		Jersey:          p.JerseyNumber,
		DepthChartOrder: p.DepthChartOrder,
		College:         p.College,
		Active:          p.Active,
	}
}

func formatYahooID(id int) string {
	if id == 0 {
		return ""
	}
	return fmt.Sprintf("%d", id)
}

func parseInt(i, playerID string) int {
	if i == "" {
		return 0
	}
	v, err := strconv.Atoi(i)
	if err != nil {
		log.Printf("error converting string to int for player %s: %v", playerID, v)
		return 0
	}
	return v
}

// Get the height of the player in inches
func parseHeight(h, playerID string) int {
	if h == "" {
		return 0
	}

	// Sometimes the heights are listed like `5'11"` and we must convert that
	// to inches before returning it.
	m := heightRegex.FindStringSubmatch(h)
	if m != nil {
		feet := m[heightRegex.SubexpIndex("feet")]
		inches := m[heightRegex.SubexpIndex("inches")]
		f := parseInt(feet, playerID)
		if f == 0 {
			return 0
		}
		i := parseInt(inches, playerID)
		return (f * 12) + i
	}

	// Default case, the height is just listed in inches, but as a string
	height, err := strconv.Atoi(h)
	if err != nil {
		log.Printf("error parsing player height for %s (%s): %v", playerID, h, err)
		return 0
	}
	return height
}

func parseBirthdate(b, playerID string) time.Time {
	if b == "" {
		return zeroTime
	}

	d, err := time.Parse(time.DateOnly, b)
	if err != nil {
		log.Printf("error parsing birthdate for %s: %v", playerID, err)
		return zeroTime
	}
	return d
}

func parseRookieYear(m *metadata, playerID string) time.Time {
	if m == nil || m.RookieYear == "" || m.RookieYear == "0" {
		return zeroTime
	}

	d, err := time.Parse(model.RookieYearFormat, m.RookieYear)
	if err != nil {
		log.Printf("error parsing rookie year for %s: %v", playerID, err)
		return zeroTime
	}
	return d
}
