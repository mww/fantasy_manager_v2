package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

func (db *postgresDB) GetPlayer(ctx context.Context, id string) (*model.Player, error) {
	p, err := db.getPlayer(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPlayerNotFound
		}
		return nil, err
	}
	return p, nil
}

func (db *postgresDB) SavePlayer(ctx context.Context, p *model.Player) error {
	old, err := db.getPlayer(ctx, p.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// This is an insert
			err := db.insertPlayer(ctx, p)
			if err != nil {
				return fmt.Errorf("error inserting player: %w", err)
			}
			return nil
		}

		return fmt.Errorf("error reading player at start of SavePlayer(): %w", err)
	}

	// This is an update, see what, if anything changed
	changes, err := db.calculateChanges(old, p)
	if err != nil {
		return fmt.Errorf("error calculating changes: %w", err)
	}
	if len(changes) > 0 {
		return db.updatePlayer(ctx, p, changes)
	}
	return nil
}

func (db *postgresDB) DeleteNickname(ctx context.Context, id, oldNickname string) error {
	const query = `UPDATE players SET nickname1=NULL WHERE id=@id`
	args := pgx.NamedArgs{
		"id": id,
	}

	change := &model.Change{
		Time:         db.clock.Now().UTC(),
		PropertyName: "Nickname1",
		OldValue:     oldNickname,
		NewValue:     "",
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("error deleteing player nickname (%s): %w", id, err)
	}
	if err := insertPlayerChange(ctx, tx, id, change); err != nil {
		return fmt.Errorf("error inserting player change (%s): %w", id, err)
	}

	return tx.Commit(ctx)
}

func (db *postgresDB) Search(ctx context.Context, q string, pos model.Position, team *model.NFLTeam) ([]model.Player, error) {
	const query = `SELECT id, yahoo_id, name_first, name_last, nickname1,
				  		position, team, weight_lb, height_in, birth_date,
						rookie_year, years_exp, jersey_num, depth_chart_order,
						college, active, created, updated
					FROM players WHERE fts_player @@ websearch_to_tsquery(@q)
						AND team ILIKE @team
						AND position ILIKE @pos`

	const teamAndPosQuery = `SELECT id, yahoo_id, name_first, name_last, nickname1,
					    		position, team, weight_lb, height_in, birth_date,
					  			rookie_year, years_exp, jersey_num, depth_chart_order,
					  			college, active, created, updated
				  			FROM players WHERE team ILIKE @team AND position ILIKE @pos`

	teamQ := "%"
	if team != nil {
		teamQ = team.String()
	}
	posQ := "%"
	if pos != model.POS_UNKNOWN {
		posQ = string(pos)
	}

	args := pgx.NamedArgs{
		"q":    q,
		"team": teamQ,
		"pos":  posQ,
	}

	qq := query
	if q == "" {
		qq = teamAndPosQuery
	}
	rows, err := db.pool.Query(ctx, qq, args)
	if err != nil {
		return nil, fmt.Errorf("error running search query: %w", err)
	}

	results := make([]model.Player, 0, 8)
	for rows.Next() {
		p, err := scanPlayer(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *p)
	}

	return results, nil
}

func (db *postgresDB) SavePlayerScores(ctx context.Context, leagueID int32, week int, scores []model.PlayerScore) error {
	const insert = `INSERT INTO player_scores(player_id, league_id, week, score) 
			VALUES (@playerID, @leagueID, @week, @score)`

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, s := range scores {
		args := pgx.NamedArgs{
			"playerID": s.PlayerID,
			"leagueID": leagueID,
			"week":     week,
			"score":    s.Score,
		}
		if _, err := tx.Exec(ctx, insert, args); err != nil {
			return fmt.Errorf("error inserting score for %s in league %d: %w", s.PlayerID, leagueID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error commiting player scores: %w", err)
	}

	return nil
}

func (db *postgresDB) GetPlayerScores(ctx context.Context, playerID string) ([]model.SeasonScores, error) {
	const query = `SELECT p.league_id, p.week, p.score, l.name, l.year FROM player_scores AS p
            INNER JOIN leagues AS l ON p.league_id=l.id
			WHERE p.player_id=@playerID ORDER BY p.league_id, p.week`

	rows, err := db.pool.Query(ctx, query, pgx.NamedArgs{"playerID": playerID})
	if err != nil {
		return nil, fmt.Errorf("error querying player scores: %w", err)
	}

	scoreMap := make(map[int32]*model.SeasonScores)
	for rows.Next() {
		var leagueID, score int32
		var week int
		var leagueName, leagueYear string
		if err := rows.Scan(&leagueID, &week, &score, &leagueName, &leagueYear); err != nil {
			return nil, fmt.Errorf("error scanning score: %w", err)
		}

		if scores, found := scoreMap[leagueID]; found {
			scores.Scores[week] = score
		} else {
			s := &model.SeasonScores{
				LeagueID:   leagueID,
				LeagueName: leagueName,
				LeagueYear: leagueYear,
				PlayerID:   playerID,
				Scores:     make([]int32, 19),
			}
			s.Scores[week] = score
			scoreMap[leagueID] = s
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error with rows: %w", err)
	}

	results := make([]model.SeasonScores, 0, len(scoreMap))
	for _, s := range scoreMap {
		results = append(results, *s)
	}
	slices.SortFunc(results, func(a, b model.SeasonScores) int {
		return strings.Compare(a.LeagueYear, b.LeagueYear)
	})

	return results, nil
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
	const prQuery = `SELECT ranking_id, week FROM power_rankings WHERE id=@id AND league_id=@leagueID`

	pr := model.PowerRanking{
		ID: powerRankingID,
	}
	args := pgx.NamedArgs{
		"id":       powerRankingID,
		"leagueID": leagueID,
	}
	if err := db.pool.QueryRow(ctx, prQuery, args).Scan(&pr.RankingID, &pr.Week); err != nil {
		return nil, fmt.Errorf("error querying by power ranking id: %w", err)
	}

	if err := db.getPowerRankingTeams(ctx, &pr, leagueID); err != nil {
		return nil, err
	}

	return &pr, nil
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

func scanPlayer(row pgx.Row) (*model.Player, error) {
	var result model.Player
	var pos DBPosition
	var team DBNFLTeam
	var yahooID, nickname1, college sql.NullString
	var birthDate, rookieYear pgtype.Date
	var created, updated pgtype.Timestamptz
	err := row.Scan(
		&result.ID,
		&yahooID,
		&result.FirstName,
		&result.LastName,
		&nickname1,
		&pos,
		&team,
		&result.Weight,
		&result.Height,
		&birthDate,
		&rookieYear,
		&result.YearsExp,
		&result.Jersey,
		&result.DepthChartOrder,
		&college,
		&result.Active,
		&created,
		&updated)

	if err != nil {
		return nil, err
	}

	result.Position = pos.position
	result.Team = team.team
	result.YahooID = valueOrEmpty(yahooID)
	result.Nickname1 = valueOrEmpty(nickname1)
	result.College = valueOrEmpty(college)
	result.BirthDate = birthDate.Time
	result.RookieYear = rookieYear.Time
	result.Created = created.Time
	result.Updated = updated.Time

	return &result, nil
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

func (db *postgresDB) getChangesByID(ctx context.Context, id string) ([]model.Change, error) {
	const query = `SELECT created, prop, old, new FROM player_changes WHERE player=@id ORDER BY created DESC`

	args := pgx.NamedArgs{
		"id": id,
	}
	rows, err := db.pool.Query(ctx, query, args)
	if err != nil {
		return nil, err
	}
	changes := make([]model.Change, 0, 16)
	for rows.Next() {
		var created pgtype.Timestamptz
		c := model.Change{}
		err := rows.Scan(&created, &c.PropertyName, &c.OldValue, &c.NewValue)
		if err != nil {
			return nil, fmt.Errorf("error scanning player change: %v", err)
		}
		c.Time = created.Time

		changes = append(changes, c)
	}

	return changes, nil
}

func (db *postgresDB) insertPlayer(ctx context.Context, p *model.Player) error {
	if p == nil {
		return errors.New("insertPlayer - player is nil")
	}
	const query = `INSERT INTO players (
		id,
		yahoo_id,
		name_first,
		name_last,
		nickname1,
		position,
		team,
		weight_lb,
		height_in,
		birth_date,
		rookie_year,
		years_exp,
		jersey_num,
		depth_chart_order,
		college,
		active
	) VALUES (
		@id,
		@yahooID,
		@nameFirst,
		@nameLast,
		@nickname1,
		@position,
		@team,
		@weight,
		@height,
		@birthDate,
		@rookieYear,
		@yearsExp,
		@jerseyNum,
		@depthChartOrder,
		@college,
		@active
	)`

	args := namedArgsForPlayer(p, db.clock)
	_, err := db.pool.Exec(ctx, query, args)
	if err != nil {
		return fmt.Errorf("error inserting player(%s): %w", p.ID, err)
	}
	return nil
}

func (db *postgresDB) updatePlayer(ctx context.Context, p *model.Player, changes []model.Change) error {
	const update = `UPDATE players
		SET yahoo_id=@yahooID,
			name_first=@nameFirst,
			name_last=@nameLast,
			nickname1=@nickname1,
			position=@position,
			team=@team,
			weight_lb=@weight,
			height_in=@height,
			birth_date=@birthDate,
			rookie_year=@rookieYear,
			years_exp=@yearsExp,
			jersey_num=@jerseyNum,
			depth_chart_order=@depthChartOrder,
			college=@college,
			active=@active,
			updated=@updated
		WHERE id=@id`

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	args := namedArgsForPlayer(p, db.clock)
	_, err = tx.Exec(ctx, update, args)
	if err != nil {
		return fmt.Errorf("error updating player (%s): %w", p.ID, err)
	}

	for _, change := range changes {
		err := insertPlayerChange(ctx, tx, p.ID, &change)
		if err != nil {
			return fmt.Errorf("error inserting player change: %w", err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("error commiting player transaction: %w", err)
	}

	p.Changes = append(p.Changes, changes...)
	slices.SortFunc(p.Changes, func(a, b model.Change) int {
		return b.Time.Compare(a.Time)
	})

	return nil
}

func insertPlayerChange(ctx context.Context, tx pgx.Tx, id string, change *model.Change) error {
	const insertChange = `INSERT INTO player_changes(
		player,
		prop,
		old,
		new
	) VALUES (
		@playerId,
		@prop,
		@old,
		@new
	)`

	args := pgx.NamedArgs{
		"playerId": id,
		"prop":     change.PropertyName,
		"old":      change.OldValue,
		"new":      change.NewValue,
	}
	_, err := tx.Exec(ctx, insertChange, args)
	return err
}

func (db *postgresDB) calculateChanges(old, new *model.Player) ([]model.Change, error) {
	changes := make([]model.Change, 0, 1)

	changes = checkChange(changes, db.clock, "YahooId", old.YahooID, new.YahooID)
	changes = checkChange(changes, db.clock, "FirstName", old.FirstName, new.FirstName)
	changes = checkChange(changes, db.clock, "LastName", old.LastName, new.LastName)
	changes = checkChange(changes, db.clock, "Position", string(old.Position), string(new.Position))
	changes = checkChange(changes, db.clock, "Team", old.Team.String(), new.Team.String())
	changes = checkChangeInt(changes, db.clock, "Weight", old.Weight, new.Weight)
	changes = checkChangeInt(changes, db.clock, "Height", old.Height, new.Height)
	changes = checkChange(changes, db.clock, "BirthDate", old.FormattedBirthDate(), new.FormattedBirthDate())
	changes = checkChange(changes, db.clock, "RookieYear", old.FormattedRookieYear(), new.FormattedRookieYear())
	changes = checkChangeInt(changes, db.clock, "YearsExp", old.YearsExp, new.YearsExp)
	changes = checkChangeInt(changes, db.clock, "Jersey", old.Jersey, new.Jersey)
	changes = checkChangeInt(changes, db.clock, "DepthChartOrder", old.DepthChartOrder, new.DepthChartOrder)
	changes = checkChange(changes, db.clock, "College", old.College, new.College)
	changes = checkChange(changes, db.clock, "Active", fmt.Sprintf("%v", old.Active), fmt.Sprintf("%v", new.Active))

	// Nickname is special because it isn't part of the sleeper data, therefore
	// only check for a change if the new data sets a nickname.
	// To delete a nickname use db.DeleteNickname().
	if new.Nickname1 != "" {
		changes = checkChange(changes, db.clock, "Nickname1", old.Nickname1, new.Nickname1)
	}
	return changes, nil
}

func checkChange(changes []model.Change, clock clock.Clock, prop, old, new string) []model.Change {
	if old != new {
		c := model.Change{
			Time:         clock.Now().UTC(),
			PropertyName: prop,
			OldValue:     old,
			NewValue:     new,
		}
		changes = append(changes, c)
	}
	return changes
}

func checkChangeInt(changes []model.Change, clock clock.Clock, prop string, old, new int) []model.Change {
	return checkChange(changes, clock, prop, fmt.Sprintf("%d", old), fmt.Sprintf("%d", new))
}

func namedArgsForPlayer(p *model.Player, clock clock.Clock) pgx.NamedArgs {
	return pgx.NamedArgs{
		"id": p.ID,
		"yahooID": sql.NullString{
			String: p.YahooID,
			Valid:  p.YahooID != "",
		},
		"nameFirst": p.FirstName,
		"nameLast":  p.LastName,
		"nickname1": sql.NullString{
			String: p.Nickname1,
			Valid:  p.Nickname1 != "",
		},
		"position": &DBPosition{position: p.Position},
		"team":     &DBNFLTeam{team: p.Team},
		"weight":   p.Weight,
		"height":   p.Height,
		"birthDate": pgtype.Date{
			Time:  p.BirthDate,
			Valid: !p.BirthDate.IsZero(),
		},
		"rookieYear": pgtype.Date{
			Time:  p.RookieYear,
			Valid: !p.RookieYear.IsZero(),
		},
		"yearsExp":        p.YearsExp,
		"jerseyNum":       p.Jersey,
		"depthChartOrder": p.DepthChartOrder,
		"college": sql.NullString{
			String: p.College,
			Valid:  p.College != "",
		},
		"active": p.Active,
		"updated": pgtype.Timestamptz{
			Time:             clock.Now().UTC(),
			InfinityModifier: pgtype.Finite,
			Valid:            true,
		},
	}
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
