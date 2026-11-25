package yandex

import (
	"strings"

	"golang.org/x/net/html"
)

func readHighlights(main *html.Node, item *offer, data *Listing) {
	for _, highlight := range classNodes(main, "OfferCardHighlight__container") {
		value := nodeText(firstClass(highlight, "OfferCardHighlight__value"))
		label := nodeText(firstClass(highlight, "OfferCardHighlight__label"))
		switch {
		case label == "общая":
			if item.Area.Value == 0 {
				item.Area.Value = decimal(value)
			}
		case label == "жилая":
			item.LivingSpace.Value = decimal(value)
		case label == "кухня":
			item.KitchenSpace.Value = decimal(value)
		case strings.HasPrefix(label, "из ") && strings.Contains(value, "этаж"):
			item.Floor = integer(value)
			item.FloorsTotal = integer(label)
		case label == "потолки":
			item.CeilingHeight = decimal(value)
		case label == "год постройки":
			data.BuildingYear = integer(value)
		case label == "срок сдачи":
			data.CompletionYear = integer(value)
		}
	}
}

func readFeatures(item *offer, data *Listing) {
	item.House.Apartments = strings.Contains(strings.ToLower(data.Title), "апартамент")
	if strings.Contains(strings.ToLower(data.SellerType), "собственник") {
		item.Author.Category = "OWNER"
	}
	for _, tag := range data.Tags {
		switch strings.ToLower(tag) {
		case "ипотека":
			item.Mortgage = true
		case "свободная продажа":
			item.DealStatus = "SALE"
		case "альтернативная продажа":
			item.DealStatus = "COUNTERSALE"
		case "новостройка", "первичная продажа":
			item.NewFlatSale = true
		}
	}
	for _, feature := range data.ApartmentFeatures {
		value := strings.ToLower(feature)
		switch {
		case strings.HasSuffix(value, "₽ за м²"):
			if price := rubles(value); price != nil {
				data.PricePerM2 = *price
			}
		case strings.HasPrefix(value, "отделка"), strings.HasPrefix(value, "ремонт"):
			item.Apartment.Renovation = cardRenovation(value)
		case strings.HasPrefix(value, "санузел"):
			if strings.Contains(value, "раздельн") {
				item.Apartment.SeparateWcsCount = 1
			}
			if strings.Contains(value, "совмещ") {
				item.Apartment.CombinedWcsCount = 1
			}
		case strings.HasPrefix(value, "вид из окон"):
			yard := strings.Contains(value, "во двор")
			street := strings.Contains(value, "на улицу")
			switch {
			case yard && street:
				item.House.WindowView = "BOTH"
			case yard:
				item.House.WindowView = "YARD"
			case street:
				item.House.WindowView = "STREET"
			}
		case value == "посудомойка", value == "посудомоечная машина":
			item.Apartment.HasDishwasher = true
		case value == "кондиционер":
			item.Apartment.HasAircondition = true
		case value == "можно с детьми", value == "с детьми":
			item.WithChildren = true
		case value == "можно с животными", value == "с животными":
			item.WithPets = true
		default:
			if strings.Contains(value, "балкон") {
				item.Apartment.BalconiesCount = max(1, integer(value))
			}
			if strings.Contains(value, "лоджи") {
				item.Apartment.LoggiasCount = max(1, integer(value))
			}
		}
	}
	for _, feature := range data.BuildingFeatures {
		value := strings.ToLower(feature)
		count := integer(value)
		if count <= 0 {
			continue
		}
		switch {
		case strings.Contains(value, "подъезд"):
			item.Building.EntrancesCount = count
		case strings.Contains(value, "квартир"):
			item.Building.ApartmentsCount = count
		case strings.Contains(value, "лифт"):
			item.Building.ElevatorsCount += count
		case strings.Contains(value, "этаж"):
			item.Building.FloorsCount = count
		}
	}
}

func cardRenovation(value string) string {
	switch {
	case strings.Contains(value, "дизайнер"):
		return "DESIGNER"
	case strings.Contains(value, "евро"):
		return "EURO"
	case strings.Contains(value, "косметическ"):
		return "COSMETIC"
	case strings.Contains(value, "без отделки"), strings.Contains(value, "без ремонта"), strings.Contains(value, "требуется ремонт"):
		return "NO_REPAIR"
	case strings.Contains(value, "с ремонтом"):
		return "RENOVATED"
	default:
		return ""
	}
}

func readRentalTerms(summary *html.Node) *RentalTerms {
	terms := &RentalTerms{}
	for _, row := range classNodes(summary, "OfferCardCheck__row") {
		key := strings.ToLower(nodeText(firstClass(row, "OfferCardCheck__rowKey")))
		value := nodeText(firstClass(row, "OfferCardCheck__rowValue"))
		switch {
		case strings.HasPrefix(key, "залог"):
			terms.Deposit = rubles(value)
		case strings.HasPrefix(key, "комиссия"):
			badge := nodeText(firstClass(row, "OfferCardCheck__badge"))
			terms.Commission = parsedInt(cardPercent.FindStringSubmatch(badge))
			terms.CommissionRUB = rubles(value)
		case strings.HasPrefix(key, "коммунальные услуги"):
			terms.Utilities = value
		case strings.HasPrefix(key, "итого"):
			terms.InitialPayment = rubles(value)
		}
	}
	return terms
}

func cardMetro(location *html.Node) []metro {
	var stations []metro
	for _, node := range classNodes(location, "MetroWithTime") {
		name := nodeText(firstClass(node, "MetroWithTime__title"))
		minutes := integer(nodeText(firstClass(node, "MetroWithTime__time")))
		if name == "" || minutes <= 0 {
			continue
		}
		station := metro{Name: name, TimeToMetro: minutes}
		for n := range node.Descendants() {
			label := strings.ToLower(attribute(n, "aria-label") + " " + attribute(n, "title") + " " + attribute(n, "alt"))
			switch {
			case strings.Contains(label, "пешком"), hasClass(n, "Icon_type_small-pedestrian"):
				station.MetroTransport = "ON_FOOT"
			case strings.Contains(label, "на транспорте"), hasClass(n, "Icon_type_small-car"), hasClass(n, "Icon_type_small-bus"):
				station.MetroTransport = "TRANSPORT"
			}
		}
		stations = append(stations, station)
	}
	return stations
}
