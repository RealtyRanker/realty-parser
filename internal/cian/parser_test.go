package cian

import "testing"

// CIAN packs multiple <script> blocks onto a single physical line with no
// separator, so real pages have trailing HTML/JS (typically the ");" that
// closes the surrounding .concat(...) call, then more markup) immediately
// after the offer-card array's closing "]". parseOfferCardJSON must ignore
// that trailing content instead of treating it as a parse error.
func TestParseOfferCardJSON_ToleratesTrailingScriptTags(t *testing.T) {
	rest := `[{"key":"someOtherKey","value":{}}]);</script><script type="text/javascript">doOtherStuff();</script>`

	_, err := parseOfferCardJSON(rest, "https://example.test/flat/1", 1, "rent")
	if err == nil {
		t.Fatalf("expected an error (no defaultState in fixture), got nil")
	}
	if err.Error() != "defaultState key not found in offer card" {
		t.Fatalf("expected to get past JSON parsing and fail on missing defaultState, got: %v", err)
	}
}

func TestParseOfferCardJSON_RejectsGenuinelyInvalidJSON(t *testing.T) {
	_, err := parseOfferCardJSON("not json at all", "https://example.test/flat/1", 1, "rent")
	if err == nil {
		t.Fatalf("expected an error for invalid JSON, got nil")
	}
}

func TestParseSerpJSON_ExtractsOffers(t *testing.T) {
	rest := `[{"key":"other","value":1},{"key":"initialState","value":{"results":{"offers":[` +
		`{"fullUrl":"https://www.cian.ru/rent/flat/1/?utm=x","roomsCount":2,"totalArea":"42.0","livingArea":"28.0","kitchenArea":"6.0",` +
		`"floorNumber":1,"building":{"floorsCount":5},"bargainTerms":{"price":60000,"agentFee":50,"deposit":60000},` +
		`"geo":{"coordinates":{"lat":55.9,"lng":37.4},"address":[{"type":"location","name":"Москва"},{"type":"street","name":"Головинское шоссе"},{"type":"house","name":"7"}],"undergrounds":[{"name":"Водный стадион","time":16,"transportType":"walk"}]},` +
		`"photos":[{"fullUrl":"https://img/1.jpg","thumbnailUrl":"https://img/1t.jpg"}],"humanizedTimedelta":"1 минуту назад"},` +
		`{"roomsCount":1}]}}}]);</script>`

	flats, err := parseSerpJSON(rest, 1, "rent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flats) != 1 {
		t.Fatalf("expected 1 flat (offer without fullUrl skipped), got %d", len(flats))
	}
	f := flats[0]
	if f.Link != "https://www.cian.ru/rent/flat/1/" || f.Price != 60000 || f.RoomNumber != 2 || f.TotalArea != 42 ||
		f.KitchenArea != 6 || f.Floor != 1 || f.MaxFloor != 5 || f.Comission != 50 || f.Deposit != 60000 {
		t.Fatalf("unexpected flat: %+v", f)
	}
	if f.Latitude != 55.9 || len(f.PhotoURLs) != 1 || f.Address != "Москва, Головинское шоссе, 7" {
		t.Fatalf("coordinates/photos/address not parsed: %+v", f)
	}
	if len(f.UndergroundStations) != 1 || f.UndergroundStations[0] != "Водный стадион" {
		t.Fatalf("underground not parsed: %+v", f.UndergroundStations)
	}
	if f.LastUpdated != "1 минуту назад" {
		t.Fatalf("LastUpdated fallback not applied: %q", f.LastUpdated)
	}
}

func TestParseSerpJSON_MissingInitialState(t *testing.T) {
	if _, err := parseSerpJSON(`[{"key":"x","value":1}]);`, 1, "rent"); err == nil {
		t.Fatalf("expected error when initialState is absent")
	}
}
