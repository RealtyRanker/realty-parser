-- Subscriptions for periodic CSV reports (as opposed to per-flat instant
-- notifications in user_subscriptions). Mirrors the same filter columns,
-- plus a send period and a timestamp of when the last report was sent.
CREATE TABLE IF NOT EXISTS report_user_subscriptions (
    id                     SERIAL PRIMARY KEY,
    chat_id                BIGINT NOT NULL,
    created_at             TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    deal_type              TEXT NOT NULL DEFAULT 'rent'
                           CHECK (deal_type IN ('rent', 'sale')),
    region                 INTEGER NOT NULL DEFAULT 1,

    min_price              INTEGER NOT NULL DEFAULT 0,
    max_price              INTEGER NOT NULL DEFAULT 0,
    min_area               DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_area               DOUBLE PRECISION NOT NULL DEFAULT 0,
    rooms                  INTEGER[] NOT NULL DEFAULT '{}',
    min_score              INTEGER NOT NULL DEFAULT 0,

    min_underground_place  INTEGER NOT NULL DEFAULT 0,
    min_kitchen_area       DOUBLE PRECISION NOT NULL DEFAULT 0,
    min_floor              INTEGER NOT NULL DEFAULT 0,
    max_floor              INTEGER NOT NULL DEFAULT 0,
    min_ceiling_height     DOUBLE PRECISION NOT NULL DEFAULT 0,
    children_required      BOOLEAN NOT NULL DEFAULT FALSE,
    pets_required          BOOLEAN NOT NULL DEFAULT FALSE,
    dishwasher_required    BOOLEAN NOT NULL DEFAULT FALSE,
    conditioner_required   BOOLEAN NOT NULL DEFAULT FALSE,
    min_renovation         TEXT NOT NULL DEFAULT ''
                           CHECK (min_renovation IN ('', 'cosmetic', 'euro', 'design')),
    balcony_required       BOOLEAN NOT NULL DEFAULT FALSE,
    bathroom_type          TEXT NOT NULL DEFAULT ''
                           CHECK (bathroom_type IN ('', 'separated', 'combined')),

    is_active              BOOLEAN NOT NULL DEFAULT TRUE,

    -- One of: 300 (5m), 3600 (1h), 43200 (12h), 86400 (24h), 604800 (7d), 2592000 (30d).
    period_seconds         INTEGER NOT NULL
                           CHECK (period_seconds IN (300, 3600, 43200, 86400, 604800, 2592000)),
    last_report_sent_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_report_user_subscriptions_chat_id        ON report_user_subscriptions (chat_id);
CREATE INDEX IF NOT EXISTS idx_report_user_subscriptions_chat_id_active ON report_user_subscriptions (chat_id) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_report_user_subscriptions_due            ON report_user_subscriptions (is_active, last_report_sent_at);
