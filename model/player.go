package model

import (
	"fmt"
	"time"
)

const (
	RookieYearFormat = "2006"
)

type Player struct {
	ID              string
	YahooID         string
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
	return fmt.Sprintf("%s changed from %s to %s", c.PropertyName, c.OldValue, c.NewValue)
}
