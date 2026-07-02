package model

type FlatInfo struct {
	Link                     string
	Region                   int
	DealType                 string
	Price                    int
	FlatScore                int
	Latitude                 float64 // 0 if CIAN didn't report coordinates for this listing
	Longitude                float64
	UndergroundScore         float64
	UndergroundPlace         int
	UndergroundDistanceInfo  string
	UndergroundStations      []string
	RoomNumber               int
	TotalArea                float64
	LivingArea               float64
	KitchenArea              float64
	Floor                    int
	MaxFloor                 int
	Deposit                  int
	DepositMonths            int
	Comission                int
	Renovation               string
	IsApartments             bool
	LoggiaCount              int
	BalconyCount             int
	WindowsView              string
	SeparatedBathroomCount   int
	CombinedBathroomCount    int
	HasDishwasher            bool
	HasConditioner           bool
	ChildrenAllowed          bool
	PetsAllowed              bool
	LastUpdated              string
	CeilingHeight            float64
	BuildingEntrancesNumber  int
	BuildingApartmentsNumber int
	BuildingElevatorsNumber  int

	// Sale-only fields (zero-valued for rent listings).
	SaleType                  string // "free" (свободная продажа) or "alternative" (альтернативная)
	MortgageAllowed           bool
	IsNewBuilding             bool
	NewBuildingName           string
	IsByHomeowner             bool
	DemolishedInMoscowProgram bool
}
