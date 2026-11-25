package db

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/asmisnik/realty-parser/internal/model"
)

func TestCrossPlatformDeduplication(t *testing.T) {
	tests := []struct {
		name      string
		change    func(cian, yandex *model.FlatInfo)
		duplicate bool
	}{
		{name: "formatted address", duplicate: true},
		{name: "legacy listing without address", change: func(cian, yandex *model.FlatInfo) {
			cian.Address = ""
			cian.Latitude, cian.Longitude = 55.781, 37.464
			yandex.Latitude, yandex.Longitude = 55.7811, 37.4641
		}, duplicate: true},
		{name: "different prices", change: func(_, yandex *model.FlatInfo) { yandex.Price++ }},
		{name: "different buildings", change: func(cian, yandex *model.FlatInfo) {
			yandex.Address = "Москва, ул. Генерала Глаголева, 14к2"
			cian.Latitude, cian.Longitude = 55.781, 37.464
			yandex.Latitude, yandex.Longitude = 55.781, 37.464
		}},
		{name: "different layouts", change: func(_, yandex *model.FlatInfo) { yandex.KitchenArea += 3 }},
		{name: "missing layout", change: func(_, yandex *model.FlatInfo) { yandex.KitchenArea = 0 }},
		{name: "same platform", change: func(_, yandex *model.FlatInfo) {
			yandex.Link = "https://www.cian.ru/rent/flat/2/"
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			database, _ := testDB(t)
			ctx := context.Background()
			cian := model.FlatInfo{
				Link: "https://www.cian.ru/rent/flat/1/", Region: 1, DealType: "rent",
				Price: 85000, RoomNumber: 2, Floor: 3, MaxFloor: 12, TotalArea: 45, KitchenArea: 12,
				Address: "Москва, Генерала Глаголева улица, 14 корп. 1",
			}
			yandex := cian
			yandex.Link = "https://realty.yandex.ru/offer/2/"
			yandex.Address = "Россия, Москва, ул. Генерала Глаголева, 14к1"
			yandex.TotalArea = 45.2
			yandex.HasDishwasher = true
			if tc.change != nil {
				tc.change(&cian, &yandex)
			}
			if inserted, err := database.InsertFlatAndEnqueue(ctx, &cian); err != nil || !inserted {
				t.Fatalf("insert CIAN: %t, %v", inserted, err)
			}
			if inserted, err := database.InsertFlatAndEnqueue(ctx, &yandex); err != nil || inserted == tc.duplicate {
				t.Fatalf("insert Yandex: %t, %v", inserted, err)
			}
			if exists, err := database.FlatExists(ctx, yandex.Link); err != nil || !exists {
				t.Fatalf("second platform URL was lost: %t, %v", exists, err)
			}
			if inserted, err := database.InsertFlatAndEnqueue(ctx, &yandex); err != nil || inserted {
				t.Fatalf("repeat Yandex: %t, %v", inserted, err)
			}
			if !tc.duplicate {
				return
			}
			var payload string
			var canonical, events int
			err := database.pool.QueryRowContext(ctx, `SELECT payload, flat_id, (SELECT COUNT(*) FROM flat_events)
				FROM flat_listing_aliases WHERE link = ?1`, yandex.Link).Scan(&payload, &canonical, &events)
			if err != nil || events != 1 || canonical != 1 {
				t.Fatalf("alias: canonical id %d, events %d, error %v", canonical, events, err)
			}
			var saved model.FlatInfo
			if err := json.Unmarshal([]byte(payload), &saved); err != nil || !saved.HasDishwasher || saved.Link != yandex.Link {
				t.Fatalf("alias data lost: %+v, %v", saved, err)
			}
		})
	}
}

func TestCrossPlatformDeduplicationLeavesAmbiguousOffersSeparate(t *testing.T) {
	database, _ := testDB(t)
	ctx := context.Background()
	flat := model.FlatInfo{
		Region: 1, DealType: "sale", Price: 10000000, RoomNumber: 2, Floor: 3, MaxFloor: 12,
		TotalArea: 45, KitchenArea: 12, Address: "Москва, улица Генерала Глаголева, 14к1",
	}
	for i := 1; i <= 2; i++ {
		flat.Link = fmt.Sprintf("https://www.cian.ru/sale/flat/%d/", i)
		if inserted, err := database.InsertFlatAndEnqueue(ctx, &flat); !inserted || err != nil {
			t.Fatalf("insert CIAN %d: %t, %v", i, inserted, err)
		}
	}
	flat.Link = "https://realty.yandex.ru/offer/3/"
	if inserted, err := database.InsertFlatAndEnqueue(ctx, &flat); !inserted || err != nil {
		t.Fatalf("insert ambiguous Yandex offer: %t, %v", inserted, err)
	}
}

func TestConcurrentCrossPlatformOffersGenerateOneEvent(t *testing.T) {
	first, dsn := testDB(t)
	ctx := context.Background()
	second, err := New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()

	cian := model.FlatInfo{
		Link: "https://www.cian.ru/rent/flat/1/", Region: 1, DealType: "rent", Price: 80000,
		RoomNumber: 2, Floor: 5, TotalArea: 47, LivingArea: 29,
		Address: "Москва, Головинское шоссе, 7",
	}
	yandex := cian
	yandex.Link = "https://realty.yandex.ru/offer/2/"

	results := make(chan error, 2)
	go func() {
		_, err := first.InsertFlatAndEnqueue(ctx, &cian)
		results <- err
	}()
	go func() {
		_, err := second.InsertFlatAndEnqueue(ctx, &yandex)
		results <- err
	}()
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	var flats, events, aliases int
	err = first.pool.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM flats_history),
		(SELECT COUNT(*) FROM flat_events), (SELECT COUNT(*) FROM flat_listing_aliases)`).Scan(&flats, &events, &aliases)
	if err != nil || flats != 1 || events != 1 || aliases != 1 {
		t.Fatalf("flats %d, events %d, aliases %d, error %v", flats, events, aliases, err)
	}
}
