package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/itbasis/go-clock"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mww/fantasy_manager_v2/model"
	"golang.org/x/oauth2"
)

var (
	ErrPlayerNotFound error = errors.New("player not found")
)

func New(ctx context.Context, connString string, clock clock.Clock) (DB, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return &postgresDB{pool: pool, clock: clock}, nil
}

type postgresDB struct {
	pool  *pgxpool.Pool
	clock clock.Clock
}

func (db *postgresDB) ListRankings(ctx context.Context) ([]model.Ranking, error) {
	const query = "SELECT id, ranking_date FROM rankings ORDER BY ranking_date DESC LIMIT 25"

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying for rankings: %w", err)
	}

	results := make([]model.Ranking, 0, 25)
	for rows.Next() {
		r, err := scanRanking(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error on rows: %w", err)
	}

	return results, nil
}

func (db *postgresDB) GetRanking(ctx context.Context, id int32) (*model.Ranking, error) {
	const metadataQuery = "SELECT id, ranking_date FROM rankings WHERE id=@id"
	const rankingsQuery = `SELECT player_rankings.ranking, players.id, players.name_first, players.name_last, players.position, players.team
							FROM player_rankings INNER JOIN players ON player_rankings.player_id=players.id
							WHERE player_rankings.ranking_id=@id
							ORDER BY player_rankings.ranking ASC`

	args := pgx.NamedArgs{
		"id": id,
	}
	row := db.pool.QueryRow(ctx, metadataQuery, args)
	ranking, err := scanRanking(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("no ranking with specified id found")
		}
		return nil, err
	}
	ranking.Players = make(map[string]model.RankingPlayer)

	rows, err := db.pool.Query(ctx, rankingsQuery, args)
	if err != nil {
		return nil, fmt.Errorf("error querying for rankings data: %w", err)
	}

	for rows.Next() {
		p := model.RankingPlayer{}
		var pos, team string
		if err := rows.Scan(&p.Rank, &p.ID, &p.FirstName, &p.LastName, &pos, &team); err != nil {
			return nil, fmt.Errorf("error reading rankings data: %w", err)
		}
		p.Position = model.ParsePosition(pos)
		p.Team = model.ParseTeam(team)
		ranking.Players[p.ID] = p
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading results of rankings query: %w", err)
	}

	if len(ranking.Players) == 0 {
		return nil, fmt.Errorf("ranking with id %d has no actual rankings - this should not happen", ranking.ID)
	}

	return ranking, nil
}

func (db *postgresDB) AddRanking(ctx context.Context, date time.Time, rankings map[string]int32) (*model.Ranking, error) {
	const insertRankingQuery = "INSERT INTO rankings(ranking_date) VALUES (@date) RETURNING id"
	const insertPlayerRankingQuery = "INSERT INTO player_rankings(ranking_id, player_id, ranking) VALUES (@rankingID, @playerID, @ranking)"

	if date.IsZero() {
		return nil, errors.New("rankings date must be provided")
	}
	if len(rankings) == 0 {
		return nil, errors.New("rankings cannot be empty")
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	r := &model.Ranking{
		Date:    date,
		Players: make(map[string]model.RankingPlayer),
	}

	err = tx.QueryRow(ctx, insertRankingQuery, pgx.NamedArgs{"date": date}).Scan(&r.ID)
	if err != nil {
		return nil, fmt.Errorf("error inserting ranking into rankings table: %w", err)
	}
	if r.ID <= 0 {
		return nil, fmt.Errorf("did not get a valid rankingID, got: %d", r.ID)
	}

	for playerID, ranking := range rankings {
		args := pgx.NamedArgs{
			"rankingID": r.ID,
			"playerID":  playerID,
			"ranking":   ranking,
		}
		if _, err := tx.Exec(ctx, insertPlayerRankingQuery, args); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgErr.ConstraintName == "player_rankings_player_id_fkey" {
					return nil, fmt.Errorf("no player with id: %s", playerID)
				}
			}
			return nil, fmt.Errorf("error inserting player ranking: %w", err)
		}
		r.Players[playerID] = model.RankingPlayer{Rank: ranking, ID: playerID}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("error commiting add rankings transactions: %w", err)
	}

	return r, nil
}

func (db *postgresDB) DeleteRanking(ctx context.Context, id int32) error {
	const deleteMetadataQuery = "DELETE FROM rankings WHERE id=@id"
	const deleteRankingsQuery = "DELETE FROM player_rankings WHERE ranking_id=@id"

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error begining transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	args := pgx.NamedArgs{
		"id": id,
	}
	tag, err := tx.Exec(ctx, deleteRankingsQuery, args)
	if err != nil {
		return fmt.Errorf("error deleting from player_rankings: %w", err)
	}
	if tag.RowsAffected() <= 0 {
		return fmt.Errorf("no rows deleted from player_rankings for ranking_id %d", id)
	}

	tag2, err2 := tx.Exec(ctx, deleteMetadataQuery, args)
	if err2 != nil {
		return fmt.Errorf("error deleting from rankings: %w", err2)
	}
	if tag2.RowsAffected() != 1 {
		return fmt.Errorf("wrong number of rows affected when deleting ranking %d, expected 1, got %d", id, tag2.RowsAffected())
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error commiting delete ranking transaction: %w", err)
	}

	return nil
}

func (db *postgresDB) ListLeagues(ctx context.Context) ([]model.League, error) {
	const listLeaguesQuery = `SELECT id, platform, external_id, name, year, archived FROM leagues WHERE archived=false`

	rows, err := db.pool.Query(ctx, listLeaguesQuery)
	if err != nil {
		return nil, fmt.Errorf("error listing leagues: %w", err)
	}

	leagues := make([]model.League, 0, 4)
	for rows.Next() {
		l := model.League{}

		if err := rows.Scan(&l.ID, &l.Platform, &l.ExternalID, &l.Name, &l.Year, &l.Archived); err != nil {
			return nil, fmt.Errorf("error reading league: %w", err)
		}
		leagues = append(leagues, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading results leagues query: %w", err)
	}
	return leagues, nil
}

func (db *postgresDB) GetLeague(ctx context.Context, id int32) (*model.League, error) {
	const leagueQuery = `SELECT platform, external_id, name, year, archived FROM leagues WHERE id=@id`

	l := model.League{ID: id}

	args := pgx.NamedArgs{"id": id}
	err := db.pool.QueryRow(ctx, leagueQuery, args).Scan(&l.Platform, &l.ExternalID, &l.Name, &l.Year, &l.Archived)
	if err != nil {
		return nil, fmt.Errorf("error querying league: %w", err)
	}

	return &l, nil
}

func (db *postgresDB) GetLeagueManagers(ctx context.Context, leagueID int32) ([]model.LeagueManager, error) {
	const query = `SELECT external_id, team_name, manager_name, join_key FROM league_managers WHERE league_id=@id`

	rows, err := db.pool.Query(ctx, query, pgx.NamedArgs{"id": leagueID})
	if err != nil {
		return nil, fmt.Errorf("error querying league managers: %w", err)
	}

	managers := make([]model.LeagueManager, 0, 12)
	for rows.Next() {
		m := model.LeagueManager{}

		if err := rows.Scan(&m.ExternalID, &m.TeamName, &m.ManagerName, &m.JoinKey); err != nil {
			return nil, fmt.Errorf("error reading league manager: %w", err)
		}
		managers = append(managers, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading results for leagues managers: %w", err)
	}

	return managers, nil
}

func (db *postgresDB) SaveLeagueManager(ctx context.Context, leagueID int32, manager *model.LeagueManager) error {
	const query = `SELECT COUNT(*) FROM league_managers WHERE league_id=@leagueID AND external_id=@externalID`
	const update = `UPDATE league_managers SET team_name=@teamName, manager_name=@managerName, join_key=@joinKey WHERE league_id=@leagueID AND external_id=@externalID`
	const insert = `INSERT INTO league_managers(league_id, external_id, team_name, manager_name, join_key) 
		VALUES (@leagueID, @externalID, @teamName, @managerName, @joinKey)`

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Not all of these args are needed for the query, but they are for the update/insert
	args := pgx.NamedArgs{
		"leagueID":    leagueID,
		"externalID":  manager.ExternalID,
		"teamName":    manager.TeamName,
		"managerName": manager.ManagerName,
		"joinKey":     manager.JoinKey,
	}

	var count int
	err = tx.QueryRow(ctx, query, args).Scan(&count)
	if err != nil {
		return fmt.Errorf("unexpected error getting league manager at start of save: %w", err)
	}

	queryToUse := update
	if count == 0 {
		queryToUse = insert
	} else if count > 1 {
		return fmt.Errorf("found multiple rows when 1 expected, leagueID: %d, externalID: %s", leagueID, manager.ExternalID)
	}

	// execute the insert or update
	tag, err := tx.Exec(ctx, queryToUse, args)
	if err != nil {
		return fmt.Errorf("error updating league manager: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("expected 1 league manager updated, got: %d", tag.RowsAffected())
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error commint leange manager transaction: %w", err)
	}
	return nil
}

func (db *postgresDB) AddLeague(ctx context.Context, league *model.League) error {
	const insertLeagueQuery = `INSERT INTO leagues(platform, external_id, name, year) 
		VALUES (@platform, @externalID, @name, @year) RETURNING id`

	args := pgx.NamedArgs{
		"platform":   league.Platform,
		"externalID": league.ExternalID,
		"name":       league.Name,
		"year":       league.Year,
	}

	err := db.pool.QueryRow(ctx, insertLeagueQuery, args).Scan(&league.ID)
	if err != nil {
		return fmt.Errorf("error inserting league: %w", err)
	}
	if league.ID <= 0 {
		return fmt.Errorf("did not get a valid league id, got: %d", league.ID)
	}

	return nil
}

func (db *postgresDB) ArchiveLeague(ctx context.Context, id int32) error {
	const archiveLeagueStmt = `UPDATE leagues SET archived=true WHERE id=@id`
	tag, err := db.pool.Exec(ctx, archiveLeagueStmt, pgx.NamedArgs{"id": id})
	if err != nil {
		return fmt.Errorf("error archiving league: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("expected 1 row to be affected, instead it was %d", tag.RowsAffected())
	}

	return nil
}

func (db *postgresDB) GetToken(ctx context.Context, leagueID int32) (*oauth2.Token, error) {
	const query = `SELECT access_token, refresh_token, expires FROM tokens WHERE league_id=@id`

	var t oauth2.Token
	var e pgtype.Timestamptz
	args := pgx.NamedArgs{
		"id": leagueID,
	}
	err := db.pool.QueryRow(ctx, query, args).Scan(&t.AccessToken, &t.RefreshToken, &e)
	if err != nil {
		return nil, fmt.Errorf("error getting token for league %d: %w", leagueID, err)
	}
	t.Expiry = e.Time.UTC()

	return &t, nil
}

func (db *postgresDB) SaveToken(ctx context.Context, leagueID int32, token *oauth2.Token) error {
	const stmt = `INSERT INTO tokens (league_id, access_token, refresh_token, expires)
			VALUES(@id, @accessToken, @refreshToken, @expires)
			ON CONFLICT(league_id) DO UPDATE SET
				access_token=EXCLUDED.access_token,
				refresh_token=EXCLUDED.refresh_token,
				expires=EXCLUDED.expires`

	args := pgx.NamedArgs{
		"id":           leagueID,
		"accessToken":  token.AccessToken,
		"refreshToken": token.RefreshToken,
		"expires": pgtype.Timestamptz{
			Time:             token.Expiry.UTC(),
			InfinityModifier: pgtype.Finite,
			Valid:            true,
		},
	}
	_, err := db.pool.Exec(ctx, stmt, args)
	if err != nil {
		return fmt.Errorf("error inserting/updating token %d: %w", leagueID, err)
	}
	return nil
}

func (db *postgresDB) SaveResults(ctx context.Context, leagueID int32, matchups []model.Matchup) error {
	const insert = `INSERT INTO team_results(league_id, week, match_id, team, score)
			VALUES(@leagueID, @week, @matchID, @team, @score)`
	const seq = `SELECT nextval('match_ids')`

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, m := range matchups {
		var matchID int32
		if err := tx.QueryRow(ctx, seq).Scan(&matchID); err != nil {
			return fmt.Errorf("error getting next sequence number: %w", err)
		}

		argsA := namedArgsForTeamResult(leagueID, matchID, m.Week, m.TeamA)
		if _, err := tx.Exec(ctx, insert, argsA); err != nil {
			return fmt.Errorf("error inserting teamA result: %w", err)
		}

		argsB := namedArgsForTeamResult(leagueID, matchID, m.Week, m.TeamB)
		if _, err := tx.Exec(ctx, insert, argsB); err != nil {
			return fmt.Errorf("error inserting teamB result: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error commiting transaction: %w", err)
	}

	return nil
}

func namedArgsForTeamResult(leagueID int32, matchID int32, week int, tr *model.TeamResult) pgx.NamedArgs {
	return pgx.NamedArgs{
		"leagueID": leagueID,
		"week":     week,
		"matchID":  matchID,
		"team":     tr.TeamID,
		"score":    tr.Score,
	}
}

func (db *postgresDB) GetResults(ctx context.Context, leagueID int32, week int) ([]model.Matchup, error) {
	const query = `SELECT 
					league_managers.team_name, 
					league_managers.manager_name,
					league_managers.external_id,
					team_results.match_id,
					team_results.score 
				FROM team_results INNER JOIN league_managers ON 
					(team_results.league_id=league_managers.league_id AND team_results.team=league_managers.external_id)
				WHERE team_results.league_id=@leagueID AND team_results.week=@week
				ORDER BY team_results.match_id`

	rows, err := db.pool.Query(ctx, query, pgx.NamedArgs{"leagueID": leagueID, "week": week})
	if err != nil {
		return nil, fmt.Errorf("error querying league results: %w", err)
	}

	resultMap := make(map[int32]*model.Matchup)
	for rows.Next() {
		var team, manager, id string
		var matchID, score int32
		if err := rows.Scan(&team, &manager, &id, &matchID, &score); err != nil {
			return nil, fmt.Errorf("error scanning team result: %w", err)
		}
		tr := &model.TeamResult{
			TeamID:   id,
			TeamName: first(team, manager),
			Score:    score,
		}

		if m, found := resultMap[matchID]; found {
			m.TeamB = tr
		} else {
			resultMap[matchID] = &model.Matchup{
				TeamA:     tr,
				MatchupID: matchID,
				Week:      week,
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading league results: %w", err)
	}

	results := make([]model.Matchup, 0, len(resultMap))
	for _, v := range resultMap {
		results = append(results, *v)
	}
	slices.SortFunc(results, func(a, b model.Matchup) int {
		return int(a.MatchupID - b.MatchupID)
	})
	return results, nil
}

func (db *postgresDB) ListResultWeeks(ctx context.Context, leagueID int32) ([]int, error) {
	const query = `SELECT DISTINCT(week) FROM team_results WHERE league_id=@id ORDER BY week`

	args := pgx.NamedArgs{
		"id": leagueID,
	}
	rows, err := db.pool.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("error querying team_results: %w", err)
	}

	results := make([]int, 0, 17)
	for rows.Next() {
		var i int
		if err := rows.Scan(&i); err != nil {
			return nil, fmt.Errorf("error scanning team_results row: %w", err)
		}
		results = append(results, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func (db *postgresDB) SavePowerRanking(ctx context.Context, leagueID int32, pr *model.PowerRanking) (int32, error) {
	const insertPRQuery = `INSERT INTO power_rankings (league_id, ranking_id, week) 
			VALUES (@leagueID, @rankingID, @week) RETURNING id`
	const insertTeamPowerRankingQuery = `INSERT INTO team_power_rankings (
				power_ranking_id,
				league_id,
				team,
				rank,
				rank_change,
				total_score,
				roster_score,
				record_score,
				streak_score,
				points_for_score,
				points_against_score
			) VALUES (
			 	@powerRankingID,
				@leagueID,
				@team,
				@rank,
				@rankChange,
				@totalScore,
				@rosterScore,
				@recordScore,
				@streakScore,
				@pointsForScore,
				@pointsAgainstScore
			)`
	const insertRosterQuery = `INSERT INTO power_rankings_rosters (
				power_ranking_id,
				league_id,
				team,
				player_id,
				nfl_team,
				player_rank,
				player_points,
				starter
			) VALUES (
			 	@powerRankingID,
				@leagueID,
				@team,
				@playerID,
				@nflTeam,
				@playerRank,
				@playerPoints,
				@starter
			)`

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	prArgs := pgx.NamedArgs{
		"leagueID":  leagueID,
		"rankingID": pr.RankingID,
		"week":      pr.Week,
	}
	err = tx.QueryRow(ctx, insertPRQuery, prArgs).Scan(&pr.ID)
	if err != nil {
		return 0, fmt.Errorf("error inserting power ranking: %w", err)
	}
	if pr.ID <= 0 {
		return 0, fmt.Errorf("did not get a valid ID for power ranking, got: %d", pr.ID)
	}

	for _, t := range pr.Teams {
		teamArgs := pgx.NamedArgs{
			"powerRankingID":     pr.ID,
			"leagueID":           leagueID,
			"team":               t.TeamID,
			"rank":               t.Rank,
			"rankChange":         t.RankChange,
			"totalScore":         t.TotalScore,
			"rosterScore":        t.RosterScore,
			"recordScore":        t.RecordScore,
			"streakScore":        t.StreakScore,
			"pointsForScore":     t.PointsForScore,
			"pointsAgainstScore": t.PointsAgainstScore,
		}
		if _, err := tx.Exec(ctx, insertTeamPowerRankingQuery, teamArgs); err != nil {
			return 0, fmt.Errorf("error inserting team %s into power rankings: %w", t.TeamID, err)
		}

		for _, p := range t.Roster {
			rosterArgs := pgx.NamedArgs{
				"powerRankingID": pr.ID,
				"leagueID":       leagueID,
				"team":           t.TeamID,
				"playerID":       p.PlayerID,
				"nflTeam":        &DBNFLTeam{team: p.NFLTeam},
				"playerRank":     p.Rank,
				"playerPoints":   p.PowerRankingPoints,
				"starter":        p.IsStarter,
			}
			if _, err := tx.Exec(ctx, insertRosterQuery, rosterArgs); err != nil {
				return 0, fmt.Errorf("error inserting player %s into power ranking rosters: %w", p.PlayerID, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("error commiting transaction: %w", err)
	}

	return pr.ID, nil
}

func (db *postgresDB) GetPowerRanking(ctx context.Context, leagueID, powerRankingID int32) (*model.PowerRanking, error) {
	const prQuery = `SELECT ranking_id, week, created FROM power_rankings WHERE id=@id AND league_id=@leagueID`

	pr := model.PowerRanking{
		ID: powerRankingID,
	}
	args := pgx.NamedArgs{
		"id":       powerRankingID,
		"leagueID": leagueID,
	}
	var created pgtype.Timestamptz
	if err := db.pool.QueryRow(ctx, prQuery, args).Scan(&pr.RankingID, &pr.Week, &created); err != nil {
		return nil, fmt.Errorf("error querying by power ranking id: %w", err)
	}
	pr.Created = created.Time

	if err := db.getPowerRankingTeams(ctx, &pr, leagueID); err != nil {
		return nil, err
	}

	return &pr, nil
}

func (db *postgresDB) ListPowerRankings(ctx context.Context, leagueID int32) ([]model.PowerRanking, error) {
	const query = `SELECT id, week, created FROM power_rankings WHERE league_id=@leagueID ORDER BY week DESC, created DESC`

	results := make([]model.PowerRanking, 0)

	args := pgx.NamedArgs{"leagueID": leagueID}
	rows, err := db.pool.Query(ctx, query, args)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("1 - no power rankings for league %d", leagueID)
			return results, nil
		}
		return nil, fmt.Errorf("error listing power rankings for league %d: %w", leagueID, err)
	}

	for rows.Next() {
		var pr model.PowerRanking
		var created pgtype.Timestamptz
		if err := rows.Scan(&pr.ID, &pr.Week, &created); err != nil {
			return nil, fmt.Errorf("error scanning power ranking for league %d: %w", leagueID, err)
		}
		pr.Created = created.Time

		results = append(results, pr)
	}
	if err := rows.Err(); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("2 - no power rankings for league %d", leagueID)
			return results, nil
		}
	}

	return results, nil
}

func (db *postgresDB) getPowerRankingTeams(ctx context.Context, pr *model.PowerRanking, leagueID int32) error {
	const teamQuery = `SELECT 
				t.team, m.team_name, m.manager_name, t.rank, 
				t.rank_change, t.total_score, t.roster_score, t.record_score,
				t.streak_score, t.points_for_score, t.points_against_score
			FROM team_power_rankings AS t INNER JOIN league_managers AS m 
				ON (t.team=m.external_id AND t.league_id=m.league_id) 
			WHERE t.power_ranking_id=@id AND t.league_id=@leagueID
			ORDER BY rank;`

	args := pgx.NamedArgs{
		"id":       pr.ID,
		"leagueID": leagueID,
	}
	rows, err := db.pool.Query(ctx, teamQuery, args)
	if err != nil {
		return fmt.Errorf("error getting team power rank results: %w", err)
	}
	for rows.Next() {
		t := model.TeamPowerRanking{
			Roster: make([]model.PowerRankingPlayer, 0, 15),
		}

		var teamName, managerName string
		err := rows.Scan(&t.TeamID, &teamName, &managerName, &t.Rank,
			&t.RankChange, &t.TotalScore, &t.RosterScore, &t.RecordScore,
			&t.StreakScore, &t.PointsForScore, &t.PointsAgainstScore)
		if err != nil {
			return fmt.Errorf("error scanning team result: %w", err)
		}
		t.TeamName = first(teamName, managerName)

		if err := db.getPowerRankingPlayers(ctx, &t, leagueID, pr.ID); err != nil {
			return err
		}

		pr.Teams = append(pr.Teams, t)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

func (db *postgresDB) getPowerRankingPlayers(ctx context.Context, t *model.TeamPowerRanking, leagueID, powerRankingID int32) error {
	const rosterQuery = `SELECT
				r.player_id, p.name_first, p.name_last, p.position,
				r.nfl_team, r.player_rank, r.player_points, r.starter
			FROM power_rankings_rosters AS r INNER JOIN players AS p
				ON (r.player_id=p.id)
			WHERE r.power_ranking_id=@id AND r.league_id=@leagueID AND r.team=@teamID 
			ORDER BY player_rank;`

	args := pgx.NamedArgs{
		"id":       powerRankingID,
		"leagueID": leagueID,
		"teamID":   t.TeamID,
	}
	rows, err := db.pool.Query(ctx, rosterQuery, args)
	if err != nil {
		return fmt.Errorf("error getting team roster power rank results: %w", err)
	}
	for rows.Next() {
		var p model.PowerRankingPlayer

		var pos DBPosition
		var nflTeam DBNFLTeam
		err := rows.Scan(&p.PlayerID, &p.FirstName, &p.LastName, &pos,
			&nflTeam, &p.Rank, &p.PowerRankingPoints, &p.IsStarter)
		if err != nil {
			return fmt.Errorf("error scanning team roster: %w", err)
		}
		p.NFLTeam = nflTeam.team
		p.Position = pos.position

		t.Roster = append(t.Roster, p)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

func (db *postgresDB) getPlayer(ctx context.Context, id string) (*model.Player, error) {
	const query = `SELECT id, yahoo_id, name_first, name_last, nickname1,
				  		position, team, weight_lb, height_in, birth_date,
						rookie_year, years_exp, jersey_num, depth_chart_order,
						college, active, created, updated
					FROM players WHERE id=@id`

	args := pgx.NamedArgs{
		"id": id,
	}
	row := db.pool.QueryRow(ctx, query, args)
	result, err := scanPlayer(row)
	if err != nil {
		return nil, fmt.Errorf("error scanning player %s: %w", id, err)
	}

	changes, err := db.getChangesByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error looking up player changes for %s: %w", id, err)
	}
	result.Changes = changes

	return result, nil
}

func scanRanking(row pgx.Row) (*model.Ranking, error) {
	var r model.Ranking
	var date pgtype.Timestamptz

	err := row.Scan(&r.ID, &date)
	if err != nil {
		return nil, fmt.Errorf("error scanning row: %w", err)
	}
	if !date.Valid {
		return nil, fmt.Errorf("ranking date is not valid: %w", err)
	}
	r.Date = date.Time.UTC()

	return &r, nil
}

func valueOrEmpty(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

type DBPosition struct {
	position model.Position
}

func (p *DBPosition) ScanText(v pgtype.Text) error {
	p.position = model.ParsePosition(v.String)
	return nil
}

func (p *DBPosition) TextValue() (pgtype.Text, error) {
	return pgtype.Text{
		String: string(p.position),
		Valid:  true,
	}, nil
}

type DBNFLTeam struct {
	team *model.NFLTeam
}

func (t *DBNFLTeam) ScanText(v pgtype.Text) error {
	t.team = model.ParseTeam(v.String)
	return nil
}

func (t *DBNFLTeam) TextValue() (pgtype.Text, error) {
	if t.team == nil {
		return pgtype.Text{
			String: "",
			Valid:  true,
		}, nil
	}
	return pgtype.Text{
		String: t.team.String(),
		Valid:  true,
	}, nil
}

func first(args ...string) string {
	for _, a := range args {
		if strings.TrimSpace(a) != "" {
			return a
		}
	}
	return ""
}
