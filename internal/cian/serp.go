package cian

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/asmisnik/realty-parser/internal/model"
)

const serpConfigPrefix = "window._cianConfig['frontend-serp'] = (window._cianConfig['frontend-serp'] || []).concat("

// ParseSearchPageOffers extracts full listings straight from a CIAN search results
// page. The page embeds the same offer objects as flat pages inside
// initialState.results.offers, so one search request yields ~28 listings and no
// per-flat requests are needed (flat pages are the ones CIAN shows a captcha on).
//
// Fields CIAN doesn't include in search results (renovation, bathrooms, appliances,
// pets/children rules, ceiling height, building data) are left zero-valued.
func ParseSearchPageOffers(body string, region int, dealType string, logger *zap.Logger) ([]*model.FlatInfo, error) {
	idx := strings.Index(body, serpConfigPrefix)
	if idx == -1 {
		logger.Warn("search config not found in page — captcha/block or markup change",
			zap.String("body_preview", truncate(body, 300)))
		return nil, fmt.Errorf("search results data not found in page")
	}
	return parseSerpJSON(body[idx+len(serpConfigPrefix):], region, dealType)
}

func parseSerpJSON(rest string, region int, dealType string) ([]*model.FlatInfo, error) {
	var items []json.RawMessage
	if err := json.NewDecoder(strings.NewReader(rest)).Decode(&items); err != nil {
		return nil, fmt.Errorf("unmarshalling search config array: %w", err)
	}

	for _, item := range items {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(item, &obj); err != nil {
			continue
		}
		var key string
		if err := json.Unmarshal(obj["key"], &key); err != nil || key != "initialState" {
			continue
		}

		var state map[string]json.RawMessage
		if err := json.Unmarshal(obj["value"], &state); err != nil {
			return nil, fmt.Errorf("unmarshalling initialState: %w", err)
		}
		offersRaw := jsonArray(nestedMap(state, "results"), "offers")

		flats := make([]*model.FlatInfo, 0, len(offersRaw))
		for _, raw := range offersRaw {
			var offer map[string]json.RawMessage
			if json.Unmarshal(raw, &offer) != nil {
				continue
			}
			href := jsonString(offer, "fullUrl")
			if q := strings.IndexByte(href, '?'); q != -1 {
				href = href[:q]
			}
			if href == "" {
				continue
			}
			normalizeSerpUndergrounds(offer)
			flats = append(flats, buildFlatInfo(offer, nil, href, region, dealType))
		}
		return flats, nil
	}
	return nil, fmt.Errorf("initialState key not found in search config")
}

// normalizeSerpUndergrounds renames the station fields used in search results
// ("time", "transportType") to the ones scoring.ParseUndergroundInfo expects
// from flat pages ("travelTime", "travelType").
func normalizeSerpUndergrounds(offer map[string]json.RawMessage) {
	geo := nestedMap(offer, "geo")
	stations := jsonArray(geo, "undergrounds")
	if len(stations) == 0 {
		return
	}
	norm := make([]map[string]json.RawMessage, 0, len(stations))
	for _, raw := range stations {
		var st map[string]json.RawMessage
		if json.Unmarshal(raw, &st) != nil {
			continue
		}
		if _, ok := st["travelTime"]; !ok {
			if v, ok := st["time"]; ok {
				st["travelTime"] = v
			}
		}
		if _, ok := st["travelType"]; !ok {
			if v, ok := st["transportType"]; ok {
				st["travelType"] = v
			}
		}
		norm = append(norm, st)
	}
	if b, err := json.Marshal(norm); err == nil {
		geo["undergrounds"] = b
		if gb, err := json.Marshal(geo); err == nil {
			offer["geo"] = gb
		}
	}
}
