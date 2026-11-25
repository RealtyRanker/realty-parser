package yandex

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/asmisnik/realty-parser/internal/config"
)

func TestSearchMapsSaleToFlatInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("type") != "SELL" || q.Get("rgid") != "587795" || q.Get("priceMax") != "40000000" ||
			!reflect.DeepEqual(q["roomsTotal"], []string{"STUDIO", "1", "PLUS_4"}) {
			t.Errorf("unexpected search query: %s", q.Encode())
		}
		fmt.Fprint(w, `{"response":{"search":{"offers":{"entities":[
			{"offerId":"3937530341846061844","offerType":"SELL","offerCategory":"APARTMENT","active":true,
			 "roomsTotal":1,"floorsTotal":49,"floorsOffered":[36],"ceilingHeight":3.1,
			 "area":{"value":44},"livingSpace":{"value":25},"kitchenSpace":{"value":15},
			 "price":{"currency":"RUR","value":727273,"unit":"SQUARE_METER","valueForWhole":32000000,"period":"WHOLE_LIFE"},
			 "location":{"address":"Москва, улица Генерала Глаголева, 14Ак1","point":{"latitude":55.78199,"longitude":37.46377},
				"metroList":[{"name":"Народное Ополчение","timeToMetro":20,"metroTransport":"ON_FOOT"}]},
			 "house":{"windowView":"YARD"},"apartment":{"renovation":"RENOVATED"},
			 "fullImages":["https://images.example/large"],"origImages":["https://images.example/orig"],
			 "dealStatus":"SALE","updateDate":"2026-08-22T06:26:00Z"},
			{"offerId":"2","roomsTotal":2,"price":{"value":32000000},"area":{"value":44}}
		]}}}}`)
	}))
	defer server.Close()
	c := New(5 * time.Second)
	c.url = server.URL
	flats, err := c.Search(context.Background(), config.SearchConfig{Rooms: []int{1, 2, 6}, MaxPrice: 40000000}, 1, 587795, "sale")
	if err != nil || len(flats) != 1 {
		t.Fatalf("search = %d listings, %v", len(flats), err)
	}
	f := flats[0]
	if f.Link != "https://realty.yandex.ru/offer/3937530341846061844/" || f.Price != 32000000 ||
		f.DealType != "sale" || f.TotalArea != 44 || f.LivingArea != 25 || f.KitchenArea != 15 ||
		f.Floor != 36 || f.MaxFloor != 49 || f.SaleType != "free" || f.Address == "" || f.Latitude == 0 ||
		f.UndergroundPlace <= 0 || len(f.UndergroundStations) != 1 || f.WindowsView != "yard" || f.FlatScore == 0 {
		t.Fatalf("wrong mapping: %+v", f)
	}
	if !reflect.DeepEqual(f.PhotoURLs, []string{"https://images.example/orig"}) ||
		!reflect.DeepEqual(f.PhotoThumbnailURLs, []string{"https://images.example/large"}) {
		t.Fatalf("wrong photo mapping: %v %v", f.PhotoURLs, f.PhotoThumbnailURLs)
	}
}

func TestWrappedRentalOffersUseMonthlyRublePrices(t *testing.T) {
	body := []byte(`{"response":{"search":{"offers":[
		{"offer":{"id":9383468145864276905,"rooms":0,"price":{"value":73000,"currency":"RUB","period":"MONTH"},"area":{"value":26},"rentPledge":73000,"withPets":true,"agentFee":40}},
		{"offer":{"id":"2","rooms":1,"price":{"value":3000,"period":"DAY"},"area":{"value":31}}},
		{"offer":{"id":"3","rooms":1,"price":{"value":3000,"currency":"USD","period":"MONTH"},"area":{"value":31}}},
		{"offer":{"id":"4","rooms":1,"price":{"value":3000},"area":{"value":31}}}
	]}}}`)
	flats, err := parseSearch(body, 1, "rent")
	if err != nil || len(flats) != 1 {
		t.Fatalf("search = %d listings, %v", len(flats), err)
	}
	f := flats[0]
	if f.Link != "https://realty.yandex.ru/offer/9383468145864276905/" || f.RoomNumber != 0 ||
		f.Price != 73000 || f.Deposit != 73000 || f.Comission != 40 || !f.PetsAllowed {
		t.Fatalf("wrong mapping: %+v", f)
	}
}

func TestSearchStopsAtChallenge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/showcaptcha" {
			t.Error("client followed the challenge redirect")
			return
		}
		http.Redirect(w, r, "/showcaptcha", http.StatusFound)
	}))
	defer server.Close()
	c := New(time.Second)
	c.url = server.URL
	if _, err := c.Search(context.Background(), config.SearchConfig{}, 1, 587795, "rent"); !errors.Is(err, ErrBlocked) {
		t.Fatalf("expected a challenge error, got %v", err)
	}
}
