package scoring

import (
	"math"

	"github.com/asmisnik/realty-parser/internal/model"
)

// scoreConstants holds every tunable multiplier used by calculateScore. Rent
// and sale listings use different price magnitudes (monthly rent in the tens
// of thousands of rubles vs. a total sale price in the millions), so the two
// deal types need independently calibrated constants — see rentConstants and
// saleConstants below.
type scoreConstants struct {
	defaultScore             float64
	depositMultiplier        float64
	allAreaMultiplier        float64
	costMultiplier           float64
	maxPossiblePrice         float64
	livingAreaMultiplier     float64
	livingAreaPart           float64
	kitchenAreaMultiplier    float64
	maxKitchenArea           float64
	defaultKitchenArea       float64
	comissionMultiplier      float64
	prepaidMonthsMultiplier  float64
	petsMultiplier           float64
	dishwasherMultiplier     float64
	conditionerMultiplier    float64
	apartmentsMultiplier     float64
	twoRoomMultiplier        float64
	threeRoomMultiplier      float64
	fourRoomMultiplier       float64
	additionalRoomsMultipier float64
	windowsYardMultiplier    float64
	windowsStreetMultiplier  float64
	windowsBothMultiplier    float64
	renovationCosmeticMult   float64
	renovationEuroMult       float64
	renovationDesignMult     float64
	bathroomSeparatedMult    float64
	balconyMultiplier        float64
	loggiaMultiplier         float64
	defaultUndergroundScore  float64
	undergroundScoreMult     float64
}

// rentConstants are ported directly from the original flat_score.hpp, tuned
// for monthly rent prices (tens/hundreds of thousands of rubles).
var rentConstants = scoreConstants{
	defaultScore:             600000.0,
	depositMultiplier:        0.02,
	allAreaMultiplier:        300.0,
	costMultiplier:           1.0,
	maxPossiblePrice:         300000,
	livingAreaMultiplier:     600.0,
	livingAreaPart:           0.5,
	kitchenAreaMultiplier:    50.0,
	maxKitchenArea:           50.0,
	defaultKitchenArea:       5.0,
	comissionMultiplier:      0.0006,
	prepaidMonthsMultiplier:  3000.0,
	petsMultiplier:           4000.0,
	dishwasherMultiplier:     1500.0,
	conditionerMultiplier:    1500.0,
	apartmentsMultiplier:     3000.0,
	twoRoomMultiplier:        3000.0,
	threeRoomMultiplier:      5000.0,
	fourRoomMultiplier:       7500.0,
	additionalRoomsMultipier: 1500.0,
	windowsYardMultiplier:    400.0,
	windowsStreetMultiplier:  200.0,
	windowsBothMultiplier:    500.0,
	renovationCosmeticMult:   3000.0,
	renovationEuroMult:       5500.0,
	renovationDesignMult:     7000.0,
	bathroomSeparatedMult:    1000.0,
	balconyMultiplier:        1500.0,
	loggiaMultiplier:         2500.0,
	defaultUndergroundScore:  50.0,
	undergroundScoreMult:     1000.0,
}

// saleScaleMultiplier is a rough rent-to-sale conversion factor (roughly the
// number of months of rent a sale price represents, i.e. 1/cap-rate). It is
// NOT independently calibrated against real sale data — it's a reasoned
// starting point (~200 months ≈ typical Moscow gross rental yield of ~5-6%)
// so purely additive amenity/location bonuses stay proportionally meaningful
// next to a sale price that's ~200x larger than a monthly rent. Constants
// that already multiply against the actual price/commission (costMultiplier,
// comissionMultiplier) are deliberately left unscaled, since Price itself is
// already on the sale-price scale. Tune saleConstants directly once real
// sale outcomes are available to calibrate against.
const saleScaleMultiplier = 200.0

var saleConstants = scoreConstants{
	defaultScore:             rentConstants.defaultScore * saleScaleMultiplier,
	depositMultiplier:        rentConstants.depositMultiplier, // unused: sale listings have no deposit
	allAreaMultiplier:        rentConstants.allAreaMultiplier * saleScaleMultiplier,
	costMultiplier:           rentConstants.costMultiplier, // Price is already sale-scale; don't double-scale
	maxPossiblePrice:         rentConstants.maxPossiblePrice * saleScaleMultiplier,
	livingAreaMultiplier:     rentConstants.livingAreaMultiplier * saleScaleMultiplier,
	livingAreaPart:           rentConstants.livingAreaPart, // ratio, not a monetary constant
	kitchenAreaMultiplier:    rentConstants.kitchenAreaMultiplier * saleScaleMultiplier,
	maxKitchenArea:           rentConstants.maxKitchenArea,          // area in m², not monetary
	defaultKitchenArea:       rentConstants.defaultKitchenArea,      // area in m², not monetary
	comissionMultiplier:      rentConstants.comissionMultiplier,     // multiplies Price directly; already sale-scale
	prepaidMonthsMultiplier:  rentConstants.prepaidMonthsMultiplier, // unused: sale listings have no prepay months
	petsMultiplier:           rentConstants.petsMultiplier * saleScaleMultiplier,
	dishwasherMultiplier:     rentConstants.dishwasherMultiplier * saleScaleMultiplier,
	conditionerMultiplier:    rentConstants.conditionerMultiplier * saleScaleMultiplier,
	apartmentsMultiplier:     rentConstants.apartmentsMultiplier * saleScaleMultiplier,
	twoRoomMultiplier:        rentConstants.twoRoomMultiplier * saleScaleMultiplier,
	threeRoomMultiplier:      rentConstants.threeRoomMultiplier * saleScaleMultiplier,
	fourRoomMultiplier:       rentConstants.fourRoomMultiplier * saleScaleMultiplier,
	additionalRoomsMultipier: rentConstants.additionalRoomsMultipier * saleScaleMultiplier,
	windowsYardMultiplier:    rentConstants.windowsYardMultiplier * saleScaleMultiplier,
	windowsStreetMultiplier:  rentConstants.windowsStreetMultiplier * saleScaleMultiplier,
	windowsBothMultiplier:    rentConstants.windowsBothMultiplier * saleScaleMultiplier,
	renovationCosmeticMult:   rentConstants.renovationCosmeticMult * saleScaleMultiplier,
	renovationEuroMult:       rentConstants.renovationEuroMult * saleScaleMultiplier,
	renovationDesignMult:     rentConstants.renovationDesignMult * saleScaleMultiplier,
	bathroomSeparatedMult:    rentConstants.bathroomSeparatedMult * saleScaleMultiplier,
	balconyMultiplier:        rentConstants.balconyMultiplier * saleScaleMultiplier,
	loggiaMultiplier:         rentConstants.loggiaMultiplier * saleScaleMultiplier,
	defaultUndergroundScore:  rentConstants.defaultUndergroundScore, // a 0-100ish rating, not monetary
	undergroundScoreMult:     rentConstants.undergroundScoreMult * saleScaleMultiplier,
}

const eps = 1e-5

// CalculateFlatScore computes a composite desirability score for a flat,
// using rent- or sale-calibrated constants depending on f.DealType.
func CalculateFlatScore(f *model.FlatInfo) int {
	if f.DealType == "sale" {
		return calculateScore(f, saleConstants)
	}
	return calculateScore(f, rentConstants)
}

func calculateScore(f *model.FlatInfo, c scoreConstants) int {
	score := c.defaultScore
	score -= c.depositMultiplier * float64(f.Deposit)
	score += c.allAreaMultiplier * f.TotalArea
	score += costScore(f, c)
	score += livingAreaScore(f, c)
	score += kitchenAreaScore(f, c)
	score -= c.comissionMultiplier * float64(f.Comission) * float64(f.Price)
	score -= c.prepaidMonthsMultiplier * float64(f.DepositMonths)
	if f.PetsAllowed {
		score += c.petsMultiplier
	}
	if f.HasDishwasher {
		score += c.dishwasherMultiplier
	}
	if f.HasConditioner {
		score += c.conditionerMultiplier
	}
	if f.IsApartments {
		score += c.apartmentsMultiplier
	}
	score += roomsScore(f, c)
	score += windowsScore(f, c)
	score += renovationScore(f, c)
	score += bathroomScore(f, c)
	score += math.Max(c.balconyMultiplier*float64(f.BalconyCount), c.loggiaMultiplier*float64(f.LoggiaCount))
	score += c.undergroundScoreMult * (c.defaultUndergroundScore - f.UndergroundScore)
	return int(score)
}

func costScore(f *model.FlatInfo, c scoreConstants) float64 {
	if f.Price == 0 || float64(f.Price) > c.maxPossiblePrice {
		return -c.costMultiplier * float64(f.Deposit)
	}
	return -c.costMultiplier * float64(f.Price)
}

func livingAreaScore(f *model.FlatInfo, c scoreConstants) float64 {
	if math.Abs(f.LivingArea) < eps {
		return c.livingAreaMultiplier * c.livingAreaPart * f.TotalArea
	}
	return c.livingAreaMultiplier * f.LivingArea
}

func kitchenAreaScore(f *model.FlatInfo, c scoreConstants) float64 {
	if math.Abs(f.KitchenArea) < eps || f.KitchenArea > c.maxKitchenArea {
		return c.kitchenAreaMultiplier * c.defaultKitchenArea
	}
	return c.kitchenAreaMultiplier * f.KitchenArea
}

func roomsScore(f *model.FlatInfo, c scoreConstants) float64 {
	switch {
	case f.RoomNumber <= 1:
		return 0
	case f.RoomNumber == 2:
		return c.twoRoomMultiplier
	case f.RoomNumber == 3:
		return c.threeRoomMultiplier
	case f.RoomNumber == 4:
		return c.fourRoomMultiplier
	default:
		return c.fourRoomMultiplier + float64(f.RoomNumber)*c.additionalRoomsMultipier
	}
}

func windowsScore(f *model.FlatInfo, c scoreConstants) float64 {
	switch f.WindowsView {
	case "yard":
		return c.windowsYardMultiplier
	case "street":
		return c.windowsStreetMultiplier
	case "":
		return 0
	default:
		return c.windowsBothMultiplier
	}
}

func renovationScore(f *model.FlatInfo, c scoreConstants) float64 {
	switch f.Renovation {
	case "cosmetic":
		return c.renovationCosmeticMult
	case "euro":
		return c.renovationEuroMult
	case "design":
		return c.renovationDesignMult
	default:
		return 0
	}
}

func bathroomScore(f *model.FlatInfo, c scoreConstants) float64 {
	return math.Max(0, c.bathroomSeparatedMult*float64(f.SeparatedBathroomCount))
}
