CREATE TABLE IF NOT EXISTS players (
    -- id == the sleeper player id
    id                varchar(16) PRIMARY KEY,
    yahoo_id          varchar(16),
    name_first        varchar(64) NOT NULL,
    name_last         varchar(64) NOT NULL,
    nickname1         varchar(64),
    position          varchar(3),
    team              varchar(3),
    weight_lb         smallint,
    height_in         smallint,
    birth_date        date,
    rookie_year       date,
    years_exp         smallint,
    jersey_num        smallint,
    depth_chart_order smallint,
    college           varchar(64),
    active            boolean,
    created           timestamp with time zone DEFAULT (now() at time zone 'utc'),
    updated           timestamp with time zone,
    fts_player        tsvector GENERATED ALWAYS AS (to_tsvector(
        'english', name_first || ' ' || name_last || coalesce(nickname1, '')
    )) STORED
);

CREATE TABLE IF NOT EXISTS player_changes (
    id      bigserial PRIMARY KEY,
    player  varchar(16) references players(id),
    created timestamp DEFAULT (now() at time zone 'utc'),
    prop    varchar(32) NOT NULL,
    old     text NOT NULL,
    new     text NOT NULL
);

CREATE INDEX IF NOT EXISTS player_name_idx ON players USING gin(fts_player);
CREATE INDEX IF NOT EXISTS player_yahoo_id_idx ON players(yahoo_id);
CREATE INDEX IF NOT EXISTS player_change_idx ON player_changes(player, created DESC);
