package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/itbasis/go-clock"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mww/fantasy_manager_v2/model"
)

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

	return db.updatePlayer(ctx, old, p)
}

func (db *postgresDB) DeletePlayerNickname(ctx context.Context, playerID string, oldNickname string) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := savePlayerNickname(ctx, tx, playerID, ""); err != nil {
		return err
	}

	change := model.Change{
		Time:         db.clock.Now().UTC(),
		PropertyName: "Nickname1",
		OldValue:     oldNickname,
		NewValue:     "",
	}
	if err := insertPlayerChange(ctx, tx, playerID, &change); err != nil {
		return err
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

func (db *postgresDB) ConvertYahooPlayerIDs(ctx context.Context, players []model.YahooPlayer) ([]string, error) {
	results := make([]string, 0, len(players))
	for _, p := range players {
		id, err := db.findByYahooID(ctx, p.YahooID)
		if errors.Is(err, pgx.ErrTooManyRows) {
			return nil, fmt.Errorf("multiple results found for yahoo_id: %s", p.YahooID)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			id, err = db.findByPlayerName(ctx, &p)
		}
		if err != nil {
			return nil, err
		}

		results = append(results, id)
	}

	return results, nil
}

func (db *postgresDB) findByYahooID(ctx context.Context, yahooID string) (string, error) {
	const idQuery = `SELECT id FROM players WHERE yahoo_id=@yahooID`

	rows, err := db.pool.Query(ctx, idQuery, pgx.NamedArgs{"yahooID": yahooID})
	if err != nil {
		return "", fmt.Errorf("error querying player with yahoo_id=%s: %w", yahooID, err)
	}

	return pgx.CollectExactlyOneRow(rows, func(row pgx.CollectableRow) (string, error) {
		var id string
		err := row.Scan(&id)
		return id, err
	})
}

func (db *postgresDB) findByPlayerName(ctx context.Context, p *model.YahooPlayer) (string, error) {
	yahooDetails := fmt.Sprintf("%s - %s %s %v", p.YahooID, p.FirstName, p.LastName, p.Pos)

	fullName := model.TrimNameSuffix(fmt.Sprintf("%s %s", p.FirstName, p.LastName))
	results, err := db.Search(ctx, fullName, p.Pos, nil)
	if err != nil {
		return "", fmt.Errorf("error searching for %s: %w", yahooDetails, err)
	}
	if len(results) == 0 {
		return "", fmt.Errorf("no results found for %s", yahooDetails)
	}
	if len(results) > 1 {
		return "", fmt.Errorf("multiple results found for %s", yahooDetails)
	}

	f := results[0]
	if err := db.savePlayerYahooID(ctx, f.ID, p.YahooID); err != nil {
		return "", err
	}
	log.Printf("match found for yahoo id %s, sleeper id %s - %s %s", yahooDetails, f.ID, f.FirstName, f.LastName)
	return f.ID, nil
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
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("error inserting player(%s): %w", p.ID, err)
	}

	if p.Nickname1 != "" {
		if err := savePlayerNickname(ctx, tx, p.ID, p.Nickname1); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error commiting transaction: %w", err)
	}

	return nil
}

func (db *postgresDB) updatePlayer(ctx context.Context, old, new *model.Player) error {
	const update = `UPDATE players
		SET name_first=@nameFirst,
			name_last=@nameLast,
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

	changes, err := db.calculateChanges(old, new)
	if err != nil {
		return fmt.Errorf("error calculating changes: %w", err)
	}

	// Don't delete the nickname just because it is empty. Sleeper doesn't have nicknames
	// so player updates will always be empty.
	if new.Nickname1 != "" && new.Nickname1 != old.Nickname1 {
		change := model.Change{
			Time:         db.clock.Now().UTC(),
			PropertyName: "Nickname1",
			OldValue:     old.Nickname1,
			NewValue:     new.Nickname1,
		}
		changes = append(changes, change)

		if err := savePlayerNickname(ctx, tx, new.ID, new.Nickname1); err != nil {
			return err
		}
	}

	if len(changes) == 0 {
		// There are no changes for the player
		return nil
	}

	args := namedArgsForPlayer(new, db.clock)
	_, err = tx.Exec(ctx, update, args)
	if err != nil {
		return fmt.Errorf("error updating player (%s): %w", new.ID, err)
	}

	for _, change := range changes {
		err := insertPlayerChange(ctx, tx, new.ID, &change)
		if err != nil {
			return fmt.Errorf("error inserting player change: %w", err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("error commiting player transaction: %w", err)
	}

	new.Changes = append(new.Changes, changes...)
	slices.SortFunc(new.Changes, func(a, b model.Change) int {
		return b.Time.Compare(a.Time)
	})

	return nil
}

func savePlayerNickname(ctx context.Context, tx pgx.Tx, id string, nickname string) error {
	const query = `UPDATE players SET nickname1=@nickname1 WHERE id=@id`

	args := pgx.NamedArgs{
		"id": id,
		"nickname1": sql.NullString{
			String: nickname,
			Valid:  nickname != "",
		},
	}
	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("error setting player nickname (%s): %w", id, err)
	}

	return nil
}

func (db *postgresDB) savePlayerYahooID(ctx context.Context, playerID string, yahooID string) error {
	const query = `UPDATE players SET yahoo_id=@yahooID WHERE id=@playerID`

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	args := pgx.NamedArgs{
		"playerID": playerID,
		"yahooID": sql.NullString{
			String: yahooID,
			Valid:  yahooID != "",
		},
	}
	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("error setting player yahooID (%s): %w", playerID, err)
	}

	change := model.Change{
		Time:         db.clock.Now().UTC(),
		PropertyName: "YahooID",
		OldValue:     "",
		NewValue:     yahooID,
	}
	if err := insertPlayerChange(ctx, tx, playerID, &change); err != nil {
		return fmt.Errorf("error inserting player change for updated yahoo id: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error commiting transaction: %w", err)
	}

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

	// Don't look for changes in the following properties. They can be updated outside
	// of sleeper, so we don't want player refreshes from sleeper to wipe out our
	// changes.
	// - Nickname1
	// - YahooID

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
