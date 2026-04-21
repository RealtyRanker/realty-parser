CREATE TABLE IF NOT EXISTS user_sent_messages (
    id                  SERIAL PRIMARY KEY,
    subscription_id     INTEGER NOT NULL REFERENCES user_subscriptions (id) ON DELETE CASCADE,
    flat_id             INTEGER NOT NULL REFERENCES flats_history (id) ON DELETE CASCADE,
    sent_at             TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE (subscription_id, flat_id)
);

CREATE INDEX IF NOT EXISTS idx_user_sent_messages_subscription_id ON user_sent_messages (subscription_id);
CREATE INDEX IF NOT EXISTS idx_user_sent_messages_flat_id         ON user_sent_messages (flat_id);
