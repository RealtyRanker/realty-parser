CREATE TABLE IF NOT EXISTS user_subscriptions (
    id           SERIAL PRIMARY KEY,
    chat_id      BIGINT NOT NULL UNIQUE,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    min_price    INTEGER NOT NULL DEFAULT 0,
    max_price    INTEGER NOT NULL DEFAULT 0,
    min_area     DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_area     DOUBLE PRECISION NOT NULL DEFAULT 0,
    rooms        INTEGER[] NOT NULL DEFAULT '{}',
    min_score    INTEGER NOT NULL DEFAULT 0,

    is_active    BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_user_subscriptions_chat_id   ON user_subscriptions (chat_id);
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_is_active ON user_subscriptions (is_active);
