package gofakeit

import "math/rand"

// Interjection will generate a random word expressing emotion
func Interjection() string { return interjection(globalFaker.Rand) }

// Interjection will generate a random word expressing emotion
func (f *Faker) Interjection() string { return interjection(f.Rand) }

func interjection(r *rand.Rand) string { return getRandValue(r, []string{"word", "interjection"}) }

func addWordMiscLookup() {
	AddFuncLookup("interjection", Info{
		Display:     "Interjection",
		Category:    "word",
		Description: "Word expressing emotion",
		Example:     "wow",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return interjection(r), nil
		},
	})
}
