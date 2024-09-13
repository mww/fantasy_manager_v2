package internal

type FantasyContent struct {
	League *League `xml:"league"`
	Team   *Team   `xml:"team"`
}

type League struct {
	Key        string      `xml:"league_key"`
	Name       string      `xml:"name"`
	Settings   *Settings   `xml:"settings"`
	Standings  *Standings  `xml:"standings"`
	Scoreboard *Scoreboard `xml:"scoreboard"`
}

type Settings struct {
	RosterPositions *RosterPositions `xml:"roster_positions"`
}

type RosterPositions struct {
	Positions []RosterPosition `xml:"roster_position"`
}

type RosterPosition struct {
	Position string `xml:"position"`
	Count    int    `xml:"count"`
}

type Standings struct {
	Teams *Teams `xml:"teams"`
}

type Teams struct {
	Teams []Team `xml:"team"`
}

type Team struct {
	Key        string      `xml:"team_key"`
	Name       string      `xml:"name"`
	Managers   *Managers   `xml:"managers"`
	TeamPoints *TeamPoints `xml:"team_points"`
	Roster     *Roster     `xml:"roster"`
}

type Managers struct {
	Managers []Manager `xml:"manager"`
}

type Manager struct {
	Nickname string `xml:"nickname"`
}

type Scoreboard struct {
	Week     int       `xml:"week"`
	Matchups *Matchups `xml:"matchups"`
}

type Matchups struct {
	Matchups []Matchup `xml:"matchup"`
}

type Matchup struct {
	Teams *Teams `xml:"teams"`
}

type TeamPoints struct {
	Total float64 `xml:"total"`
}

type Roster struct {
	Players *Players `xml:"players"`
}

type Players struct {
	Players []Player `xml:"player"`
}

type Player struct {
	Key          string      `xml:"player_key"`
	ID           string      `xml:"player_id"`
	Name         *PlayerName `xml:"name"`
	Position     string      `xml:"primary_position"`
	TeamFullName string      `xml:"editorial_team_full_name"`
}

type PlayerName struct {
	First string `xml:"first"`
	Last  string `xml:"last"`
}
