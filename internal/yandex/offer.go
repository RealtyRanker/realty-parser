package yandex

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/asmisnik/realty-parser/internal/model"
	"github.com/asmisnik/realty-parser/internal/scoring"
)

type area struct {
	Value float64
}

type metro struct {
	Name           string
	TimeToMetro    int
	MetroTransport string
}

type offer struct {
	OfferID       json.RawMessage
	ID            json.RawMessage
	OfferType     string
	OfferCategory string
	Active        *bool
	Obsolete      bool
	RoomsTotal    *int
	RoomsTotalKey string
	Rooms         *int
	Floor         int
	FloorsOffered []int
	FloorsTotal   int
	Area          area
	LivingSpace   area
	KitchenSpace  area
	CeilingHeight float64
	Price         struct {
		Value         float64
		ValueForWhole float64
		Currency      string
		Period        string
		Unit          string
	}
	Location struct {
		Address string
		Point   struct {
			Latitude  float64
			Longitude float64
		}
		Metro     metro
		MetroList []metro
	}
	Apartment struct {
		Renovation       string
		BalconiesCount   int
		LoggiasCount     int
		SeparateWcsCount int
		CombinedWcsCount int
		HasDishwasher    bool
		HasAircondition  bool
	}
	House struct {
		WindowView string
		Apartments bool
	}
	Building struct {
		FloorsCount     int
		SiteName        string
		EntrancesCount  int
		ApartmentsCount int
		ElevatorsCount  int
	}
	Author struct {
		Category string
	}
	RentPledge      int
	AgentFee        int
	WithChildren    bool
	WithPets        bool
	HasDishwasher   bool
	HasAircondition bool
	Mortgage        bool
	DealStatus      string
	FlatType        string
	NewFlatSale     bool
	UpdateDate      string
	CreationDate    string
	OrigImages      []string
	FullImages      []string
}

func parseSearch(body []byte, region int, dealType string) ([]*model.FlatInfo, error) {
	body = bytes.TrimSpace(body)
	if bytes.HasPrefix(body, []byte("<")) &&
		(bytes.Contains(body, []byte("showcaptcha")) || bytes.Contains(body, []byte("Вы не робот?"))) {
		return nil, ErrBlocked
	}
	var envelope struct {
		Response struct {
			Search struct {
				Offers json.RawMessage
			}
		}
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decoding Yandex Realty search response: %w", err)
	}
	raw := bytes.TrimSpace(envelope.Response.Search.Offers)
	var entries []json.RawMessage
	switch {
	case bytes.HasPrefix(raw, []byte("[")):
		if err := json.Unmarshal(raw, &entries); err != nil {
			return nil, err
		}
	case bytes.HasPrefix(raw, []byte("{")):
		var result struct{ Entities []json.RawMessage }
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, err
		}
		if result.Entities == nil {
			return nil, errors.New("Yandex Realty search offers have no entities array")
		}
		entries = result.Entities
	default:
		return nil, errors.New("Yandex Realty response has no search offers")
	}

	flats := make([]*model.FlatInfo, 0, len(entries))
	seen := make(map[string]bool)
	decoded := 0
	for _, entry := range entries {
		var wrapped struct{ Offer json.RawMessage }
		if json.Unmarshal(entry, &wrapped) != nil {
			continue
		}
		if len(wrapped.Offer) > 0 {
			entry = wrapped.Offer
		}
		var item offer
		if json.Unmarshal(entry, &item) != nil || item.link() == "" {
			continue
		}
		decoded++
		if f := item.flat(region, dealType); f != nil && !seen[f.Link] {
			flats = append(flats, f)
			seen[f.Link] = true
		}
	}
	if len(entries) > 0 && decoded == 0 {
		return nil, errors.New("Yandex Realty response contained no recognizable offers")
	}
	return flats, nil
}

func (o *offer) link() string {
	// Decode the identifier separately so JSON numbers never pass through float64.
	id := strings.Trim(string(o.OfferID), "\"")
	if id == "" {
		id = strings.Trim(string(o.ID), "\"")
	}
	if _, err := strconv.ParseUint(id, 10, 64); err != nil {
		return ""
	}
	return "https://realty.yandex.ru/offer/" + id + "/"
}

func (o *offer) flat(region int, dealType string) *model.FlatInfo {
	if (o.Active != nil && !*o.Active) || o.Obsolete ||
		(o.OfferCategory != "" && o.OfferCategory != "APARTMENT") {
		return nil
	}
	wantType := "RENT"
	if dealType == "sale" {
		wantType = "SELL"
	}
	if o.OfferType != "" && o.OfferType != wantType {
		return nil
	}
	price := o.Price.Value
	if o.Price.ValueForWhole > 0 {
		price = o.Price.ValueForWhole
	} else if o.Price.Unit != "" && o.Price.Unit != "WHOLE_OFFER" {
		return nil
	}
	switch o.Price.Currency {
	case "", "RUR", "RUB":
	default:
		return nil
	}
	if dealType == "rent" {
		switch o.Price.Period {
		case "MONTH", "MONTHLY":
		default:
			return nil
		}
	}
	rooms := o.RoomsTotal
	if rooms == nil {
		rooms = o.Rooms
	}
	if rooms == nil {
		n, err := strconv.Atoi(o.RoomsTotalKey)
		if o.RoomsTotalKey != "STUDIO" && err != nil {
			return nil
		}
		rooms = &n
	}
	if price < 1 || price >= float64(math.MaxInt64) || o.Area.Value <= 0 || *rooms < 0 {
		return nil
	}

	f := &model.FlatInfo{
		Link: o.link(), Region: region, DealType: dealType, Address: o.Location.Address,
		Price: int(math.Round(price)), RoomNumber: *rooms,
		TotalArea: o.Area.Value, LivingArea: o.LivingSpace.Value, KitchenArea: o.KitchenSpace.Value,
		Floor: o.Floor, MaxFloor: o.FloorsTotal, CeilingHeight: o.CeilingHeight,
		Latitude: o.Location.Point.Latitude, Longitude: o.Location.Point.Longitude,
		IsApartments: o.House.Apartments, Renovation: renovation(o.Apartment.Renovation),
		WindowsView:  windowsView(o.House.WindowView),
		BalconyCount: o.Apartment.BalconiesCount, LoggiaCount: o.Apartment.LoggiasCount,
		SeparatedBathroomCount: o.Apartment.SeparateWcsCount, CombinedBathroomCount: o.Apartment.CombinedWcsCount,
		HasDishwasher:           o.Apartment.HasDishwasher || o.HasDishwasher,
		HasConditioner:          o.Apartment.HasAircondition || o.HasAircondition,
		BuildingEntrancesNumber: o.Building.EntrancesCount, BuildingApartmentsNumber: o.Building.ApartmentsCount,
		BuildingElevatorsNumber: o.Building.ElevatorsCount,
		LastUpdated:             o.UpdateDate,
	}
	if len(o.FloorsOffered) == 1 {
		f.Floor = o.FloorsOffered[0]
	}
	if f.MaxFloor == 0 {
		f.MaxFloor = o.Building.FloorsCount
	}
	if f.LastUpdated == "" {
		f.LastUpdated = o.CreationDate
	}
	f.PhotoURLs, f.PhotoThumbnailURLs = photos(o.OrigImages, o.FullImages)
	f.Comission = o.AgentFee
	if dealType == "rent" {
		f.Deposit = o.RentPledge
		f.ChildrenAllowed, f.PetsAllowed = o.WithChildren, o.WithPets
	} else {
		f.MortgageAllowed = o.Mortgage
		f.IsNewBuilding = o.NewFlatSale || o.FlatType == "NEW" || o.FlatType == "NEW_BUILDING"
		f.NewBuildingName = o.Building.SiteName
		f.IsByHomeowner = o.Author.Category == "OWNER"
		switch o.DealStatus {
		case "SALE":
			f.SaleType = "free"
		case "COUNTERSALE":
			f.SaleType = "alternative"
		}
	}
	metroList := o.Location.MetroList
	if len(metroList) == 0 {
		metroList = []metro{o.Location.Metro}
	}
	var stations []scoring.UndergroundStation
	for _, m := range metroList {
		if m.Name == "" || m.TimeToMetro <= 0 || m.MetroTransport == "" {
			continue
		}
		travel := "transport"
		if m.MetroTransport == "ON_FOOT" {
			travel = "walk"
		}
		stations = append(stations, scoring.UndergroundStation{Name: m.Name, TravelTime: m.TimeToMetro, TravelType: travel})
	}
	score, place, info, names := scoring.ScoreUnderground(stations)
	f.UndergroundDistanceInfo, f.UndergroundStations = info, names
	if region == 1 {
		f.UndergroundScore, f.UndergroundPlace = score, place
	}
	f.FlatScore = scoring.CalculateFlatScore(f)
	return f
}

func photos(original, reduced []string) (full, thumbnails []string) {
	for i := 0; i < max(len(original), len(reduced)); i++ {
		var image, thumbnail string
		if i < len(original) && isImageURL(original[i]) {
			image = original[i]
		}
		if i < len(reduced) && isImageURL(reduced[i]) {
			thumbnail = reduced[i]
		}
		if image == "" {
			image = thumbnail
		}
		if thumbnail == "" {
			thumbnail = image
		}
		if image != "" {
			full, thumbnails = append(full, image), append(thumbnails, thumbnail)
		}
	}
	return full, thumbnails
}

func isImageURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Hostname() != ""
}

func renovation(value string) string {
	switch value {
	case "COSMETIC":
		return "cosmetic"
	case "RENOVATED":
		return "renovated"
	case "EURO":
		return "euro"
	case "DESIGNER", "DESIGN":
		return "design"
	case "NO_REPAIR", "NEEDS_RENOVATION":
		return "none"
	default:
		return ""
	}
}

func windowsView(value string) string {
	switch value {
	case "YARD":
		return "yard"
	case "STREET":
		return "street"
	case "BOTH", "YARD_AND_STREET":
		return "both"
	default:
		return ""
	}
}
