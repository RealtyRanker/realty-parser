-- Replace the column-level UNIQUE constraint on chat_id with a partial unique
-- index that allows multiple historical (inactive) rows per user but still
-- enforces at most one active subscription per chat_id.

ALTER TABLE user_subscriptions DROP CONSTRAINT user_subscriptions_chat_id_key;

CREATE UNIQUE INDEX user_subscriptions_chat_id_active_uidx
    ON user_subscriptions (chat_id)
    WHERE is_active = TRUE;
