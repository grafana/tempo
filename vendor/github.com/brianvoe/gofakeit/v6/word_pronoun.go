package gofakeit

import "math/rand"

// Pronoun will generate a random pronoun
func Pronoun() string { return pronoun(globalFaker.Rand) }

// Pronoun will generate a random pronoun
func (f *Faker) Pronoun() string { return pronoun(f.Rand) }

func pronoun(r *rand.Rand) string {
	var pronounType = map[int]string{
		0: "pronoun_personal",
		1: "pronoun_object",
		2: "pronoun_possessive",
		3: "pronoun_reflective",
		4: "pronoun_indefinite",
		5: "pronoun_demonstrative",
		6: "pronoun_interrogative",
		7: "pronoun_relative",
	}
	return getRandValue(r, []string{"word", pronounType[number(r, 0, 7)]})
}

// PronounPersonal will generate a random personal pronoun
func PronounPersonal() string { return pronounPersonal(globalFaker.Rand) }

// PronounPersonal will generate a random personal pronoun
func (f *Faker) PronounPersonal() string { return pronounPersonal(f.Rand) }

func pronounPersonal(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "pronoun_personal"})
}

// PronounObject will generate a random object pronoun
func PronounObject() string { return pronounObject(globalFaker.Rand) }

// PronounObject will generate a random object pronoun
func (f *Faker) PronounObject() string { return pronounObject(f.Rand) }

func pronounObject(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "pronoun_object"})
}

// PronounPossessive will generate a random possessive pronoun
func PronounPossessive() string { return pronounPossessive(globalFaker.Rand) }

// PronounPossessive will generate a random possessive pronoun
func (f *Faker) PronounPossessive() string { return pronounPossessive(f.Rand) }

func pronounPossessive(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "pronoun_possessive"})
}

// PronounReflective will generate a random reflective pronoun
func PronounReflective() string { return pronounReflective(globalFaker.Rand) }

// PronounReflective will generate a random reflective pronoun
func (f *Faker) PronounReflective() string { return pronounReflective(f.Rand) }

func pronounReflective(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "pronoun_reflective"})
}

// PronounIndefinite will generate a random indefinite pronoun
func PronounIndefinite() string { return pronounIndefinite(globalFaker.Rand) }

// PronounIndefinite will generate a random indefinite pronoun
func (f *Faker) PronounIndefinite() string { return pronounIndefinite(f.Rand) }

func pronounIndefinite(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "pronoun_indefinite"})
}

// PronounDemonstrative will generate a random demonstrative pronoun
func PronounDemonstrative() string { return pronounDemonstrative(globalFaker.Rand) }

// PronounDemonstrative will generate a random demonstrative pronoun
func (f *Faker) PronounDemonstrative() string { return pronounDemonstrative(f.Rand) }

func pronounDemonstrative(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "pronoun_demonstrative"})
}

// PronounInterrogative will generate a random interrogative pronoun
func PronounInterrogative() string { return pronounInterrogative(globalFaker.Rand) }

// PronounInterrogative will generate a random interrogative pronoun
func (f *Faker) PronounInterrogative() string { return pronounInterrogative(f.Rand) }

func pronounInterrogative(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "pronoun_interrogative"})
}

// PronounRelative will generate a random relative pronoun
func PronounRelative() string { return pronounRelative(globalFaker.Rand) }

// PronounRelative will generate a random relative pronoun
func (f *Faker) PronounRelative() string { return pronounRelative(f.Rand) }

func pronounRelative(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "pronoun_relative"})
}

func addWordPronounLookup() {
	AddFuncLookup("pronoun", Info{
		Display:     "Pronoun",
		Category:    "word",
		Description: "Word used in place of a noun to avoid repetition",
		Example:     "me",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return pronoun(r), nil
		},
	})

	AddFuncLookup("pronounpersonal", Info{
		Display:     "Pronoun Personal",
		Category:    "word",
		Description: "Pronoun referring to a specific persons or things",
		Example:     "it",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return pronounPersonal(r), nil
		},
	})

	AddFuncLookup("pronounobject", Info{
		Display:     "Pronoun Object",
		Category:    "word",
		Description: "Pronoun used as the object of a verb or preposition",
		Example:     "it",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return pronounObject(r), nil
		},
	})

	AddFuncLookup("pronounpossessive", Info{
		Display:     "Pronoun Possessive",
		Category:    "word",
		Description: "Pronoun indicating ownership or belonging",
		Example:     "mine",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return pronounPossessive(r), nil
		},
	})

	AddFuncLookup("pronounreflective", Info{
		Display:     "Pronoun Reflective",
		Category:    "word",
		Description: "Pronoun referring back to the subject of the sentence",
		Example:     "myself",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return pronounReflective(r), nil
		},
	})

	AddFuncLookup("pronounindefinite", Info{
		Display:     "Pronoun Indefinite",
		Category:    "word",
		Description: "Pronoun that does not refer to a specific person or thing",
		Example:     "few",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return pronounIndefinite(r), nil
		},
	})

	AddFuncLookup("pronoundemonstrative", Info{
		Display:     "Pronoun Demonstrative",
		Category:    "word",
		Description: "Pronoun that points out specific people or things",
		Example:     "this",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return pronounDemonstrative(r), nil
		},
	})

	AddFuncLookup("pronouninterrogative", Info{
		Display:     "Pronoun Interrogative",
		Category:    "word",
		Description: "Pronoun used to ask questions",
		Example:     "what",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return pronounInterrogative(r), nil
		},
	})

	AddFuncLookup("pronounrelative", Info{
		Display:     "Pronoun Relative",
		Category:    "word",
		Description: "Pronoun that introduces a clause, referring back to a noun or pronoun",
		Example:     "as",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return pronounRelative(r), nil
		},
	})
}
