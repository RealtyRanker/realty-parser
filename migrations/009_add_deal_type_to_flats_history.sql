-- Track whether a parsed flat is a rental or sale listing, mirroring
-- user_subscriptions.deal_type, so matching can distinguish the two.
ALTER TABLE flats_history
    ADD COLUMN deal_type TEXT NOT NULL DEFAULT 'rent'
        CHECK (deal_type IN ('rent', 'sale'));

CREATE INDEX IF NOT EXISTS idx_flats_history_deal_type ON flats_history (deal_type);
