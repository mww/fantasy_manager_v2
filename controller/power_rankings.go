package controller

import (
	"context"
	"fmt"
	"log"
	"math"
	"slices"

	"github.com/mww/fantasy_manager_v2/model"
)

func (c *controller) GetPowerRanking(ctx context.Context, leagueID, powerRankingID int32) (*model.PowerRanking, error) {
	return c.db.GetPowerRanking(ctx, leagueID, powerRankingID)
}

func (c *controller) CalculatePowerRanking(ctx context.Context, leagueID, rankingID int32, week int) (int32, error) {
	l, err := c.GetLeague(ctx, leagueID)
	if err != nil {
		return 0, fmt.Errorf("error getting league with id %d: %w", leagueID, err)
	}

	adaptor := getPlatformAdapter(l.Platform, c)

	rosters, err := adaptor.getRosters(l)
	if err != nil {
		return 0, fmt.Errorf("error getting league rosters: %w", err)
	}

	ranking, err := c.GetRanking(ctx, rankingID)
	if err != nil {
		return 0, fmt.Errorf("error getting ranking with id %d: %w", rankingID, err)
	}

	starters, err := adaptor.getStarters(l)
	if err != nil {
		return 0, fmt.Errorf("error getting starters list for league %d: %w", l.ID, err)
	}

	weeklyResults := make(map[int][]model.Matchup)
	for w := week; w > 0; w-- {
		// only use a max of last 3 weeks of data
		if len(weeklyResults) == 3 {
			break
		}
		results, err := c.GetLeagueResults(ctx, leagueID, w)
		if err != nil {
			log.Printf("error getting results for league %d, week %d", leagueID, w)
			continue
		}
		weeklyResults[w] = results
	}

	powerRanking := initializePowerRankings(rosters, ranking, week)
	calculateRosterScores(powerRanking, starters)
	calculateFantasyPointsScore(powerRanking, weeklyResults)
	// Calculate more parts of the scores
	sumFinalScore(powerRanking)

	// Sort by score
	slices.SortFunc(powerRanking.Teams, func(a, b model.TeamPowerRanking) int {
		return int(b.TotalScore - a.TotalScore)
	})
	// Assign a rank value
	for i := range powerRanking.Teams {
		powerRanking.Teams[i].Rank = i + 1
	}

	id, err := c.db.SavePowerRanking(ctx, leagueID, powerRanking)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func initializePowerRankings(rosters []model.Roster, ranking *model.Ranking, week int) *model.PowerRanking {
	powerRanking := &model.PowerRanking{
		RankingID: ranking.ID,
		Teams:     make([]model.TeamPowerRanking, 0, len(rosters)),
		Week:      int16(week),
	}
	for _, r := range rosters {
		pr := model.TeamPowerRanking{
			TeamID: r.TeamID,
			Roster: make([]model.PowerRankingPlayer, 0, len(r.PlayerIDs)),
		}

		for _, id := range r.PlayerIDs {
			if p, found := ranking.Players[id]; found {
				pr.Roster = append(pr.Roster, model.FromRankingPlayer(&p))
			} else {
				// Players not included in the ranking won't have a significant impact on
				// the overall power ranking, so we can just create a dummy entry for power
				// ranking purposes. Assign to the rank of 1000 to show that the player
				// is not valuable in these calculations
				pr.Roster = append(pr.Roster, model.PowerRankingPlayer{PlayerID: id, Rank: 1000})
			}
		}
		// Sort the roster by rank, so that the higher ranked players will come first.
		slices.SortFunc(pr.Roster, func(a, b model.PowerRankingPlayer) int {
			return int(a.Rank - b.Rank)
		})
		powerRanking.Teams = append(powerRanking.Teams, pr)
	}

	return powerRanking
}

func calculateRosterScores(powerRanking *model.PowerRanking, starters []model.RosterSpot) {
	for i := range powerRanking.Teams {
		usedPlayers := make(map[string]bool)
		// Go through all the starters and select the highest ranked player on the roster that matches
		// the roster spot and hasn't already been used.
		for _, s := range starters {
			for j, p := range powerRanking.Teams[i].Roster {
				if s.IsAllowed(p.Position) {
					if _, used := usedPlayers[p.PlayerID]; !used {
						v := calculatePlayerValue(p.Rank)
						powerRanking.Teams[i].Roster[j].PowerRankingPoints = v
						powerRanking.Teams[i].Roster[j].IsStarter = true
						powerRanking.Teams[i].RosterScore += v
						usedPlayers[p.PlayerID] = true
						break
					}
				}
			}
		}

		// Once all the starters are selected, put the rest of the players on the bench
		for j, p := range powerRanking.Teams[i].Roster {
			if !powerRanking.Teams[i].Roster[j].IsStarter {
				v := int32(float64(calculatePlayerValue(p.Rank)) * 0.4)
				powerRanking.Teams[i].Roster[j].PowerRankingPoints = v
				powerRanking.Teams[i].RosterScore += v
			}
		}

		powerRanking.Teams[i].RosterScore = powerRanking.Teams[i].RosterScore / 100
	}
}

// Get the score for both points for and points against scored.
func calculateFantasyPointsScore(pr *model.PowerRanking, weeklyResults map[int][]model.Matchup) {
	type points struct {
		pointsFor     int32
		pointsAgainst int32
		matches       int32
	}
	data := make(map[string]*points)

	for _, matchups := range weeklyResults {
		for _, m := range matchups {
			a, found := data[m.TeamA.TeamID]
			if !found {
				a = &points{}
				data[m.TeamA.TeamID] = a
			}
			b, found := data[m.TeamB.TeamID]
			if !found {
				b = &points{}
				data[m.TeamB.TeamID] = b
			}
			a.pointsFor += m.TeamA.Score
			a.pointsAgainst += m.TeamB.Score
			a.matches += 1

			b.pointsFor += m.TeamB.Score
			b.pointsAgainst += m.TeamA.Score
			b.matches += 1
		}
	}

	for i := range pr.Teams {
		p, found := data[pr.Teams[i].TeamID]
		if !found {
			log.Printf("did not find points data for team %s (%s)", pr.Teams[i].TeamID, pr.Teams[i].TeamName)
			continue
		}
		pr.Teams[i].PointsForScore = (p.pointsFor / p.matches)
		pr.Teams[i].PointsAgainstScore = int32(math.Round(0.3 * float64(p.pointsAgainst/p.matches))) // take 30% points against

		// Since we store points * 1000 in the DB, divid by 1000 here to get back to normal
		pr.Teams[i].PointsForScore /= 1000
		pr.Teams[i].PointsAgainstScore /= 1000
	}
}

func sumFinalScore(pr *model.PowerRanking) {
	for i, t := range pr.Teams {
		pr.Teams[i].TotalScore = t.RosterScore + t.RecordScore + t.StreakScore + t.PointsForScore + t.PointsAgainstScore
	}
}

// calcValue uses an expential decay function in the form of y = a(1-b)^x
// to calculate a player value for a given rank. The values picked for
// a and b were to get results so that the top 500 players have a value
// greater than 1 and each value step has a different score.
func calculatePlayerValue(rank int32) int32 {
	return int32(math.Ceil(10000 * math.Pow(0.983, float64(rank))))
}
