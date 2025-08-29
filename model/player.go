package model

import (
	"fmt"
	"strings"
	"time"
)

const (
	RookieYearFormat = "2006"
)

type Player struct {
	ID              string
	YahooID         string
	Tank01ID        string
	FirstName       string
	LastName        string
	Nickname1       string
	Position        Position
	Team            *NFLTeam
	Weight          int
	Height          int
	BirthDate       time.Time
	RookieYear      time.Time
	YearsExp        int
	Jersey          int
	DepthChartOrder int
	College         string
	Active          bool
	Created         time.Time
	Updated         time.Time
	Changes         []Change
}

func (p *Player) FormattedBirthDate() string {
	if p.BirthDate.IsZero() {
		return "unknown"
	}
	return p.BirthDate.Format(time.DateOnly)
}

func (p *Player) FormattedRookieYear() string {
	if p.RookieYear.IsZero() {
		return "unknown"
	}
	return p.RookieYear.Format(RookieYearFormat)
}

func (p *Player) FormattedCreatedTime() string {
	if p.Created.IsZero() {
		return "unknown"
	}
	return p.Created.Format(time.DateTime)
}

func (p *Player) FormattedUpdatedTime() string {
	if p.Updated.IsZero() {
		return "unknown"
	}
	return p.Updated.Format(time.DateTime)
}

type Change struct {
	Time         time.Time
	PropertyName string
	OldValue     string
	NewValue     string
}

func (c *Change) String() string {
	return fmt.Sprintf("%s changed from '%s' to '%s'", c.PropertyName, c.OldValue, c.NewValue)
}

// PlayerScore represents how many fantasy points a specific player scored in a single week in a single league.
// FirstName and LastName are typically empty, but used when getting the top scores for a given week.
type PlayerScore struct {
	PlayerID  string
	FirstName string
	LastName  string
	Score     int32
}

// SeasonScores aggregates all the weekly scores for a player in a league. It
// is to make displaying the data more natural. All of the scores are saved in
// a slice. To make things easier Scores[0] is not used. Week 1 is at Scores[1],
// Week 2 is at Scores[2], etc.
type SeasonScores struct {
	LeagueID   int32
	LeagueName string
	LeagueYear string
	PlayerID   string
	Scores     []int32
}

// Take a full name, like "Deebo Samuel Sr."" and return "Deebo Samuel".
func TrimNameSuffix(fullName string) string {
	suffixList := []string{
		"Jr.",
		"Sr.",
		"II",
		"IV",
	}

	for _, s := range suffixList {
		fullName = strings.TrimSuffix(fullName, s)
	}

	return strings.TrimSpace(fullName)
}
