package scoring

import (
	"math"

	"github.com/asmisnik/realty-parser/internal/model"
)

// Scoring multipliers (ported directly from flat_score.hpp)
const (
	defaultScore             = 600000.0
	depositMultiplier        = 0.02
	allAreaMultiplier        = 300.0
	costMultiplier           = 1.0
	maxPossiblePrice         = 300000
	livingAreaMultiplier     = 600.0
	livingAreaPart           = 0.5
	kitchenAreaMultiplier    = 50.0
	maxKitchenArea           = 50.0
	defaultKitchenArea       = 5.0
	comissionMultiplier      = 0.0006
	prepaidMonthsMultiplier  = 3000.0
	petsMultiplier           = 4000.0
	dishwasherMultiplier     = 1500.0
	conditionerMultiplier    = 1500.0
	apartmentsMultiplier     = 3000.0
	twoRoomMultiplier        = 3000.0
	threeRoomMultiplier      = 5000.0
	fourRoomMultiplier       = 7500.0
	additionalRoomsMultipier = 1500.0
	windowsYardMultiplier    = 400.0
	windowsStreetMultiplier  = 200.0
	windowsBothMultiplier    = 500.0
	renovationCosmeticMult   = 3000.0
	renovationEuroMult       = 5500.0
	renovationDesignMult     = 7000.0
	bathroomSeparatedMult    = 1000.0
	balconyMultiplier        = 1500.0
	loggiaMultiplier         = 2500.0
	defaultUndergroundScore  = 50.0
	undergroundScoreMult     = 1000.0
	eps                      = 1e-5
)

// CalculateFlatScore computes a composite desirability score for a flat.
func CalculateFlatScore(f *model.FlatInfo) int {
	score := defaultScore
	score -= depositMultiplier * float64(f.Deposit)
	score += allAreaMultiplier * f.TotalArea
	score += costScore(f)
	score += livingAreaScore(f)
	score += kitchenAreaScore(f)
	score -= comissionMultiplier * float64(f.Comission) * float64(f.Price)
	score -= prepaidMonthsMultiplier * float64(f.DepositMonths)
	if f.PetsAllowed {
		score += petsMultiplier
	}
	if f.HasDishwasher {
		score += dishwasherMultiplier
	}
	if f.HasConditioner {
		score += conditionerMultiplier
	}
	if f.IsApartments {
		score += apartmentsMultiplier
	}
	score += roomsScore(f)
	score += windowsScore(f)
	score += renovationScore(f)
	score += bathroomScore(f)
	score += math.Max(balconyMultiplier*float64(f.BalconyCount), loggiaMultiplier*float64(f.LoggiaCount))
	score += undergroundScoreMult * (defaultUndergroundScore - f.UndergroundScore)
	return int(score)
}

func costScore(f *model.FlatInfo) float64 {
	if f.Price == 0 || f.Price > maxPossiblePrice {
		return -costMultiplier * float64(f.Deposit)
	}
	return -costMultiplier * float64(f.Price)
}

func livingAreaScore(f *model.FlatInfo) float64 {
	if math.Abs(f.LivingArea) < eps {
		return livingAreaMultiplier * livingAreaPart * f.TotalArea
	}
	return livingAreaMultiplier * f.LivingArea
}

func kitchenAreaScore(f *model.FlatInfo) float64 {
	if math.Abs(f.KitchenArea) < eps || f.KitchenArea > maxKitchenArea {
		return kitchenAreaMultiplier * defaultKitchenArea
	}
	return kitchenAreaMultiplier * f.KitchenArea
}

func roomsScore(f *model.FlatInfo) float64 {
	switch {
	case f.RoomNumber <= 1:
		return 0
	case f.RoomNumber == 2:
		return twoRoomMultiplier
	case f.RoomNumber == 3:
		return threeRoomMultiplier
	case f.RoomNumber == 4:
		return fourRoomMultiplier
	default:
		return fourRoomMultiplier + float64(f.RoomNumber)*additionalRoomsMultipier
	}
}

func windowsScore(f *model.FlatInfo) float64 {
	switch f.WindowsView {
	case "yard":
		return windowsYardMultiplier
	case "street":
		return windowsStreetMultiplier
	case "":
		return 0
	default:
		return windowsBothMultiplier
	}
}

func renovationScore(f *model.FlatInfo) float64 {
	switch f.Renovation {
	case "cosmetic":
		return renovationCosmeticMult
	case "euro":
		return renovationEuroMult
	case "design":
		return renovationDesignMult
	default:
		return 0
	}
}

func bathroomScore(f *model.FlatInfo) float64 {
	return math.Max(0, bathroomSeparatedMult*float64(f.SeparatedBathroomCount))
}
