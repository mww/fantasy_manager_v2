package model

import (
	"strings"
	"time"
)

type PowerRanking struct {
	ID        int32
	RankingID int32 // The ID of the ranking data to use
	Week      int16 // week used to calculate win/loss and streaks
	Teams     []TeamPowerRanking
	Created   time.Time
}

type TeamPowerRanking struct {
	TeamID             string
	TeamName           string
	Rank               int
	RankChange         int
	TotalScore         int32
	RosterScore        int32
	RecordScore        int32
	StreakScore        int32
	PointsForScore     int32
	PointsAgainstScore int32
	Roster             []PowerRankingPlayer
}

type PowerRankingPlayer struct {
	PlayerID           string
	Rank               int32
	FirstName          string
	LastName           string
	Position           Position
	NFLTeam            *NFLTeam
	PowerRankingPoints int32
	IsStarter          bool
}

func FromRankingPlayer(p *RankingPlayer) PowerRankingPlayer {
	return PowerRankingPlayer{
		PlayerID:  p.ID,
		Rank:      p.Rank,
		FirstName: p.FirstName,
		LastName:  p.LastName,
		Position:  p.Position,
		NFLTeam:   p.Team,
	}
}

type Roster struct {
	TeamID    string
	PlayerIDs []string
}

func GetRosterSpot(pos string) RosterSpot {
	if strings.ToUpper(pos) == "FLEX" {
		return RosterSpot{Allowed: []Position{POS_RB, POS_WR, POS_TE}}
	}
	return RosterSpot{Allowed: []Position{ParsePosition(pos)}}
}

type RosterSpot struct {
	Allowed []Position
}

func (rs *RosterSpot) IsAllowed(pos Position) bool {
	for _, a := range rs.Allowed {
		if a == pos {
			return true
		}
	}
	return false
}
