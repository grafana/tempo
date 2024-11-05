package gofakeit

import "math/rand"

// CelebrityActor will generate a random celebrity actor
func CelebrityActor() string { return celebrityActor(globalFaker.Rand) }

// CelebrityActor will generate a random celebrity actor
func (f *Faker) CelebrityActor() string { return celebrityActor(f.Rand) }

func celebrityActor(r *rand.Rand) string { return getRandValue(r, []string{"celebrity", "actor"}) }

// CelebrityBusiness will generate a random celebrity business person
func CelebrityBusiness() string { return celebrityBusiness(globalFaker.Rand) }

// CelebrityBusiness will generate a random celebrity business person
func (f *Faker) CelebrityBusiness() string { return celebrityBusiness(f.Rand) }

func celebrityBusiness(r *rand.Rand) string {
	return getRandValue(r, []string{"celebrity", "business"})
}

// CelebritySport will generate a random celebrity sport person
func CelebritySport() string { return celebritySport(globalFaker.Rand) }

// CelebritySport will generate a random celebrity sport person
func (f *Faker) CelebritySport() string { return celebritySport(f.Rand) }

func celebritySport(r *rand.Rand) string { return getRandValue(r, []string{"celebrity", "sport"}) }

func addCelebrityLookup() {
	AddFuncLookup("celebrityactor", Info{
		Display:     "Celebrity Actor",
		Category:    "celebrity",
		Description: "Famous person known for acting in films, television, or theater",
		Example:     "Brad Pitt",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return celebrityActor(r), nil
		},
	})

	AddFuncLookup("celebritybusiness", Info{
		Display:     "Celebrity Business",
		Category:    "celebrity",
		Description: "High-profile individual known for significant achievements in business or entrepreneurship",
		Example:     "Elon Musk",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return celebrityBusiness(r), nil
		},
	})

	AddFuncLookup("celebritysport", Info{
		Display:     "Celebrity Sport",
		Category:    "celebrity",
		Description: "Famous athlete known for achievements in a particular sport",
		Example:     "Michael Phelps",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return celebritySport(r), nil
		},
	})
}
