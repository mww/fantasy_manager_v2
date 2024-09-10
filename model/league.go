package model

var PlatformSleeper = "sleeper"
var PlatformYahoo = "yahoo"

type League struct {
	ID         int32
	Platform   string
	ExternalID string
	Name       string
	Year       string
	Archived   bool
	Managers   []LeagueManager
}

type LeagueManager struct {
	ExternalID  string
	TeamName    string
	ManagerName string
	JoinKey     string
}

// TODO - getting a Match from the point of view of one of the teams, the other is opponent
type TeamResult struct {
	TeamID   string
	TeamName string
	Score    int32
	JoinKey  string // Not persisted
}

type Matchup struct {
	TeamA     *TeamResult
	TeamB     *TeamResult
	MatchupID int32
	Week      int
}
