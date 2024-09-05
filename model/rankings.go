package model

import (
	"time"
)

type Ranking struct {
	ID   int32
	Date time.Time
	// Map of players indexed by player id
	Players map[string]RankingPlayer
}

type RankingPlayer struct {
	Rank      int32
	ID        string
	FirstName string
	LastName  string
	Position  Position
	Team      *NFLTeam
}
