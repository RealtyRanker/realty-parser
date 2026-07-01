-- Per-subscription custom scoring weights, overriding the 18 customizable
-- multipliers in the default rent/sale scoring formula. A subscription has
-- at most one row here; absence means "use default scoring".
CREATE TABLE IF NOT EXISTS subscription_scoring_params (
    id                           SERIAL PRIMARY KEY,
    subscription_id              INTEGER NOT NULL UNIQUE
                                  REFERENCES user_subscriptions (id) ON DELETE CASCADE,

    all_area_multiplier          DOUBLE PRECISION NOT NULL,
    kitchen_area_multiplier      DOUBLE PRECISION NOT NULL,
    pets_multiplier              DOUBLE PRECISION NOT NULL,
    apartments_multiplier        DOUBLE PRECISION NOT NULL,
    two_room_multiplier          DOUBLE PRECISION NOT NULL,
    three_room_multiplier        DOUBLE PRECISION NOT NULL,
    four_room_multiplier         DOUBLE PRECISION NOT NULL,
    additional_rooms_multiplier  DOUBLE PRECISION NOT NULL,
    windows_yard_multiplier      DOUBLE PRECISION NOT NULL,
    windows_street_multiplier    DOUBLE PRECISION NOT NULL,
    windows_both_multiplier      DOUBLE PRECISION NOT NULL,
    renovation_design_mult       DOUBLE PRECISION NOT NULL,
    renovation_euro_mult         DOUBLE PRECISION NOT NULL,
    renovation_cosmetic_mult     DOUBLE PRECISION NOT NULL,
    bathroom_separated_mult      DOUBLE PRECISION NOT NULL,
    balcony_multiplier           DOUBLE PRECISION NOT NULL,
    loggia_multiplier            DOUBLE PRECISION NOT NULL,
    underground_score_mult       DOUBLE PRECISION NOT NULL
);
