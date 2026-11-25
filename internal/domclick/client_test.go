package domclick

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/asmisnik/realty-parser/internal/config"
)

func TestSearchMapsAndFiltersOffer(t *testing.T) {
	const flat = `{"id":2082491537,"path":"https://domclick.ru/card/rent__flat__2082491537?from=search","offerType":"flat","dealType":"rent","status":2,"price":77800,
		"objectInfo":{"area":38,"rooms":2,"floor":5},"house":{"floors":17},"location":{"lat":55.658986,"lon":37.726335},
		"address":{"displayName":"Москва, Иловайская улица, 4","subways":[{"name":"Перерва","remoteness":{"kind":1,"time":9}}]},
		"photos":[{"url":"/vitrina/9b/96/test.jpg"},{"url":"/vitrina/9b/96/test.jpg"}],"publishedDate":"2026-11-25T19:28:12Z"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("offset") != "20" || q.Get("address") != "location" || q.Get("aids") != "2299" ||
			q.Get("deal_type") != "rent" || q.Get("sort") != "published" || !reflect.DeepEqual(q["offer_type"], []string{"flat"}) {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		other := strings.Replace(strings.ReplaceAll(flat, "2082491537", "2082491538"), `"price":77800`, `"price":120000`, 1)
		fmt.Fprintf(w, `{"result":{"items":[%s,%s,%s],"pagination":{"offset":20,"limit":20,"total":45}}}`, flat, flat, other)
	}))
	defer server.Close()
	client := New(time.Second)
	client.url = server.URL
	page, err := client.Search(context.Background(), config.SearchConfig{MaxPrice: 80000, Rooms: []int{3}}, 1, config.DomclickRegion{Address: "location", AID: 2299}, "rent", 2)
	if err != nil || len(page.Flats) != 1 || !page.HasMore {
		t.Fatalf("search = %+v, %v", page, err)
	}
	got := page.Flats[0]
	if got.Price != 77800 || got.Floor != 5 || got.MaxFloor != 17 || got.RoomNumber != 2 || got.TotalArea != 38 || got.Longitude != 37.726335 || got.LastUpdated == "" ||
		got.Link != "https://domclick.ru/card/rent__flat__2082491537" || !reflect.DeepEqual(got.UndergroundStations, []string{"Перерва"}) ||
		!reflect.DeepEqual(got.PhotoURLs, []string{"https://img.dmclk.ru/s1200x800q80/vitrina/9b/96/test.webp"}) {
		t.Fatalf("unexpected offer mapping: %+v", got)
	}
}

func TestSearchStopsAtAccessDenial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !reflect.DeepEqual(r.URL.Query()["offer_type"], []string{"flat", "layout"}) {
			t.Errorf("unexpected sale query: %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	client := New(time.Second)
	client.url = server.URL
	_, err := client.Search(context.Background(), config.SearchConfig{}, 1, config.DomclickRegion{Address: "location", AID: 2299}, "sale", 1)
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("expected blocked error, got %v", err)
	}
}
