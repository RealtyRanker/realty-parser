CREATE TABLE IF NOT EXISTS flats_history (
    id                        SERIAL PRIMARY KEY,
    link                      TEXT NOT NULL UNIQUE,
    parsed_at                 TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    price                     INTEGER NOT NULL DEFAULT 0,
    flat_score                INTEGER NOT NULL DEFAULT 0,

    underground_score         DOUBLE PRECISION NOT NULL DEFAULT 0,
    underground_place         INTEGER NOT NULL DEFAULT 0,
    underground_distance_info TEXT NOT NULL DEFAULT '',

    room_number               INTEGER NOT NULL DEFAULT 0,
    total_area                DOUBLE PRECISION NOT NULL DEFAULT 0,
    living_area               DOUBLE PRECISION NOT NULL DEFAULT 0,
    kitchen_area              DOUBLE PRECISION NOT NULL DEFAULT 0,

    floor                     INTEGER NOT NULL DEFAULT 0,
    max_floor                 INTEGER NOT NULL DEFAULT 0,

    deposit                   INTEGER NOT NULL DEFAULT 0,
    deposit_months            INTEGER NOT NULL DEFAULT 0,
    comission                 INTEGER NOT NULL DEFAULT 0,

    renovation                TEXT NOT NULL DEFAULT '',
    is_apartments             BOOLEAN NOT NULL DEFAULT FALSE,

    loggia_count              INTEGER NOT NULL DEFAULT 0,
    balcony_count             INTEGER NOT NULL DEFAULT 0,
    windows_view              TEXT NOT NULL DEFAULT '',

    separated_bathroom_count  INTEGER NOT NULL DEFAULT 0,
    combined_bathroom_count   INTEGER NOT NULL DEFAULT 0,

    has_dishwasher            BOOLEAN NOT NULL DEFAULT FALSE,
    has_conditioner           BOOLEAN NOT NULL DEFAULT FALSE,
    children_allowed          BOOLEAN NOT NULL DEFAULT FALSE,
    pets_allowed              BOOLEAN NOT NULL DEFAULT FALSE,

    last_updated              TEXT NOT NULL DEFAULT '',
    ceiling_height            DOUBLE PRECISION NOT NULL DEFAULT 0,

    building_entrances_number  INTEGER NOT NULL DEFAULT 0,
    building_apartments_number INTEGER NOT NULL DEFAULT 0,
    building_elevators_number  INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_flats_history_parsed_at  ON flats_history (parsed_at DESC);
CREATE INDEX IF NOT EXISTS idx_flats_history_flat_score ON flats_history (flat_score DESC);
