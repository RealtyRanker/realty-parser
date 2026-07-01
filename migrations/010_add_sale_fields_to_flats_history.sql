-- Sale listings carry several fields rentals don't: whether a mortgage is
-- possible, whether it's a "free" or "alternative" sale (альтернативная
-- продажа is slower/riskier — depends on the seller buying their own next
-- home first), whether the listing is posted directly by the owner, and
-- whether it's a new-building (novostroyka) unit vs secondary market.
ALTER TABLE flats_history
    ADD COLUMN sale_type TEXT NOT NULL DEFAULT '',
    ADD COLUMN mortgage_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN is_new_building BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN new_building_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN is_by_homeowner BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN demolished_in_moscow_program BOOLEAN NOT NULL DEFAULT FALSE;
