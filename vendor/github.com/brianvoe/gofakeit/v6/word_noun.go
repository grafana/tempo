package gofakeit

import "math/rand"

// Noun will generate a random noun
func Noun() string { return noun(globalFaker.Rand) }

// Noun will generate a random noun
func (f *Faker) Noun() string { return noun(f.Rand) }

func noun(r *rand.Rand) string {
	var nounType = map[int]string{
		0: "noun_common",
		1: "noun_concrete",
		2: "noun_abstract",
		3: "noun_collective_people",
		4: "noun_collective_animal",
		5: "noun_collective_thing",
		6: "noun_countable",
		7: "noun_uncountable",
	}
	return getRandValue(r, []string{"word", nounType[number(r, 0, 7)]})
}

// NounCommon will generate a random common noun
func NounCommon() string { return nounCommon(globalFaker.Rand) }

// NounCommon will generate a random common noun
func (f *Faker) NounCommon() string { return nounCommon(f.Rand) }

func nounCommon(r *rand.Rand) string { return getRandValue(r, []string{"word", "noun_common"}) }

// NounConcrete will generate a random concrete noun
func NounConcrete() string { return nounConcrete(globalFaker.Rand) }

// NounConcrete will generate a random concrete noun
func (f *Faker) NounConcrete() string { return nounConcrete(f.Rand) }

func nounConcrete(r *rand.Rand) string { return getRandValue(r, []string{"word", "noun_concrete"}) }

// NounAbstract will generate a random abstract noun
func NounAbstract() string { return nounAbstract(globalFaker.Rand) }

// NounAbstract will generate a random abstract noun
func (f *Faker) NounAbstract() string { return nounAbstract(f.Rand) }

func nounAbstract(r *rand.Rand) string { return getRandValue(r, []string{"word", "noun_abstract"}) }

// NounCollectivePeople will generate a random collective noun person
func NounCollectivePeople() string { return nounCollectivePeople(globalFaker.Rand) }

// NounCollectivePeople will generate a random collective noun person
func (f *Faker) NounCollectivePeople() string { return nounCollectivePeople(f.Rand) }

func nounCollectivePeople(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "noun_collective_people"})
}

// NounCollectiveAnimal will generate a random collective noun animal
func NounCollectiveAnimal() string { return nounCollectiveAnimal(globalFaker.Rand) }

// NounCollectiveAnimal will generate a random collective noun animal
func (f *Faker) NounCollectiveAnimal() string { return nounCollectiveAnimal(f.Rand) }

func nounCollectiveAnimal(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "noun_collective_animal"})
}

// NounCollectiveThing will generate a random collective noun thing
func NounCollectiveThing() string { return nounCollectiveThing(globalFaker.Rand) }

// NounCollectiveThing will generate a random collective noun thing
func (f *Faker) NounCollectiveThing() string { return nounCollectiveThing(f.Rand) }

func nounCollectiveThing(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "noun_collective_thing"})
}

// NounCountable will generate a random countable noun
func NounCountable() string { return nounCountable(globalFaker.Rand) }

// NounCountable will generate a random countable noun
func (f *Faker) NounCountable() string { return nounCountable(f.Rand) }

func nounCountable(r *rand.Rand) string { return getRandValue(r, []string{"word", "noun_countable"}) }

// NounUncountable will generate a random uncountable noun
func NounUncountable() string { return nounUncountable(globalFaker.Rand) }

// NounUncountable will generate a random uncountable noun
func (f *Faker) NounUncountable() string { return nounUncountable(f.Rand) }

func nounUncountable(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "noun_uncountable"})
}

// NounProper will generate a random proper noun
func NounProper() string { return nounProper(globalFaker.Rand) }

// NounProper will generate a random proper noun
func (f *Faker) NounProper() string { return nounProper(f.Rand) }

func nounProper(r *rand.Rand) string {
	switch randInt := randIntRange(r, 1, 3); randInt {
	case 1:
		return getRandValue(r, []string{"celebrity", "actor"})
	case 2:
		return generate(r, getRandValue(r, []string{"address", "city"}))
	}

	return getRandValue(r, []string{"person", "first"})
}

// NounDeterminer will generate a random noun determiner
func NounDeterminer() string { return nounDeterminer(globalFaker.Rand) }

// NounDeterminer will generate a random noun determiner
func (f *Faker) NounDeterminer() string { return nounDeterminer(f.Rand) }

func nounDeterminer(r *rand.Rand) string { return getRandValue(r, []string{"word", "noun_determiner"}) }

func addWordNounLookup() {
	AddFuncLookup("noun", Info{
		Display:     "Noun",
		Category:    "word",
		Description: "Person, place, thing, or idea, named or referred to in a sentence",
		Example:     "aunt",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return noun(r), nil
		},
	})

	AddFuncLookup("nouncommon", Info{
		Display:     "Noun Common",
		Category:    "word",
		Description: "General name for people, places, or things, not specific or unique",
		Example:     "part",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return nounCommon(r), nil
		},
	})

	AddFuncLookup("nounconcrete", Info{
		Display:     "Noun Concrete",
		Category:    "word",
		Description: "Names for physical entities experienced through senses like sight, touch, smell, or taste",
		Example:     "snowman",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return nounConcrete(r), nil
		},
	})

	AddFuncLookup("nounabstract", Info{
		Display:     "Noun Abstract",
		Category:    "word",
		Description: "Ideas, qualities, or states that cannot be perceived with the five senses",
		Example:     "confusion",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return nounAbstract(r), nil
		},
	})

	AddFuncLookup("nouncollectivepeople", Info{
		Display:     "Noun Collective People",
		Category:    "word",
		Description: "Group of people or things regarded as a unit",
		Example:     "body",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return nounCollectivePeople(r), nil
		},
	})

	AddFuncLookup("nouncollectiveanimal", Info{
		Display:     "Noun Collective Animal",
		Category:    "word",
		Description: "Group of animals, like a 'pack' of wolves or a 'flock' of birds",
		Example:     "party",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return nounCollectiveAnimal(r), nil
		},
	})

	AddFuncLookup("nouncollectivething", Info{
		Display:     "Noun Collective Thing",
		Category:    "word",
		Description: "Group of objects or items, such as a 'bundle' of sticks or a 'cluster' of grapes",
		Example:     "hand",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return nounCollectiveThing(r), nil
		},
	})

	AddFuncLookup("nouncountable", Info{
		Display:     "Noun Countable",
		Category:    "word",
		Description: "Items that can be counted individually",
		Example:     "neck",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return nounCountable(r), nil
		},
	})

	AddFuncLookup("noununcountable", Info{
		Display:     "Noun Uncountable",
		Category:    "word",
		Description: "Items that can't be counted individually",
		Example:     "seafood",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return nounUncountable(r), nil
		},
	})

	AddFuncLookup("nounproper", Info{
		Display:     "Noun Proper",
		Category:    "word",
		Description: "Specific name for a particular person, place, or organization",
		Example:     "John",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return nounProper(r), nil
		},
	})

	AddFuncLookup("noundeterminer", Info{
		Display:     "Noun Determiner",
		Category:    "word",
		Description: "Word that introduces a noun and identifies it as a noun",
		Example:     "your",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return nounDeterminer(r), nil
		},
	})
}
