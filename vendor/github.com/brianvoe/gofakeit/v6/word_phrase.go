package gofakeit

import (
	"math/rand"
	"strings"
)

// Phrase will return a random phrase
func Phrase() string { return phrase(globalFaker.Rand) }

// Phrase will return a random phrase
func (f *Faker) Phrase() string { return phrase(f.Rand) }

func phrase(r *rand.Rand) string { return getRandValue(r, []string{"sentence", "phrase"}) }

// PhraseNoun will return a random noun phrase
func PhraseNoun() string { return phraseNoun(globalFaker.Rand) }

// PhraseNoun will return a random noun phrase
func (f *Faker) PhraseNoun() string { return phraseNoun(f.Rand) }

func phraseNoun(r *rand.Rand) string {
	str := ""

	// You may also want to add an adjective to describe the noun
	if boolFunc(r) {
		str = adjectiveDescriptive(r) + " " + noun(r)
	} else {
		str = noun(r)
	}

	// Add determiner from weighted list
	prob, _ := weighted(r, []any{1, 2, 3}, []float32{2, 1.5, 1})
	if prob == 1 {
		str = getArticle(str) + " " + str
	} else if prob == 2 {
		str = "the " + str
	}

	return str
}

// PhraseVerb will return a random preposition phrase
func PhraseVerb() string { return phraseVerb(globalFaker.Rand) }

// PhraseVerb will return a random preposition phrase
func (f *Faker) PhraseVerb() string { return phraseVerb(f.Rand) }

func phraseVerb(r *rand.Rand) string {
	// Put together a string builder
	sb := []string{}

	// You may have an adverb phrase
	if boolFunc(r) {
		sb = append(sb, phraseAdverb(r))
	}

	// Lets add the primary verb
	sb = append(sb, verbAction(r))

	// You may have a noun phrase
	if boolFunc(r) {
		sb = append(sb, phraseNoun(r))
	}

	// You may have an adverb phrase
	if boolFunc(r) {
		sb = append(sb, phraseAdverb(r))

		// You may also have a preposition phrase
		if boolFunc(r) {
			sb = append(sb, phrasePreposition(r))
		}

		// You may also hae an adverb phrase
		if boolFunc(r) {
			sb = append(sb, phraseAdverb(r))
		}
	}

	return strings.Join(sb, " ")
}

// PhraseAdverb will return a random adverb phrase
func PhraseAdverb() string { return phraseAdverb(globalFaker.Rand) }

// PhraseAdverb will return a random adverb phrase
func (f *Faker) PhraseAdverb() string { return phraseAdverb(f.Rand) }

func phraseAdverb(r *rand.Rand) string {
	if boolFunc(r) {
		return adverbDegree(r) + " " + adverbManner(r)
	}

	return adverbManner(r)
}

// PhrasePreposition will return a random preposition phrase
func PhrasePreposition() string { return phrasePreposition(globalFaker.Rand) }

// PhrasePreposition will return a random preposition phrase
func (f *Faker) PhrasePreposition() string { return phrasePreposition(f.Rand) }

func phrasePreposition(r *rand.Rand) string {
	return prepositionSimple(r) + " " + phraseNoun(r)
}

func addWordPhraseLookup() {
	AddFuncLookup("phrase", Info{
		Display:     "Phrase",
		Category:    "word",
		Description: "A small group of words standing together",
		Example:     "time will tell",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return phrase(r), nil
		},
	})

	AddFuncLookup("phrasenoun", Info{
		Display:     "Noun Phrase",
		Category:    "word",
		Description: "Phrase with a noun as its head, functions within sentence like a noun",
		Example:     "a tribe",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return phraseNoun(r), nil
		},
	})

	AddFuncLookup("phraseverb", Info{
		Display:     "Verb Phrase",
		Category:    "word",
		Description: "Phrase that Consists of a verb and its modifiers, expressing an action or state",
		Example:     "a tribe",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return phraseVerb(r), nil
		},
	})

	AddFuncLookup("phraseadverb", Info{
		Display:     "Adverb Phrase",
		Category:    "word",
		Description: "Phrase that modifies a verb, adjective, or another adverb, providing additional information.",
		Example:     "fully gladly",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return phraseAdverb(r), nil
		},
	})

	AddFuncLookup("phrasepreposition", Info{
		Display:     "Preposition Phrase",
		Category:    "word",
		Description: "Phrase starting with a preposition, showing relation between elements in a sentence.",
		Example:     "out the black thing",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return phrasePreposition(r), nil
		},
	})
}
