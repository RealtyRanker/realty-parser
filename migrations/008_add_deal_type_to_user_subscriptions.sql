-- Track whether a subscription is for rental or sale listings. All flats
-- parsed so far are rentals; sale support is being added incrementally.
ALTER TABLE user_subscriptions
    ADD COLUMN deal_type TEXT NOT NULL DEFAULT 'rent'
        CHECK (deal_type IN ('rent', 'sale'));

CREATE INDEX IF NOT EXISTS idx_user_subscriptions_deal_type ON user_subscriptions (deal_type);
