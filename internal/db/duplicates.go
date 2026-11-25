package db

import (
	"context"
	"database/sql"
	"math"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/asmisnik/realty-parser/internal/model"
)

const areaTolerance = 0.5 // square metres, for differences in rounding

func platform(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	switch {
	case host == "cian.ru" || strings.HasSuffix(host, ".cian.ru"):
		return "cian"
	case host == "realty.yandex.ru" || host == "realty.ya.ru":
		return "yandex"
	case host == "domclick.ru" || strings.HasSuffix(host, ".domclick.ru"):
		return "domclick"
	default:
		return ""
	}
}

func findDuplicate(ctx context.Context, tx *sql.Tx, f *model.FlatInfo) (int, error) {
	if f.Price <= 0 || f.TotalArea <= 0 || f.Floor <= 0 || f.RoomNumber < 0 {
		return 0, nil
	}
	cutoff := time.Now().AddDate(0, 0, -30).UnixMicro()
	rows, err := tx.QueryContext(ctx, `
		SELECT id, link, address, latitude, longitude, max_floor, living_area, kitchen_area
		FROM flats_history
		WHERE region = ?1 AND deal_type = ?2 AND price = ?3
		  AND room_number = ?4 AND floor = ?5 AND total_area BETWEEN ?6 AND ?7
		  AND parsed_at >= ?8`,
		f.Region, f.DealType, f.Price, f.RoomNumber, f.Floor,
		f.TotalArea-areaTolerance, f.TotalArea+areaTolerance, cutoff)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	source := platform(f.Link)
	var match int
	for rows.Next() {
		var candidate model.FlatInfo
		var id int
		if err := rows.Scan(&id, &candidate.Link, &candidate.Address, &candidate.Latitude, &candidate.Longitude,
			&candidate.MaxFloor, &candidate.LivingArea, &candidate.KitchenArea); err != nil {
			return 0, err
		}
		other := platform(candidate.Link)
		if other == "" || other == source || !matchesApartment(f, &candidate) {
			continue
		}
		if match != 0 {
			// Two plausible counterparts are insufficient to pick a canonical listing.
			return 0, nil
		}
		match = id
	}
	return match, rows.Err()
}

func matchesApartment(a, b *model.FlatInfo) bool {
	if a.MaxFloor > 0 && b.MaxFloor > 0 && a.MaxFloor != b.MaxFloor {
		return false
	}
	livingKnown := a.LivingArea > 0 && b.LivingArea > 0
	kitchenKnown := a.KitchenArea > 0 && b.KitchenArea > 0
	livingDiff := math.Abs(a.LivingArea - b.LivingArea)
	kitchenDiff := math.Abs(a.KitchenArea - b.KitchenArea)
	if (livingKnown && livingDiff > 1) || (kitchenKnown && kitchenDiff > 1) {
		return false
	}
	if !(livingKnown && livingDiff <= areaTolerance) && !(kitchenKnown && kitchenDiff <= areaTolerance) {
		return false
	}

	addressA, addressB := buildingAddress(a.Address), buildingAddress(b.Address)
	distance := distanceMetres(a, b)
	if addressA != "" && addressB != "" {
		if addressA == addressB {
			return distance < 0 || distance <= 75
		}
		// Nearby buildings may have identical layouts. Respect house/corpus numbers.
		if addressA[strings.LastIndexByte(addressA, ','):] != addressB[strings.LastIndexByte(addressB, ','):] {
			return false
		}
	}
	return distance >= 0 && distance <= 25
}

var houseNumber = regexp.MustCompile(`^[0-9]+[0-9а-яa-z/]*$`)
var streetType = strings.NewReplacer("ул.", "улица ", "пр-т", "проспект", "пер.", "переулок ", "ш.", "шоссе ")

func buildingAddress(raw string) string {
	raw = strings.ReplaceAll(strings.ToLower(raw), "ё", "е")
	parts := strings.Split(raw, ",")
	if len(parts) > 0 && strings.TrimSpace(parts[0]) == "россия" {
		parts = parts[1:]
	}
	if len(parts) < 3 {
		return ""
	}
	house := strings.TrimSpace(parts[len(parts)-1])
	house = strings.TrimPrefix(strings.TrimPrefix(house, "д."), "дом ")
	house = strings.NewReplacer("корпус", "к", "корп.", "к", "стр.", "с", "строение", "с").Replace(house)
	house = strings.Join(strings.Fields(house), "")
	if !houseNumber.MatchString(house) {
		return ""
	}
	street := strings.Fields(streetType.Replace(parts[len(parts)-2]))
	// The services can place a street's name before or after its type.
	if len(street) > 1 && isStreetType(street[len(street)-1]) {
		street = append([]string{street[len(street)-1]}, street[:len(street)-1]...)
	}
	locality := strings.Join(strings.Fields(strings.Join(parts[:len(parts)-2], ",")), " ")
	if locality == "" || len(street) == 0 {
		return ""
	}
	return locality + "," + strings.Join(street, " ") + "," + house
}

func isStreetType(word string) bool {
	switch word {
	case "улица", "проспект", "переулок", "проезд", "шоссе", "бульвар", "набережная", "площадь":
		return true
	default:
		return false
	}
}

func distanceMetres(a, b *model.FlatInfo) float64 {
	if a.Latitude == 0 || a.Longitude == 0 || b.Latitude == 0 || b.Longitude == 0 {
		return -1
	}
	const radians = math.Pi / 180
	const metresPerDegree = 111195
	north := a.Latitude - b.Latitude
	east := (a.Longitude - b.Longitude) * math.Cos((a.Latitude+b.Latitude)*radians/2)
	return metresPerDegree * math.Hypot(north, east)
}
