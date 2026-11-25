package yandex

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/asmisnik/realty-parser/internal/model"
)

type Listing struct {
	Flat              *model.FlatInfo `json:"flat"`
	Title             string          `json:"title"`
	Description       string          `json:"description,omitempty"`
	Publication       string          `json:"publication,omitempty"`
	PricePerM2        int             `json:"price_per_m2,omitempty"`
	BuildingYear      int             `json:"building_year,omitempty"`
	CompletionYear    int             `json:"completion_year,omitempty"`
	SellerType        string          `json:"seller_type,omitempty"`
	Tags              []string        `json:"tags,omitempty"`
	ApartmentFeatures []string        `json:"apartment_features,omitempty"`
	BuildingFeatures  []string        `json:"building_features,omitempty"`
	Rent              *RentalTerms    `json:"rent_terms,omitempty"`
}

type RentalTerms struct {
	Deposit        *int   `json:"deposit_rub,omitempty"`
	Commission     *int   `json:"commission_percent,omitempty"`
	CommissionRUB  *int   `json:"commission_rub,omitempty"`
	InitialPayment *int   `json:"initial_payment_rub,omitempty"`
	Utilities      string `json:"utilities,omitempty"`
}

type cardSchema struct {
	Type          string                                          `json:"@type"`
	URL           string                                          `json:"url"`
	Category      string                                          `json:"category"`
	SKU           string                                          `json:"sku"`
	Description   string                                          `json:"description"`
	Image         []string                                        `json:"image"`
	Price         json.Number                                     `json:"price"`
	PriceCurrency string                                          `json:"priceCurrency"`
	Availability  string                                          `json:"availability"`
	NumberOfRooms *int                                            `json:"numberOfRooms"`
	FloorSize     struct{ Value float64 }                         `json:"floorSize"`
	FloorLevel    json.Number                                     `json:"floorLevel"`
	Geo           struct{ Latitude, Longitude float64 }           `json:"geo"`
	Address       struct{ AddressLocality, StreetAddress string } `json:"address"`
}

func readCardSchema(doc *html.Node, link string) (product, apartment, terms cardSchema) {
	for n := range doc.Descendants() {
		if n.Type != html.ElementNode || n.Data != "script" || attribute(n, "type") != "application/ld+json" {
			continue
		}
		var data struct {
			Graph []json.RawMessage `json:"@graph"`
		}
		if n.FirstChild == nil || json.Unmarshal([]byte(n.FirstChild.Data), &data) != nil {
			continue
		}
		for _, raw := range data.Graph {
			var item cardSchema
			if json.Unmarshal(raw, &item) != nil {
				continue
			}
			if item.Type == "Product" && listingURL("/offer/"+item.SKU+"/") == link {
				product = item
				continue
			}
			if listingURL(item.URL) != link {
				continue
			}
			switch item.Type {
			case "Apartment":
				apartment = item
			case "Offer":
				terms = item
			}
		}
	}
	return product, apartment, terms
}

var (
	cardRooms   = regexp.MustCompile(`(\d+)-комнатн`)
	cardNumber  = regexp.MustCompile(`-?\d+(?:[.,]\d+)?`)
	cardMoney   = regexp.MustCompile(`(\d[\d ]*)\s*₽`)
	cardPercent = regexp.MustCompile(`(\d+)\s*%`)
)

func parseListingHTML(doc, main, summary *html.Node, region int) (*Listing, error) {
	var link string
	for n := range doc.Descendants() {
		if n.Type == html.ElementNode && n.Data == "link" && attribute(n, "rel") == "canonical" {
			link = listingURL(attribute(n, "href"))
			break
		}
	}
	if link == "" {
		return nil, errors.New("saved listing has no canonical Yandex Realty URL")
	}
	product, apartment, terms := readCardSchema(doc, link)
	if terms.Availability != "" && !strings.HasSuffix(terms.Availability, "/InStock") {
		return nil, errors.New("Yandex Realty listing is unavailable")
	}
	priceText := nodeText(firstClass(summary, "OfferCardSummaryInfo__price"))
	dealType, err := cardDealType(main, priceText)
	if err != nil {
		return nil, err
	}
	data := &Listing{
		Title:             nodeText(firstTag(summary, "h1")),
		Description:       strings.TrimSpace(product.Description),
		Publication:       strings.TrimSpace(strings.SplitN(nodeText(firstClass(summary, "OfferCardSummaryHeader__text")), ",", 2)[0]),
		Tags:              textsByClass(summary, "OfferCardSummaryTags__badgeText"),
		ApartmentFeatures: textsByClass(main, "OfferCardDetailsFeatures__feature"),
		BuildingFeatures:  textsByClass(main, "OfferCardBuildingFeatures__feature"),
		SellerType:        nodeText(firstClass(summary, "OfferCardAuthorBadge__category")),
	}
	if data.SellerType == "" && firstClass(summary, "OfferCardAgencyAuthorBadge__logoWrapper") != nil {
		data.SellerType = "Агентство"
	}
	if price := rubles(nodeText(firstClass(summary, "OfferCardSummaryInfo__pricePerMeterExtra"))); price != nil {
		data.PricePerM2 = *price
	}
	if data.Description == "" {
		data.Description = nodeText(firstClass(main, "OfferCardTextDescription__text"))
	}
	id := strings.TrimSuffix(strings.TrimPrefix(link, realtyOrigin+"/offer/"), "/")
	item := offer{OfferID: json.RawMessage(strconv.Quote(id)), OfferCategory: product.Category, RoomsTotal: apartment.NumberOfRooms}
	if data.Title == "" {
		return nil, errors.New("saved listing has no title")
	}
	if item.RoomsTotal == nil {
		item.RoomsTotal = roomsFromTitle(data.Title)
	}
	item.Price.Value, _ = terms.Price.Float64()
	item.Price.Currency = terms.PriceCurrency
	if item.Price.Value == 0 {
		if price := rubles(priceText); price != nil {
			item.Price.Value = float64(*price)
		}
	}
	item.Area.Value = apartment.FloorSize.Value
	item.Floor = integer(apartment.FloorLevel.String())
	item.Location.Point.Latitude = apartment.Geo.Latitude
	item.Location.Point.Longitude = apartment.Geo.Longitude
	location := firstClass(summary, "OfferCardSummary__location")
	item.Location.Address = nodeText(firstClass(location, "CardLocation__addressItem"))
	if item.Location.Address == "" && apartment.Address.StreetAddress != "" {
		item.Location.Address = strings.Trim(strings.Join([]string{apartment.Address.AddressLocality, apartment.Address.StreetAddress}, ", "), ", ")
	}
	item.Location.MetroList = cardMetro(location)
	item.OrigImages = cardPhotos(main, product.Image)
	if updated := attribute(firstTag(firstClass(summary, "OfferCardSummaryHeader__text"), "time"), "datetime"); updated != "" {
		if _, err := time.Parse(time.RFC3339, updated); err == nil {
			item.UpdateDate = updated
		}
	}
	readHighlights(main, &item, data)
	readFeatures(&item, data)
	if item.NewFlatSale {
		item.Building.SiteName = cardNewBuilding(main)
	}
	if dealType == "rent" {
		item.Price.Period = "MONTH"
		data.Rent = readRentalTerms(summary)
		if data.Rent.Deposit != nil {
			item.RentPledge = *data.Rent.Deposit
		}
		if data.Rent.Commission != nil {
			item.AgentFee = *data.Rent.Commission
		}
	}
	data.Flat = item.flat(region, dealType)
	if data.Flat == nil {
		return nil, fmt.Errorf("listing %s has no valid apartment price, area or room count", id)
	}
	return data, nil
}

func cardDealType(main *html.Node, price string) (string, error) {
	price = strings.ToLower(price)
	if nav := firstClass(main, "OfferCard__breadcrumbs"); nav != nil {
		for n := range nav.Descendants() {
			if n.Type != html.ElementNode || n.Data != "a" {
				continue
			}
			u := realtyURL(attribute(n, "href"))
			if u == nil {
				continue
			}
			switch {
			case strings.Contains(u.Path, "/snyat/"):
				if strings.Contains(price, "в месяц") || strings.Contains(price, "/мес") || strings.Contains(price, "/ мес") {
					return "rent", nil
				}
				return "", errors.New("saved rental listing has no monthly price")
			case strings.Contains(u.Path, "/kupit/"):
				return "sale", nil
			}
		}
	}
	return "", errors.New("saved listing has no sale or rent breadcrumb")
}

func roomsFromTitle(title string) *int {
	title = strings.ToLower(title)
	if strings.Contains(title, "студия") {
		studio := 0
		return &studio
	}
	if m := cardRooms.FindStringSubmatch(title); m != nil {
		rooms := integer(m[1])
		return &rooms
	}
	return nil
}

func cardNewBuilding(main *html.Node) string {
	if nav := firstClass(main, "OfferCard__breadcrumbs"); nav != nil {
		for n := range nav.Descendants() {
			if n.Type == html.ElementNode && n.Data == "a" {
				if u := realtyURL(attribute(n, "href")); u != nil && strings.Contains(u.Path, "/zhk-") {
					return nodeText(n)
				}
			}
		}
	}
	return ""
}

func decimal(text string) float64 {
	v, _ := strconv.ParseFloat(strings.ReplaceAll(cardNumber.FindString(text), ",", "."), 64)
	return v
}

func integer(text string) int { return int(decimal(text)) }

func rubles(text string) *int {
	return parsedInt(cardMoney.FindStringSubmatch(text))
}

func parsedInt(match []string) *int {
	if len(match) == 2 {
		if value, err := strconv.Atoi(strings.ReplaceAll(match[1], " ", "")); err == nil {
			return &value
		}
	}
	return nil
}

func cardPhotos(main *html.Node, structured []string) []string {
	var gallery []string
	if root := firstClass(main, "OfferCard__gallery"); root != nil {
		for n := range root.Descendants() {
			if n.Type == html.ElementNode && n.Data == "img" && hasClass(n, "GalleryV2Slide__wrappedImage") {
				gallery = append(gallery, attribute(n, "src"))
			}
		}
	}
	var images []string
	seen := make(map[string]bool)
	for _, raw := range append(gallery, structured...) {
		if strings.HasPrefix(raw, "//") {
			raw = "https:" + raw
		}
		if !isImageURL(raw) {
			continue
		}
		u, _ := url.Parse(raw)
		key := raw
		if u.Hostname() == "avatars.mds.yandex.net" && strings.HasPrefix(u.Path, "/get-realty-offers/") {
			key = u.Hostname() + u.Path[:strings.LastIndex(u.Path, "/")]
		}
		if !seen[key] {
			seen[key] = true
			images = append(images, raw)
		}
	}
	return images
}
