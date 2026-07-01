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
	logger.Debug("search page received", zap.Int("body_bytes", len(body)))

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
	logger.Debug("LinkArea nodes found", zap.Int("count", len(linkAreaNodes)))

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
			logger.Debug("skipping odd LinkArea node", zap.Int("index", i))
			continue
		}
		href, classFound := extractHrefDebug(node)
		// Strip tracking query params — store only canonical flat URL
		if q := strings.IndexByte(href, '?'); q != -1 {
			href = href[:q]
		}
		logger.Debug("LinkArea node processed",
			zap.Int("index", i),
			zap.String("href", href),
			zap.Bool("link_class_matched", classFound),
		)
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
	logger.Debug("flat page received", zap.String("href", href), zap.Int("body_bytes", len(body)))

	idx := strings.Index(body, "window._cianConfig['frontend-offer-card']")
	if idx == -1 {
		logger.Warn("offer card JS not found in flat page",
			zap.String("href", href),
			zap.String("body_preview", truncate(body, 300)),
		)
		return nil, fmt.Errorf("offer card data not found in page")
	}
	logger.Debug("offer card JS found", zap.String("href", href), zap.Int("offset", idx))

	end := strings.IndexByte(body[idx:], '\n')
	var line string
	if end == -1 {
		line = body[idx:]
	} else {
		line = body[idx : idx+end]
	}
	line = strings.TrimRight(line, "\r")
	logger.Debug("offer card line extracted", zap.String("href", href), zap.Int("line_bytes", len(line)))

	info, err := parseOfferCardLine(line, href, region, dealType)
	if err != nil {
		logger.Warn("offer card line parse failed", zap.String("href", href), zap.Error(err))
		return nil, err
	}
	logger.Info("flat page parsed",
		zap.String("href", href),
		zap.Int("price", info.Price),
		zap.Int("score", info.FlatScore),
		zap.Int("rooms", info.RoomNumber),
		zap.Float64("area", info.TotalArea),
	)
	return info, nil
}

func parseOfferCardLine(line, href string, region int, dealType string) (*model.FlatInfo, error) {
	if len(line) < len(cianConfigPrefix)+2 {
		return nil, fmt.Errorf("offer card line too short (%d bytes)", len(line))
	}
	trimmed := line[:len(line)-2]
	jsonStr := trimmed[len(cianConfigPrefix):]

	var items []json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &items); err != nil {
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
	bargainTerms := nestedMap(offer, "bargainTerms")
	building := nestedMap(offer, "building")

	newbuilding := nestedMap(offer, "newbuilding")

	// Comission represents the agent's percentage fee for both deal types,
	// but CIAN structures it differently: a flat "agentFee" for rent vs a
	// nested "agentBonus": {paymentType, value} object for sale.
	comission := jsonInt(bargainTerms, "agentFee")
	if dealType == "sale" {
		agentBonus := nestedMap(bargainTerms, "agentBonus")
		comission = jsonInt(agentBonus, "value")
	}

	// The underground (metro) ranking data only covers Moscow; skip it for
	// other regions instead of scoring against Moscow-only station data.
	var undergroundScore float64
	var undergroundPlace int
	var undergroundDistInfo string
	if region == moscowRegionID {
		geo := nestedMap(offer, "geo")
		undergroundsRaw := jsonArray(geo, "undergrounds")
		undergroundScore, undergroundPlace, undergroundDistInfo = scoring.ParseUndergroundInfo(undergroundsRaw)
	}

	info := &model.FlatInfo{
		Link:                     href,
		Region:                   region,
		DealType:                 dealType,
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
		LastUpdated:              jsonString(offer, "humanizedEditDate"),
		CeilingHeight:            jsonFloat(building, "ceilingHeight"),
		BuildingEntrancesNumber:  jsonInt(houseData, "entrances"),
		BuildingApartmentsNumber: jsonInt(houseData, "flatCount"),
		BuildingElevatorsNumber:  jsonInt(houseData, "lifts"),
		UndergroundScore:         undergroundScore,
		UndergroundPlace:         undergroundPlace,
		UndergroundDistanceInfo:  undergroundDistInfo,

		SaleType:                  jsonString(bargainTerms, "saleType"),
		MortgageAllowed:           jsonBool(bargainTerms, "mortgageAllowed"),
		IsNewBuilding:             jsonInt(newbuilding, "id") != 0,
		NewBuildingName:           jsonString(newbuilding, "name"),
		IsByHomeowner:             jsonBool(offer, "isByHomeowner"),
		DemolishedInMoscowProgram: jsonBool(offer, "demolishedInMoscowProgramm"),
	}
	info.FlatScore = scoring.CalculateFlatScore(info)
	return info, nil
}

// JSON navigation helpers

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
