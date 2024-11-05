package gofakeit

import "math/rand"

// CarInfo is a struct dataset of all car information
type CarInfo struct {
	Type         string `json:"type" xml:"type"`
	Fuel         string `json:"fuel" xml:"fuel"`
	Transmission string `json:"transmission" xml:"transmission"`
	Brand        string `json:"brand" xml:"brand"`
	Model        string `json:"model" xml:"model"`
	Year         int    `json:"year" xml:"year"`
}

// Car will generate a struct with car information
func Car() *CarInfo { return car(globalFaker.Rand) }

// Car will generate a struct with car information
func (f *Faker) Car() *CarInfo { return car(f.Rand) }

func car(r *rand.Rand) *CarInfo {
	return &CarInfo{
		Type:         carType(r),
		Fuel:         carFuelType(r),
		Transmission: carTransmissionType(r),
		Brand:        carMaker(r),
		Model:        carModel(r),
		Year:         year(r),
	}
}

// CarType will generate a random car type string
func CarType() string { return carType(globalFaker.Rand) }

// CarType will generate a random car type string
func (f *Faker) CarType() string { return carType(f.Rand) }

func carType(r *rand.Rand) string { return getRandValue(r, []string{"car", "type"}) }

// CarFuelType will return a random fuel type
func CarFuelType() string { return carFuelType(globalFaker.Rand) }

// CarFuelType will return a random fuel type
func (f *Faker) CarFuelType() string { return carFuelType(f.Rand) }

func carFuelType(r *rand.Rand) string { return getRandValue(r, []string{"car", "fuel_type"}) }

// CarTransmissionType will return a random transmission type
func CarTransmissionType() string { return carTransmissionType(globalFaker.Rand) }

// CarTransmissionType will return a random transmission type
func (f *Faker) CarTransmissionType() string { return carTransmissionType(f.Rand) }

func carTransmissionType(r *rand.Rand) string {
	return getRandValue(r, []string{"car", "transmission_type"})
}

// CarMaker will return a random car maker
func CarMaker() string { return carMaker(globalFaker.Rand) }

// CarMaker will return a random car maker
func (f *Faker) CarMaker() string { return carMaker(f.Rand) }

func carMaker(r *rand.Rand) string { return getRandValue(r, []string{"car", "maker"}) }

// CarModel will return a random car model
func CarModel() string { return carModel(globalFaker.Rand) }

// CarModel will return a random car model
func (f *Faker) CarModel() string { return carModel(f.Rand) }

func carModel(r *rand.Rand) string { return getRandValue(r, []string{"car", "model"}) }

func addCarLookup() {
	AddFuncLookup("car", Info{
		Display:     "Car",
		Category:    "car",
		Description: "Wheeled motor vehicle used for transportation",
		Example: `{
	"type": "Passenger car mini",
	"fuel": "Gasoline",
	"transmission": "Automatic",
	"brand": "Fiat",
	"model": "Freestyle Fwd",
	"year": 1991
}`,
		Output:      "map[string]any",
		ContentType: "application/json",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return car(r), nil
		},
	})

	AddFuncLookup("cartype", Info{
		Display:     "Car Type",
		Category:    "car",
		Description: "Classification of cars based on size, use, or body style",
		Example:     "Passenger car mini",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return carType(r), nil
		},
	})

	AddFuncLookup("carfueltype", Info{
		Display:     "Car Fuel Type",
		Category:    "car",
		Description: "Type of energy source a car uses",
		Example:     "CNG",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return carFuelType(r), nil
		},
	})

	AddFuncLookup("cartransmissiontype", Info{
		Display:     "Car Transmission Type",
		Category:    "car",
		Description: "Mechanism a car uses to transmit power from the engine to the wheels",
		Example:     "Manual",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return carTransmissionType(r), nil
		},
	})

	AddFuncLookup("carmaker", Info{
		Display:     "Car Maker",
		Category:    "car",
		Description: "Company or brand that manufactures and designs cars",
		Example:     "Nissan",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return carMaker(r), nil
		},
	})

	AddFuncLookup("carmodel", Info{
		Display:     "Car Model",
		Category:    "car",
		Description: "Specific design or version of a car produced by a manufacturer",
		Example:     "Aveo",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return carModel(r), nil
		},
	})
}
