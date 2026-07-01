ALTER TABLE user_subscriptions
    ADD COLUMN region INTEGER NOT NULL DEFAULT 1;

CREATE INDEX IF NOT EXISTS idx_user_subscriptions_region ON user_subscriptions (region);
