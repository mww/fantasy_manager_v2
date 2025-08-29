package internal

type Player struct {
	ID        string `json:"playerID"`
	SleeperID string `json:"sleeperBotID"`
	FullName  string `json:"longName"`
}

type NFLPlayerResponse struct {
	StatusCode int      `json:"statusCode"`
	Body       []Player `json:"body"`
}
