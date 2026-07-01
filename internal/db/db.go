package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/asmisnik/realty-parser/internal/model"
)

type DB struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*DB, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("creating pgx pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}
	return &DB{pool: pool}, nil
}

func (d *DB) Close() {
	d.pool.Close()
}

// GetDistinctRegions returns the distinct region ids of currently active user subscriptions,
// ordered ascending.
func (d *DB) GetDistinctRegions(ctx context.Context) ([]int, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT DISTINCT region FROM user_subscriptions WHERE is_active = TRUE ORDER BY region`)
	if err != nil {
		return nil, fmt.Errorf("querying distinct regions: %w", err)
	}
	defer rows.Close()

	var regions []int
	for rows.Next() {
		var region int
		if err := rows.Scan(&region); err != nil {
			return nil, fmt.Errorf("scanning region: %w", err)
		}
		regions = append(regions, region)
	}
	return regions, rows.Err()
}

// FlatExists returns true if a flat with the given link is already in flats_history.
func (d *DB) FlatExists(ctx context.Context, link string) (bool, error) {
	var exists bool
	err := d.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM flats_history WHERE link = $1)", link,
	).Scan(&exists)
	return exists, err
}

// InsertFlat writes a parsed flat into flats_history.
func (d *DB) InsertFlat(ctx context.Context, f *model.FlatInfo) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO flats_history (
			link, region, price, flat_score,
			underground_score, underground_place, underground_distance_info,
			room_number, total_area, living_area, kitchen_area,
			floor, max_floor, deposit, deposit_months, comission,
			renovation, is_apartments, loggia_count, balcony_count, windows_view,
			separated_bathroom_count, combined_bathroom_count,
			has_dishwasher, has_conditioner, children_allowed, pets_allowed,
			last_updated, ceiling_height,
			building_entrances_number, building_apartments_number, building_elevators_number
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,
			$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32
		) ON CONFLICT (link) DO NOTHING`,
		f.Link, f.Region, f.Price, f.FlatScore,
		f.UndergroundScore, f.UndergroundPlace, f.UndergroundDistanceInfo,
		f.RoomNumber, f.TotalArea, f.LivingArea, f.KitchenArea,
		f.Floor, f.MaxFloor, f.Deposit, f.DepositMonths, f.Comission,
		f.Renovation, f.IsApartments, f.LoggiaCount, f.BalconyCount, f.WindowsView,
		f.SeparatedBathroomCount, f.CombinedBathroomCount,
		f.HasDishwasher, f.HasConditioner, f.ChildrenAllowed, f.PetsAllowed,
		f.LastUpdated, f.CeilingHeight,
		f.BuildingEntrancesNumber, f.BuildingApartmentsNumber, f.BuildingElevatorsNumber,
	)
	return err
}
