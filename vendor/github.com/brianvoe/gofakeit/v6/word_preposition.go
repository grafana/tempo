package gofakeit

import "math/rand"

// Preposition will generate a random preposition
func Preposition() string { return preposition(globalFaker.Rand) }

// Preposition will generate a random preposition
func (f *Faker) Preposition() string { return preposition(f.Rand) }

func preposition(r *rand.Rand) string {
	var prepType = map[int]string{
		0: "preposition_simple",
		1: "preposition_double",
		2: "preposition_compound",
	}
	return getRandValue(r, []string{"word", prepType[number(r, 0, 2)]})
}

// PrepositionSimple will generate a random simple preposition
func PrepositionSimple() string { return prepositionSimple(globalFaker.Rand) }

// PrepositionSimple will generate a random simple preposition
func (f *Faker) PrepositionSimple() string { return prepositionSimple(f.Rand) }

func prepositionSimple(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "preposition_simple"})
}

// PrepositionDouble will generate a random double preposition
func PrepositionDouble() string { return prepositionDouble(globalFaker.Rand) }

// PrepositionDouble will generate a random double preposition
func (f *Faker) PrepositionDouble() string { return prepositionDouble(f.Rand) }

func prepositionDouble(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "preposition_double"})
}

// PrepositionCompound will generate a random compound preposition
func PrepositionCompound() string { return prepositionCompound(globalFaker.Rand) }

// PrepositionCompound will generate a random compound preposition
func (f *Faker) PrepositionCompound() string { return prepositionCompound(f.Rand) }

func prepositionCompound(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "preposition_compound"})
}

func addWordPrepositionLookup() {
	AddFuncLookup("preposition", Info{
		Display:     "Preposition",
		Category:    "word",
		Description: "Random preposition",
		Example:     "other than",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return preposition(r), nil
		},
	})

	AddFuncLookup("prepositionsimple", Info{
		Display:     "Preposition Simple",
		Category:    "word",
		Description: "Random simple preposition",
		Example:     "out",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return prepositionSimple(r), nil
		},
	})

	AddFuncLookup("prepositiondouble", Info{
		Display:     "Preposition Double",
		Category:    "word",
		Description: "Random double preposition",
		Example:     "before",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return prepositionDouble(r), nil
		},
	})

	AddFuncLookup("prepositioncompound", Info{
		Display:     "Preposition Compound",
		Category:    "word",
		Description: "Random compound preposition",
		Example:     "according to",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return prepositionCompound(r), nil
		},
	})
}
