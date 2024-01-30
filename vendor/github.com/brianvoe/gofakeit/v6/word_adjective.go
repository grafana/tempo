package gofakeit

import "math/rand"

// Adjective will generate a random adjective
func Adjective() string { return adjective(globalFaker.Rand) }

// Adjective will generate a random adjective
func (f *Faker) Adjective() string { return adjective(f.Rand) }

func adjective(r *rand.Rand) string {
	var adjType = map[int]string{
		0: "adjective_descriptive",
		1: "adjective_quantitative",
		2: "adjective_proper",
		3: "adjective_demonstrative",
		4: "adjective_possessive",
		5: "adjective_interrogative",
		6: "adjective_indefinite",
	}
	return getRandValue(r, []string{"word", adjType[number(r, 0, 6)]})
}

// AdjectiveDescriptive will generate a random descriptive adjective
func AdjectiveDescriptive() string { return adjectiveDescriptive(globalFaker.Rand) }

// AdjectiveDescriptive will generate a random descriptive adjective
func (f *Faker) AdjectiveDescriptive() string { return adjectiveDescriptive(f.Rand) }

func adjectiveDescriptive(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "adjective_descriptive"})
}

// AdjectiveQuantitative will generate a random quantitative adjective
func AdjectiveQuantitative() string { return adjectiveQuantitative(globalFaker.Rand) }

// AdjectiveQuantitative will generate a random quantitative adjective
func (f *Faker) AdjectiveQuantitative() string { return adjectiveQuantitative(f.Rand) }

func adjectiveQuantitative(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "adjective_quantitative"})
}

// AdjectiveProper will generate a random proper adjective
func AdjectiveProper() string { return adjectiveProper(globalFaker.Rand) }

// AdjectiveProper will generate a random proper adjective
func (f *Faker) AdjectiveProper() string { return adjectiveProper(f.Rand) }

func adjectiveProper(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "adjective_proper"})
}

// AdjectiveDemonstrative will generate a random demonstrative adjective
func AdjectiveDemonstrative() string { return adjectiveDemonstrative(globalFaker.Rand) }

// AdjectiveDemonstrative will generate a random demonstrative adjective
func (f *Faker) AdjectiveDemonstrative() string { return adjectiveDemonstrative(f.Rand) }

func adjectiveDemonstrative(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "adjective_demonstrative"})
}

// AdjectivePossessive will generate a random possessive adjective
func AdjectivePossessive() string { return adjectivePossessive(globalFaker.Rand) }

// AdjectivePossessive will generate a random possessive adjective
func (f *Faker) AdjectivePossessive() string { return adjectivePossessive(f.Rand) }

func adjectivePossessive(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "adjective_possessive"})
}

// AdjectiveInterrogative will generate a random interrogative adjective
func AdjectiveInterrogative() string { return adjectiveInterrogative(globalFaker.Rand) }

// AdjectiveInterrogative will generate a random interrogative adjective
func (f *Faker) AdjectiveInterrogative() string { return adjectiveInterrogative(f.Rand) }

func adjectiveInterrogative(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "adjective_interrogative"})
}

// AdjectiveIndefinite will generate a random indefinite adjective
func AdjectiveIndefinite() string { return adjectiveIndefinite(globalFaker.Rand) }

// AdjectiveIndefinite will generate a random indefinite adjective
func (f *Faker) AdjectiveIndefinite() string { return adjectiveIndefinite(f.Rand) }

func adjectiveIndefinite(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "adjective_indefinite"})
}

func addWordAdjectiveLookup() {
	AddFuncLookup("adjective", Info{
		Display:     "Adjective",
		Category:    "word",
		Description: "Random adjective",
		Example:     "genuine",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return adjective(r), nil
		},
	})

	AddFuncLookup("adjectivedescriptive", Info{
		Display:     "Descriptive Adjective",
		Category:    "word",
		Description: "Random descriptive adjective",
		Example:     "brave",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return adjectiveDescriptive(r), nil
		},
	})

	AddFuncLookup("adjectivequantitative", Info{
		Display:     "Quantitative Adjective",
		Category:    "word",
		Description: "Random quantitative adjective",
		Example:     "a little",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return adjectiveQuantitative(r), nil
		},
	})

	AddFuncLookup("adjectiveproper", Info{
		Display:     "Proper Adjective",
		Category:    "word",
		Description: "Random proper adjective",
		Example:     "Afghan",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return adjectiveProper(r), nil
		},
	})

	AddFuncLookup("adjectivedemonstrative", Info{
		Display:     "Demonstrative Adjective",
		Category:    "word",
		Description: "Random demonstrative adjective",
		Example:     "this",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return adjectiveDemonstrative(r), nil
		},
	})

	AddFuncLookup("adjectivepossessive", Info{
		Display:     "Possessive Adjective",
		Category:    "word",
		Description: "Random possessive adjective",
		Example:     "my",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return adjectivePossessive(r), nil
		},
	})

	AddFuncLookup("adjectiveinterrogative", Info{
		Display:     "Interrogative Adjective",
		Category:    "word",
		Description: "Random interrogative adjective",
		Example:     "what",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return adjectiveInterrogative(r), nil
		},
	})

	AddFuncLookup("adjectiveindefinite", Info{
		Display:     "Indefinite Adjective",
		Category:    "word",
		Description: "Random indefinite adjective",
		Example:     "few",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return adjectiveIndefinite(r), nil
		},
	})
}
