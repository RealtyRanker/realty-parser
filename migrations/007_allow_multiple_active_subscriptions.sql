-- Allow a single chat_id to have more than one active subscription at a time
-- (e.g. subscriptions for different regions/price ranges). Cancellation now
-- targets a specific subscription id instead of "all active for this chat".

DROP INDEX IF EXISTS user_subscriptions_chat_id_active_uidx;

CREATE INDEX IF NOT EXISTS idx_user_subscriptions_chat_id_active
    ON user_subscriptions (chat_id)
    WHERE is_active = TRUE;
