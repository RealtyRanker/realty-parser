package cian

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/net/html"

	"github.com/asmisnik/realty-parser/internal/model"
	"github.com/asmisnik/realty-parser/internal/scoring"
)

const cianConfigPrefix = "window._cianConfig['frontend-offer-card'] = (window._cianConfig['frontend-offer-card'] || []).concat("

// moscowRegionID is the CIAN region id for Moscow, the only region the
// underground (metro) station ranking data in the scoring package covers.
const moscowRegionID = 1

// ParseSearchPageHrefs extracts flat listing URLs from a CIAN search results page.
// Every other LinkArea node is skipped (mirrors original C++ logic).
func ParseSearchPageHrefs(body string, logger *zap.Logger) ([]string, error) {
	// Quick sanity check — bot-wall or empty response
	if len(body) < 1000 {
		logger.Warn("search page body suspiciously small — possible bot block or redirect",
			zap.String("body_preview", body))
		return nil, fmt.Errorf("search page body too small (%d bytes)", len(body))
	}
	if !strings.Contains(body, "cian") {
		logger.Warn("search page body does not contain 'cian' — likely a captcha/block page",
			zap.String("body_preview", truncate(body, 500)))
	}

	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	var linkAreaNodes []*html.Node
	collectLinkAreaNodes(doc, &linkAreaNodes)

	if len(linkAreaNodes) == 0 {
		// Help diagnose a structure change: look for any data-name attrs present
		var dataNames []string
		collectDataNames(doc, &dataNames, 20)
		logger.Warn("no LinkArea nodes found — CIAN HTML structure may have changed",
			zap.Strings("data_names_sample", dataNames))
		return nil, nil
	}

	var hrefs []string
	for i, node := range linkAreaNodes {
		if i%2 == 1 {
			continue
		}
		href, _ := extractHrefDebug(node)
		// Strip tracking query params — store only canonical flat URL
		if q := strings.IndexByte(href, '?'); q != -1 {
			href = href[:q]
		}
		if href != "" {
			hrefs = append(hrefs, href)
		}
	}

	logger.Info("search page parsed", zap.Int("hrefs_found", len(hrefs)))
	return hrefs, nil
}

func collectLinkAreaNodes(n *html.Node, result *[]*html.Node) {
	if n.Type == html.ElementNode && n.Data == "div" {
		for _, attr := range n.Attr {
			if attr.Key == "data-name" && attr.Val == "LinkArea" {
				*result = append(*result, n)
				break
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectLinkAreaNodes(c, result)
	}
}

// collectDataNames gathers unique data-name attribute values for diagnostics.
func collectDataNames(n *html.Node, result *[]string, limit int) {
	if len(*result) >= limit {
		return
	}
	if n.Type == html.ElementNode {
		for _, attr := range n.Attr {
			if attr.Key == "data-name" {
				for _, existing := range *result {
					if existing == attr.Val {
						goto next
					}
				}
				*result = append(*result, attr.Val)
			}
		}
	}
next:
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectDataNames(c, result, limit)
	}
}

// extractHrefDebug returns the href of the first <a> inside the node tree.
// linkClassFound is true when the CSS class contains "--link" (used only for debug logging).
func extractHrefDebug(n *html.Node) (href string, linkClassFound bool) {
	if n.Type == html.ElementNode && n.Data == "a" {
		hasLinkClass := false
		foundHref := ""
		for _, attr := range n.Attr {
			if attr.Key == "class" && strings.Contains(attr.Val, "--link") {
				hasLinkClass = true
			}
			if attr.Key == "href" {
				foundHref = attr.Val
			}
		}
		if foundHref != "" {
			return foundHref, hasLinkClass
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if h, found := extractHrefDebug(c); h != "" {
			return h, found
		}
	}
	return "", false
}

func extractHref(n *html.Node) string {
	h, _ := extractHrefDebug(n)
	return h
}

// ParseFlatPage extracts FlatInfo from a CIAN flat detail page HTML body.
func ParseFlatPage(body, href string, region int, dealType string, logger *zap.Logger) (*model.FlatInfo, error) {
	idx := strings.Index(body, cianConfigPrefix)
	if idx == -1 {
		logger.Warn("offer card JS not found in flat page",
			zap.String("href", href),
			zap.String("body_preview", truncate(body, 300)),
		)
		return nil, fmt.Errorf("offer card data not found in page")
	}

	rest := body[idx+len(cianConfigPrefix):]

	info, err := parseOfferCardJSON(rest, href, region, dealType)
	if err != nil {
		logger.Warn("offer card line parse failed", zap.String("href", href), zap.Error(err))
		return nil, err
	}
	logger.Debug("flat coordinates",
		zap.String("href", href),
		zap.Bool("found", info.Latitude != 0 || info.Longitude != 0),
		zap.Float64("lat", info.Latitude),
		zap.Float64("lon", info.Longitude),
	)
	logger.Info("flat page parsed",
		zap.String("href", href),
		zap.Int("price", info.Price),
		zap.Int("score", info.FlatScore),
		zap.Int("rooms", info.RoomNumber),
		zap.Float64("area", info.TotalArea),
	)
	return info, nil
}

// parseOfferCardJSON decodes the offer-card JSON array that immediately
// follows cianConfigPrefix in the page source. It uses a streaming
// json.Decoder rather than json.Unmarshal because CIAN's markup packs
// multiple <script> blocks onto a single physical line with no separator:
// right after the array's closing "]" there's typically the ");" that
// closes the surrounding .concat(...) call, immediately followed by more
// HTML/JS with no newline in between. json.Unmarshal requires the entire
// input to be a single valid JSON value and errors on that trailing
// content; a Decoder stops as soon as it has read one complete value and
// simply ignores whatever follows.
func parseOfferCardJSON(rest, href string, region int, dealType string) (*model.FlatInfo, error) {
	var items []json.RawMessage
	if err := json.NewDecoder(strings.NewReader(rest)).Decode(&items); err != nil {
		return nil, fmt.Errorf("unmarshalling offer card array: %w", err)
	}

	for _, item := range items {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(item, &obj); err != nil {
			continue
		}
		var key string
		if err := json.Unmarshal(obj["key"], &key); err != nil || key != "defaultState" {
			continue
		}
		return parseDefaultState(obj["value"], href, region, dealType)
	}
	return nil, fmt.Errorf("defaultState key not found in offer card")
}

func parseDefaultState(valueRaw json.RawMessage, href string, region int, dealType string) (*model.FlatInfo, error) {
	var state map[string]json.RawMessage
	if err := json.Unmarshal(valueRaw, &state); err != nil {
		return nil, fmt.Errorf("unmarshalling defaultState: %w", err)
	}

	offerData := nestedMap(state, "offerData")
	offer := nestedMap(offerData, "offer")
	bti := nestedMap(offerData, "bti")
	houseData := nestedMap(bti, "houseData")

	return buildFlatInfo(offer, houseData, href, region, dealType), nil
}

// buildFlatInfo maps a CIAN offer object to FlatInfo. The same offer model is
// embedded in both the flat page (full data, plus houseData from BTI) and the
// search results page (a subset: no repair type, bathrooms, appliances, pet /
// children rules, ceiling height or houseData — those stay zero-valued).
func buildFlatInfo(offer, houseData map[string]json.RawMessage, href string, region int, dealType string) *model.FlatInfo {
	bargainTerms := nestedMap(offer, "bargainTerms")
	building := nestedMap(offer, "building")

	newbuilding := nestedMap(offer, "newbuilding")

	// Comission represents the agent's percentage fee for both deal types,
	// but CIAN structures it differently: a flat "agentFee" for rent vs a
	// nested "agentBonus": {paymentType, value} object for sale, where value
	// is only a percentage when paymentType is percentage-based — for a fixed
	// fee it's an absolute ruble amount, which saleCommissionPercent converts
	// back to a price-relative percentage.
	comission := jsonInt(bargainTerms, "agentFee")
	if dealType == "sale" {
		agentBonus := nestedMap(bargainTerms, "agentBonus")
		comission = saleCommissionPercent(agentBonus, jsonInt(bargainTerms, "price"))
	}

	// Coordinates are reported (when present) regardless of region, unlike
	// the underground ranking below which only covers Moscow.
	geo := nestedMap(offer, "geo")
	coordinates := nestedMap(geo, "coordinates")
	latitude := jsonFloat(coordinates, "lat")
	longitude := jsonFloat(coordinates, "lng")

	photoURLs, photoThumbnailURLs := parsePhotos(offer)
	address := parseBuildingAddress(geo)

	// The underground (metro) ranking data only covers Moscow; skip it for
	// other regions instead of scoring against Moscow-only station data.
	var undergroundScore float64
	var undergroundPlace int
	var undergroundDistInfo string
	var undergroundStations []string
	if region == moscowRegionID {
		undergroundsRaw := jsonArray(geo, "undergrounds")
		undergroundScore, undergroundPlace, undergroundDistInfo, undergroundStations = scoring.ParseUndergroundInfo(undergroundsRaw)
	}

	info := &model.FlatInfo{
		Link:                     href,
		Region:                   region,
		DealType:                 dealType,
		Address:                  address,
		Latitude:                 latitude,
		Longitude:                longitude,
		PhotoURLs:                photoURLs,
		PhotoThumbnailURLs:       photoThumbnailURLs,
		Price:                    jsonInt(bargainTerms, "price"),
		RoomNumber:               jsonInt(offer, "roomsCount"),
		TotalArea:                jsonFloat(offer, "totalArea"),
		LivingArea:               jsonFloat(offer, "livingArea"),
		KitchenArea:              jsonFloat(offer, "kitchenArea"),
		Floor:                    jsonInt(offer, "floorNumber"),
		MaxFloor:                 jsonInt(building, "floorsCount"),
		Comission:                comission,
		Deposit:                  jsonInt(bargainTerms, "deposit"),
		DepositMonths:            jsonInt(bargainTerms, "prepayMonths"),
		Renovation:               jsonString(offer, "repairType"),
		IsApartments:             jsonBool(offer, "isApartments"),
		LoggiaCount:              jsonInt(offer, "loggiasCount"),
		BalconyCount:             jsonInt(offer, "balconiesCount"),
		WindowsView:              jsonString(offer, "windowsViewType"),
		SeparatedBathroomCount:   jsonInt(offer, "separateWcsCount"),
		CombinedBathroomCount:    jsonInt(offer, "combinedWcsCount"),
		HasDishwasher:            jsonBool(offer, "hasDishwasher"),
		HasConditioner:           jsonBool(offer, "hasConditioner"),
		ChildrenAllowed:          jsonBool(offer, "childrenAllowed"),
		PetsAllowed:              jsonBool(offer, "petsAllowed"),
		LastUpdated:              firstNonEmpty(jsonString(offer, "humanizedEditDate"), jsonString(offer, "humanizedTimedelta")),
		CeilingHeight:            jsonFloat(building, "ceilingHeight"),
		BuildingEntrancesNumber:  jsonInt(houseData, "entrances"),
		BuildingApartmentsNumber: jsonInt(houseData, "flatCount"),
		BuildingElevatorsNumber:  jsonInt(houseData, "lifts"),
		UndergroundScore:         undergroundScore,
		UndergroundPlace:         undergroundPlace,
		UndergroundDistanceInfo:  undergroundDistInfo,
		UndergroundStations:      undergroundStations,

		SaleType:                  jsonString(bargainTerms, "saleType"),
		MortgageAllowed:           jsonBool(bargainTerms, "mortgageAllowed"),
		IsNewBuilding:             jsonInt(newbuilding, "id") != 0,
		NewBuildingName:           jsonString(newbuilding, "name"),
		IsByHomeowner:             jsonBool(offer, "isByHomeowner"),
		DemolishedInMoscowProgram: jsonBool(offer, "demolishedInMoscowProgramm"),
	}
	info.FlatScore = scoring.CalculateFlatScore(info)
	return info
}

// CIAN lists address components in display order. Keep the locality and house
// number so addresses on streets with the same name can be distinguished.
func parseBuildingAddress(geo map[string]json.RawMessage) string {
	var parts []string
	hasLocality := false
	for _, raw := range jsonArray(geo, "address") {
		var part struct {
			Type string `json:"type"`
			Name string `json:"name"`
		}
		if json.Unmarshal(raw, &part) != nil || strings.TrimSpace(part.Name) == "" {
			continue
		}
		switch part.Type {
		case "location", "locality", "city", "settlement":
			hasLocality = true
			parts = append(parts, part.Name)
		case "street":
			parts = append(parts, part.Name)
		case "house":
			if hasLocality && len(parts) >= 2 {
				return strings.Join(append(parts, part.Name), ", ")
			}
		}
	}
	return ""
}

// saleCommissionPercent returns a sale listing's agent commission as a
// percentage of price, matching rent's agentFee semantic (calculateScore
// multiplies comissionMultiplier * comission * price, so comission must be
// percentage points, not rubles). CIAN's agentBonus.value is a percentage
// when paymentType is percentage-based, but an absolute ruble amount for a
// fixed fee — treating that as a raw percentage was overflowing FlatScore by
// billions (e.g. a ~190,000₽ fixed fee read as "190000%"). Any non-percentage
// paymentType is instead converted to its price-relative percentage.
func saleCommissionPercent(agentBonus map[string]json.RawMessage, price int) int {
	value := jsonInt(agentBonus, "value")
	paymentType := jsonString(agentBonus, "paymentType")
	if strings.EqualFold(paymentType, "percent") || strings.EqualFold(paymentType, "percentage") {
		return value
	}
	if price <= 0 {
		return 0
	}
	return value * 100 / price
}

// JSON navigation helpers

// maxPhotos caps how many photo URLs are stored per listing — CIAN listings
// can have dozens of photos, and the app only ever shows a handful.
const maxPhotos = 20

// parsePhotos extracts photo URLs from offer.photos[], each entry shaped
// like {"id":..., "isDefault":..., "fullUrl":"...", "thumbnailUrl":"...",
// ...}. Both slices are index-aligned and CIAN CDN URLs are public, so the
// app can load them directly without downloading/hosting the images itself.
func parsePhotos(offer map[string]json.RawMessage) (fullURLs, thumbnailURLs []string) {
	for _, raw := range jsonArray(offer, "photos") {
		var photo map[string]json.RawMessage
		if json.Unmarshal(raw, &photo) != nil {
			continue
		}
		full := jsonString(photo, "fullUrl")
		if full == "" {
			continue
		}
		fullURLs = append(fullURLs, full)
		thumbnailURLs = append(thumbnailURLs, jsonString(photo, "thumbnailUrl"))
		if len(fullURLs) >= maxPhotos {
			break
		}
	}
	return fullURLs, thumbnailURLs
}

func nestedMap(m map[string]json.RawMessage, key string) map[string]json.RawMessage {
	if m == nil {
		return nil
	}
	raw, ok := m[key]
	if !ok || string(raw) == "null" {
		return nil
	}
	var result map[string]json.RawMessage
	json.Unmarshal(raw, &result) //nolint:errcheck
	return result
}

func jsonArray(m map[string]json.RawMessage, key string) []json.RawMessage {
	if m == nil {
		return nil
	}
	raw, ok := m[key]
	if !ok || string(raw) == "null" {
		return nil
	}
	var result []json.RawMessage
	json.Unmarshal(raw, &result) //nolint:errcheck
	return result
}

func jsonInt(m map[string]json.RawMessage, key string) int {
	if m == nil {
		return 0
	}
	raw, ok := m[key]
	if !ok || string(raw) == "null" {
		return 0
	}
	var v int
	if json.Unmarshal(raw, &v) == nil {
		return v
	}
	return 0
}

func jsonFloat(m map[string]json.RawMessage, key string) float64 {
	if m == nil {
		return 0
	}
	raw, ok := m[key]
	if !ok || string(raw) == "null" {
		return 0
	}
	var v float64
	if json.Unmarshal(raw, &v) == nil {
		return v
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		var fv float64
		fmt.Sscanf(s, "%f", &fv)
		return fv
	}
	return 0
}

func jsonString(m map[string]json.RawMessage, key string) string {
	if m == nil {
		return ""
	}
	raw, ok := m[key]
	if !ok || string(raw) == "null" {
		return ""
	}
	var v string
	if json.Unmarshal(raw, &v) == nil {
		return v
	}
	return ""
}

func jsonBool(m map[string]json.RawMessage, key string) bool {
	if m == nil {
		return false
	}
	raw, ok := m[key]
	if !ok || string(raw) == "null" {
		return false
	}
	var v bool
	if json.Unmarshal(raw, &v) == nil {
		return v
	}
	return false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
