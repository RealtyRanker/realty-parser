ALTER TABLE flats_history
    ADD COLUMN region INTEGER NOT NULL DEFAULT 1;

CREATE INDEX IF NOT EXISTS idx_flats_history_region ON flats_history (region);
