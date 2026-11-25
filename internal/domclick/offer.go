package domclick

import (
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/asmisnik/realty-parser/internal/model"
	"github.com/asmisnik/realty-parser/internal/scoring"
)

type offer struct {
	ID         int64
	Path       string
	DealType   string
	OfferType  string
	Status     int
	IsSold     bool
	Price      int
	IsOwner    bool
	ObjectInfo struct {
		Area        float64
		Rooms       *int
		Floor       int
		IsApartment bool
	}
	House       struct{ Floors int }
	FlatComplex struct{ Name string }
	Location    struct{ Lat, Lon float64 }
	Address     struct {
		DisplayName string
		Subways     []struct {
			Name                string
			IsUnderConstruction bool
			Remoteness          struct{ Time, Kind int }
		}
	}
	Photos        []struct{ URL string }
	UpdatedDate   string
	PublishedDate string
}

func (o offer) flat(region int, dealType string) *model.FlatInfo {
	if o.Status != 2 || o.IsSold || o.DealType != dealType || o.Price <= 0 ||
		o.ObjectInfo.Area <= 0 || o.ObjectInfo.Rooms == nil || *o.ObjectInfo.Rooms < 0 {
		return nil
	}
	link := o.listingURL()
	if link == "" {
		return nil
	}
	f := &model.FlatInfo{
		Link: link, Region: region, DealType: dealType, Price: o.Price,
		Address: o.Address.DisplayName, Latitude: o.Location.Lat, Longitude: o.Location.Lon,
		TotalArea: o.ObjectInfo.Area, RoomNumber: *o.ObjectInfo.Rooms,
		Floor: o.ObjectInfo.Floor, MaxFloor: o.House.Floors, IsApartments: o.ObjectInfo.IsApartment,
		LastUpdated: o.UpdatedDate,
	}
	if f.LastUpdated == "" {
		f.LastUpdated = o.PublishedDate
	}
	if dealType == "sale" {
		f.IsNewBuilding = o.OfferType == "new_flat" || o.OfferType == "layout"
		if f.IsNewBuilding {
			f.NewBuildingName = o.FlatComplex.Name
		}
		f.IsByHomeowner = o.IsOwner
	}
	var stations []scoring.UndergroundStation
	for _, subway := range o.Address.Subways {
		if subway.IsUnderConstruction || subway.Name == "" || subway.Remoteness.Time <= 0 {
			continue
		}
		var travel string
		switch subway.Remoteness.Kind {
		case 1:
			travel = "walk"
		case 2:
			travel = "transport"
		default:
			continue
		}
		stations = append(stations, scoring.UndergroundStation{
			Name: subway.Name, TravelTime: subway.Remoteness.Time, TravelType: travel,
		})
	}
	score, place, distance, names := scoring.ScoreUnderground(stations)
	f.UndergroundDistanceInfo, f.UndergroundStations = distance, names
	if region == 1 {
		f.UndergroundScore, f.UndergroundPlace = score, place
	}
	seenPhotos := make(map[string]bool)
	for _, photo := range o.Photos {
		if link := photoURL(photo.URL); link != "" && !seenPhotos[link] {
			f.PhotoURLs = append(f.PhotoURLs, link)
			seenPhotos[link] = true
		}
	}
	f.FlatScore = scoring.CalculateFlatScore(f)
	return f
}

func (o offer) listingURL() string {
	switch o.OfferType {
	case "flat", "new_flat", "layout":
	default:
		return ""
	}
	if o.DealType == "rent" && o.OfferType != "flat" {
		return ""
	}
	u, err := url.Parse(o.Path)
	if err != nil || u.Scheme != "https" || u.User != nil {
		return ""
	}
	if u.Host != "domclick.ru" && !strings.HasSuffix(u.Host, ".domclick.ru") {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(strings.TrimSuffix(u.Path, "/"), "/card/"), "__")
	if !strings.HasPrefix(u.Path, "/card/") || len(parts) != 3 || parts[0] != o.DealType || parts[2] != strconv.FormatInt(o.ID, 10) {
		return ""
	}
	switch parts[1] {
	case "flat", "new_flat", "layout":
		return "https://domclick.ru/card/" + strings.Join(parts, "__")
	default:
		return ""
	}
}

func photoURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil {
		return ""
	}
	if u.Scheme == "https" && u.Host == "img.dmclk.ru" {
		return u.String()
	}
	if u.IsAbs() || u.Host != "" || !strings.HasPrefix(u.Path, "/") {
		return ""
	}
	extension := path.Ext(u.Path)
	switch strings.ToLower(extension) {
	case ".jpg", ".jpeg", ".png", ".webp":
		return (&url.URL{Scheme: "https", Host: "img.dmclk.ru", Path: "/s1200x800q80" + strings.TrimSuffix(u.Path, extension) + ".webp"}).String()
	default:
		return ""
	}
}
