package db

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/asmisnik/realty-parser/internal/model"
	"github.com/asmisnik/realty-parser/internal/sqlite"
)

func TestListingAndEventAreAtomicAndDurable(t *testing.T) {
	database, dsn := testDB(t)
	ctx := context.Background()
	flat := &model.FlatInfo{Link: "https://example.test/flat/1", Region: 1, DealType: "rent",
		Price: 90000, FlatScore: 1500000, UndergroundStations: []string{"Алма-Атинская", "Орехово"},
		PhotoURLs: []string{"https://example.test/full.jpg"}, PhotoThumbnailURLs: []string{"https://example.test/thumb.jpg"}, HasDishwasher: true}
	inserted, err := database.InsertFlatAndEnqueue(ctx, flat)
	if err != nil || !inserted {
		t.Fatalf("insert: %v, %v", inserted, err)
	}
	if inserted, err = database.InsertFlatAndEnqueue(ctx, flat); err != nil || inserted {
		t.Fatalf("duplicate: %v, %v", inserted, err)
	}
	other, err := New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	var payload string
	var count int
	if err := other.pool.QueryRowContext(ctx, `SELECT count(*) FROM flat_events`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("persisted events: %d, %v", count, err)
	}
	if err := other.pool.QueryRowContext(ctx, `SELECT payload FROM flat_events`).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	var event model.FlatInfo
	if err := json.Unmarshal([]byte(payload), &event); err != nil || event.Link != flat.Link || !event.HasDishwasher {
		t.Fatalf("event round trip: %+v, %v", event, err)
	}
	var stations []string
	if err := other.pool.QueryRowContext(ctx, `SELECT underground_stations FROM flats_history`).Scan(sqlite.JSON(&stations)); err != nil || len(stations) != 2 {
		t.Fatalf("station round trip: %v, %v", stations, err)
	}
	if _, err := database.pool.ExecContext(ctx, `CREATE TRIGGER fail_event BEFORE INSERT ON flat_events BEGIN SELECT RAISE(ABORT, 'queue unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	flat.Link = "https://example.test/flat/rollback"
	if _, err := database.InsertFlatAndEnqueue(ctx, flat); err == nil {
		t.Fatal("expected queue insertion failure")
	}
	if exists, err := database.FlatExists(ctx, flat.Link); err != nil || exists {
		t.Fatalf("listing survived rolled back event: %v, %v", exists, err)
	}
}

func TestCatalogImportDoesNotEnqueue(t *testing.T) {
	database, _ := testDB(t)
	ctx := context.Background()
	flat := &model.FlatInfo{Link: "https://realty.yandex.ru/offer/123/", Region: 1, DealType: "rent", Price: 55000, TotalArea: 37.9}
	if inserted, err := database.InsertCatalogFlat(ctx, flat); err != nil || !inserted {
		t.Fatalf("insert: %v, %v", inserted, err)
	}
	if inserted, err := database.InsertCatalogFlat(ctx, flat); err != nil || inserted {
		t.Fatalf("duplicate: %v, %v", inserted, err)
	}
	if exists, err := database.FlatExists(ctx, flat.Link); err != nil || !exists {
		t.Fatalf("listing exists: %v, %v", exists, err)
	}
	var count int
	if err := database.pool.QueryRowContext(ctx, `SELECT count(*) FROM flat_events`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("catalog import events: %d, %v", count, err)
	}
}
