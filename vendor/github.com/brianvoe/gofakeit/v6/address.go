package gofakeit

import (
	"errors"
	"math/rand"
	"strings"
)

// AddressInfo is a struct full of address information
type AddressInfo struct {
	Address   string  `json:"address" xml:"address"`
	Street    string  `json:"street" xml:"street"`
	City      string  `json:"city" xml:"city"`
	State     string  `json:"state" xml:"state"`
	Zip       string  `json:"zip" xml:"zip"`
	Country   string  `json:"country" xml:"country"`
	Latitude  float64 `json:"latitude" xml:"latitude"`
	Longitude float64 `json:"longitude" xml:"longitude"`
}

// Address will generate a struct of address information
func Address() *AddressInfo { return address(globalFaker.Rand) }

// Address will generate a struct of address information
func (f *Faker) Address() *AddressInfo { return address(f.Rand) }

func address(r *rand.Rand) *AddressInfo {
	street := street(r)
	city := city(r)
	state := state(r)
	zip := zip(r)

	return &AddressInfo{
		Address:   street + ", " + city + ", " + state + " " + zip,
		Street:    street,
		City:      city,
		State:     state,
		Zip:       zip,
		Country:   country(r),
		Latitude:  latitude(r),
		Longitude: longitude(r),
	}
}

// Street will generate a random address street string
func Street() string { return street(globalFaker.Rand) }

// Street will generate a random address street string
func (f *Faker) Street() string { return street(f.Rand) }

func street(r *rand.Rand) string {
	var street = ""
	switch randInt := randIntRange(r, 1, 2); randInt {
	case 1:
		street = streetNumber(r) + " " + streetPrefix(r) + " " + streetName(r) + streetSuffix(r)
	case 2:
		street = streetNumber(r) + " " + streetName(r) + streetSuffix(r)
	}

	return street
}

// StreetNumber will generate a random address street number string
func StreetNumber() string { return streetNumber(globalFaker.Rand) }

// StreetNumber will generate a random address street number string
func (f *Faker) StreetNumber() string { return streetNumber(f.Rand) }

func streetNumber(r *rand.Rand) string {
	return strings.TrimLeft(replaceWithNumbers(r, getRandValue(r, []string{"address", "number"})), "0")
}

// StreetPrefix will generate a random address street prefix string
func StreetPrefix() string { return streetPrefix(globalFaker.Rand) }

// StreetPrefix will generate a random address street prefix string
func (f *Faker) StreetPrefix() string { return streetPrefix(f.Rand) }

func streetPrefix(r *rand.Rand) string { return getRandValue(r, []string{"address", "street_prefix"}) }

// StreetName will generate a random address street name string
func StreetName() string { return streetName(globalFaker.Rand) }

// StreetName will generate a random address street name string
func (f *Faker) StreetName() string { return streetName(f.Rand) }

func streetName(r *rand.Rand) string { return getRandValue(r, []string{"address", "street_name"}) }

// StreetSuffix will generate a random address street suffix string
func StreetSuffix() string { return streetSuffix(globalFaker.Rand) }

// StreetSuffix will generate a random address street suffix string
func (f *Faker) StreetSuffix() string { return streetSuffix(f.Rand) }

func streetSuffix(r *rand.Rand) string { return getRandValue(r, []string{"address", "street_suffix"}) }

// City will generate a random city string
func City() string { return city(globalFaker.Rand) }

// City will generate a random city string
func (f *Faker) City() string { return city(f.Rand) }

func city(r *rand.Rand) string { return getRandValue(r, []string{"address", "city"}) }

// State will generate a random state string
func State() string { return state(globalFaker.Rand) }

// State will generate a random state string
func (f *Faker) State() string { return state(f.Rand) }

func state(r *rand.Rand) string { return getRandValue(r, []string{"address", "state"}) }

// StateAbr will generate a random abbreviated state string
func StateAbr() string { return stateAbr(globalFaker.Rand) }

// StateAbr will generate a random abbreviated state string
func (f *Faker) StateAbr() string { return stateAbr(f.Rand) }

func stateAbr(r *rand.Rand) string { return getRandValue(r, []string{"address", "state_abr"}) }

// Zip will generate a random Zip code string
func Zip() string { return zip(globalFaker.Rand) }

// Zip will generate a random Zip code string
func (f *Faker) Zip() string { return zip(f.Rand) }

func zip(r *rand.Rand) string {
	return replaceWithNumbers(r, getRandValue(r, []string{"address", "zip"}))
}

// Country will generate a random country string
func Country() string { return country(globalFaker.Rand) }

// Country will generate a random country string
func (f *Faker) Country() string { return country(f.Rand) }

func country(r *rand.Rand) string { return getRandValue(r, []string{"address", "country"}) }

// CountryAbr will generate a random abbreviated country string
func CountryAbr() string { return countryAbr(globalFaker.Rand) }

// CountryAbr will generate a random abbreviated country string
func (f *Faker) CountryAbr() string { return countryAbr(f.Rand) }

func countryAbr(r *rand.Rand) string { return getRandValue(r, []string{"address", "country_abr"}) }

// Latitude will generate a random latitude float64
func Latitude() float64 { return latitude(globalFaker.Rand) }

// Latitude will generate a random latitude float64
func (f *Faker) Latitude() float64 { return latitude(f.Rand) }

func latitude(r *rand.Rand) float64 { return toFixed((r.Float64()*180)-90, 6) }

// LatitudeInRange will generate a random latitude within the input range
func LatitudeInRange(min, max float64) (float64, error) {
	return latitudeInRange(globalFaker.Rand, min, max)
}

// LatitudeInRange will generate a random latitude within the input range
func (f *Faker) LatitudeInRange(min, max float64) (float64, error) {
	return latitudeInRange(f.Rand, min, max)
}

func latitudeInRange(r *rand.Rand, min, max float64) (float64, error) {
	if min > max || min < -90 || min > 90 || max < -90 || max > 90 {
		return 0, errors.New("invalid min or max range, must be valid floats and between -90 and 90")
	}
	return toFixed(float64Range(r, min, max), 6), nil
}

// Longitude will generate a random longitude float64
func Longitude() float64 { return longitude(globalFaker.Rand) }

// Longitude will generate a random longitude float64
func (f *Faker) Longitude() float64 { return longitude(f.Rand) }

func longitude(r *rand.Rand) float64 { return toFixed((r.Float64()*360)-180, 6) }

// LongitudeInRange will generate a random longitude within the input range
func LongitudeInRange(min, max float64) (float64, error) {
	return longitudeInRange(globalFaker.Rand, min, max)
}

// LongitudeInRange will generate a random longitude within the input range
func (f *Faker) LongitudeInRange(min, max float64) (float64, error) {
	return longitudeInRange(f.Rand, min, max)
}

func longitudeInRange(r *rand.Rand, min, max float64) (float64, error) {
	if min > max || min < -180 || min > 180 || max < -180 || max > 180 {
		return 0, errors.New("invalid min or max range, must be valid floats and between -180 and 180")
	}
	return toFixed(float64Range(r, min, max), 6), nil
}

func addAddressLookup() {
	AddFuncLookup("address", Info{
		Display:     "Address",
		Category:    "address",
		Description: "Residential location including street, city, state, country and postal code",
		Example: `{
	"address": "364 Unionsville, Norfolk, Ohio 99536",
	"street": "364 Unionsville",
	"city": "Norfolk",
	"state": "Ohio",
	"zip": "99536",
	"country": "Lesotho",
	"latitude": 88.792592,
	"longitude": 174.504681
}`,
		Output:      "map[string]any",
		ContentType: "application/json",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return address(r), nil
		},
	})

	AddFuncLookup("city", Info{
		Display:     "City",
		Category:    "address",
		Description: "Part of a country with significant population, often a central hub for culture and commerce",
		Example:     "Marcelside",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return city(r), nil
		},
	})

	AddFuncLookup("country", Info{
		Display:     "Country",
		Category:    "address",
		Description: "Nation with its own government and defined territory",
		Example:     "United States of America",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return country(r), nil
		},
	})

	AddFuncLookup("countryabr", Info{
		Display:     "Country Abbreviation",
		Category:    "address",
		Description: "Shortened 2-letter form of a country's name",
		Example:     "US",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return countryAbr(r), nil
		},
	})

	AddFuncLookup("state", Info{
		Display:     "State",
		Category:    "address",
		Description: "Governmental division within a country, often having its own laws and government",
		Example:     "Illinois",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return state(r), nil
		},
	})

	AddFuncLookup("stateabr", Info{
		Display:     "State Abbreviation",
		Category:    "address",
		Description: "Shortened 2-letter form of a country's state",
		Example:     "IL",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return stateAbr(r), nil
		},
	})

	AddFuncLookup("street", Info{
		Display:     "Street",
		Category:    "address",
		Description: "Public road in a city or town, typically with houses and buildings on each side",
		Example:     "364 East Rapidsborough",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return street(r), nil
		},
	})

	AddFuncLookup("streetname", Info{
		Display:     "Street Name",
		Category:    "address",
		Description: "Name given to a specific road or street",
		Example:     "View",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return streetName(r), nil
		},
	})

	AddFuncLookup("streetnumber", Info{
		Display:     "Street Number",
		Category:    "address",
		Description: "Numerical identifier assigned to a street",
		Example:     "13645",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return streetNumber(r), nil
		},
	})

	AddFuncLookup("streetprefix", Info{
		Display:     "Street Prefix",
		Category:    "address",
		Description: "Directional or descriptive term preceding a street name, like 'East' or 'Main'",
		Example:     "Lake",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return streetPrefix(r), nil
		},
	})

	AddFuncLookup("streetsuffix", Info{
		Display:     "Street Suffix",
		Category:    "address",
		Description: "Designation at the end of a street name indicating type, like 'Avenue' or 'Street'",
		Example:     "land",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return streetSuffix(r), nil
		},
	})

	AddFuncLookup("zip", Info{
		Display:     "Zip",
		Category:    "address",
		Description: "Numerical code for postal address sorting, specific to a geographic area",
		Example:     "13645",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return zip(r), nil
		},
	})

	AddFuncLookup("latitude", Info{
		Display:     "Latitude",
		Category:    "address",
		Description: "Geographic coordinate specifying north-south position on Earth's surface",
		Example:     "-73.534056",
		Output:      "float",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return latitude(r), nil
		},
	})

	AddFuncLookup("latituderange", Info{
		Display:     "Latitude Range",
		Category:    "address",
		Description: "Latitude number between the given range (default min=0, max=90)",
		Example:     "22.921026",
		Output:      "float",
		Params: []Param{
			{Field: "min", Display: "Min", Type: "float", Default: "0", Description: "Minimum range"},
			{Field: "max", Display: "Max", Type: "float", Default: "90", Description: "Maximum range"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			min, err := info.GetFloat64(m, "min")
			if err != nil {
				return nil, err
			}

			max, err := info.GetFloat64(m, "max")
			if err != nil {
				return nil, err
			}

			rangeOut, err := latitudeInRange(r, min, max)
			if err != nil {
				return nil, err
			}

			return rangeOut, nil
		},
	})

	AddFuncLookup("longitude", Info{
		Display:     "Longitude",
		Category:    "address",
		Description: "Geographic coordinate indicating east-west position on Earth's surface",
		Example:     "-147.068112",
		Output:      "float",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return longitude(r), nil
		},
	})

	AddFuncLookup("longituderange", Info{
		Display:     "Longitude Range",
		Category:    "address",
		Description: "Longitude number between the given range (default min=0, max=180)",
		Example:     "-8.170450",
		Output:      "float",
		Params: []Param{
			{Field: "min", Display: "Min", Type: "float", Default: "0", Description: "Minimum range"},
			{Field: "max", Display: "Max", Type: "float", Default: "180", Description: "Maximum range"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			min, err := info.GetFloat64(m, "min")
			if err != nil {
				return nil, err
			}

			max, err := info.GetFloat64(m, "max")
			if err != nil {
				return nil, err
			}

			rangeOut, err := longitudeInRange(r, min, max)
			if err != nil {
				return nil, err
			}

			return rangeOut, nil
		},
	})
}
