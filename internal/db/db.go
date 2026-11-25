package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/asmisnik/realty-parser/internal/model"
	"github.com/asmisnik/realty-parser/internal/sqlite"
)

type DB struct {
	pool *sql.DB
}

func New(ctx context.Context, dsn string) (*DB, error) {
	pool, err := sqlite.Open(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("opening SQLite database: %w", err)
	}
	return &DB{pool: pool}, nil
}

func (d *DB) Close() {
	d.pool.Close()
}

// RegionDealType is a distinct (region, deal_type) combination to search for.
type RegionDealType struct {
	Region   int
	DealType string
}

// GetDistinctRegionDealTypes returns the distinct (region, deal_type) combinations
// of currently active user subscriptions, ordered by region then deal_type.
func (d *DB) GetDistinctRegionDealTypes(ctx context.Context) ([]RegionDealType, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT DISTINCT region, deal_type FROM user_subscriptions
		 WHERE is_active = TRUE
		 ORDER BY region, deal_type`)
	if err != nil {
		return nil, fmt.Errorf("querying distinct region/deal_type combinations: %w", err)
	}
	defer rows.Close()

	var combos []RegionDealType
	for rows.Next() {
		var rdt RegionDealType
		if err := rows.Scan(&rdt.Region, &rdt.DealType); err != nil {
			return nil, fmt.Errorf("scanning region/deal_type: %w", err)
		}
		combos = append(combos, rdt)
	}
	return combos, rows.Err()
}

const existsByLink = `SELECT EXISTS(SELECT 1 FROM flats_history WHERE link = ?1
	UNION ALL SELECT 1 FROM flat_listing_aliases WHERE link = ?1)`

// FlatExists also recognizes links assigned to a listing from another platform.
func (d *DB) FlatExists(ctx context.Context, link string) (bool, error) {
	var exists bool
	err := d.pool.QueryRowContext(ctx, existsByLink, link).Scan(&exists)
	return exists, err
}

// InsertFlat writes a parsed flat into flats_history.
func (d *DB) InsertFlat(ctx context.Context, f *model.FlatInfo) error {
	_, err := d.InsertFlatAndEnqueue(ctx, f)
	return err
}

// InsertFlatAndEnqueue commits the listing and notification event atomically.
// A duplicate listing neither produces a second event nor increments metrics.
func (d *DB) InsertFlatAndEnqueue(ctx context.Context, f *model.FlatInfo) (bool, error) {
	payload, err := json.Marshal(f)
	if err != nil {
		return false, err
	}
	tx, err := d.pool.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	if platform(f.Link) != "" {
		var alreadySeen bool
		if err := tx.QueryRowContext(ctx, existsByLink, f.Link).Scan(&alreadySeen); err != nil {
			return false, err
		}
		if alreadySeen {
			return false, nil
		}
		match, err := findDuplicate(ctx, tx, f)
		if err != nil {
			return false, fmt.Errorf("matching cross-platform listing: %w", err)
		}
		if match != 0 {
			if _, err := tx.ExecContext(ctx, `INSERT INTO flat_listing_aliases (link, flat_id, payload)
				VALUES (?1, ?2, ?3)`, f.Link, match, string(payload)); err != nil {
				return false, err
			}
			return false, tx.Commit()
		}
	}
	stations := f.UndergroundStations
	if stations == nil {
		stations = []string{}
	}
	photoURLs := f.PhotoURLs
	if photoURLs == nil {
		photoURLs = []string{}
	}
	photoThumbnailURLs := f.PhotoThumbnailURLs
	if photoThumbnailURLs == nil {
		photoThumbnailURLs = []string{}
	}

	var id int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO flats_history (
			link, region, deal_type, price, flat_score,
			underground_score, underground_place, underground_distance_info, underground_stations,
			room_number, total_area, living_area, kitchen_area,
			floor, max_floor, deposit, deposit_months, comission,
			renovation, is_apartments, loggia_count, balcony_count, windows_view,
			separated_bathroom_count, combined_bathroom_count,
			has_dishwasher, has_conditioner, children_allowed, pets_allowed,
			last_updated, ceiling_height,
			building_entrances_number, building_apartments_number, building_elevators_number,
			sale_type, mortgage_allowed, is_new_building, new_building_name,
			is_by_homeowner, demolished_in_moscow_program, latitude, longitude,
			photo_urls, photo_thumbnail_urls, address
		) VALUES (
			?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,?12,?13,?14,?15,?16,?17,
			?18,?19,?20,?21,?22,?23,?24,?25,?26,?27,?28,?29,?30,?31,?32,?33,
			?34,?35,?36,?37,?38,?39,?40,?41,?42,?43,?44,?45
		) ON CONFLICT (link) DO NOTHING RETURNING id`,
		f.Link, f.Region, f.DealType, f.Price, f.FlatScore,
		f.UndergroundScore, f.UndergroundPlace, f.UndergroundDistanceInfo, sqlite.Strings(stations),
		f.RoomNumber, f.TotalArea, f.LivingArea, f.KitchenArea,
		f.Floor, f.MaxFloor, f.Deposit, f.DepositMonths, f.Comission,
		f.Renovation, f.IsApartments, f.LoggiaCount, f.BalconyCount, f.WindowsView,
		f.SeparatedBathroomCount, f.CombinedBathroomCount,
		f.HasDishwasher, f.HasConditioner, f.ChildrenAllowed, f.PetsAllowed,
		f.LastUpdated, f.CeilingHeight,
		f.BuildingEntrancesNumber, f.BuildingApartmentsNumber, f.BuildingElevatorsNumber,
		f.SaleType, f.MortgageAllowed, f.IsNewBuilding, f.NewBuildingName,
		f.IsByHomeowner, f.DemolishedInMoscowProgram, f.Latitude, f.Longitude,
		sqlite.Strings(photoURLs), sqlite.Strings(photoThumbnailURLs), f.Address,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO flat_events (flat_id, payload) VALUES (?1, ?2)`, id, string(payload)); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}
