-- Extended (optional) subscription filters, offered as a second tier during
-- subscription creation on top of the basic price/area/rooms/score filters.
ALTER TABLE user_subscriptions
    ADD COLUMN min_underground_place INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN min_kitchen_area DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN min_floor INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN max_floor INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN min_ceiling_height DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN children_required BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN pets_required BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN dishwasher_required BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN conditioner_required BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN min_renovation TEXT NOT NULL DEFAULT ''
        CHECK (min_renovation IN ('', 'cosmetic', 'euro', 'design')),
    ADD COLUMN balcony_required BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN bathroom_type TEXT NOT NULL DEFAULT ''
        CHECK (bathroom_type IN ('', 'separated', 'combined'));
