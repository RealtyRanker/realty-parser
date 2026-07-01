-- Add the two amenity multipliers (dishwasher, conditioner) that were
-- missing from the initial custom-scoring question set.
ALTER TABLE subscription_scoring_params
    ADD COLUMN dishwasher_multiplier DOUBLE PRECISION NOT NULL DEFAULT 1500,
    ADD COLUMN conditioner_multiplier DOUBLE PRECISION NOT NULL DEFAULT 1500;

ALTER TABLE subscription_scoring_params
    ALTER COLUMN dishwasher_multiplier DROP DEFAULT,
    ALTER COLUMN conditioner_multiplier DROP DEFAULT;
