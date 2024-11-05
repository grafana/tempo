package gofakeit

import (
	"math/rand"
	"unicode"
)

// SentenceSimple will generate a random simple sentence
func SentenceSimple() string { return sentenceSimple(globalFaker.Rand) }

// SentenceSimple will generate a random simple sentence
func (f *Faker) SentenceSimple() string { return sentenceSimple(f.Rand) }

func sentenceSimple(r *rand.Rand) string {
	// simple sentence consists of a noun phrase and a verb phrase
	str := phraseNoun(r) + " " + phraseVerb(r) + "."

	// capitalize the first letter
	strR := []rune(str)
	strR[0] = unicode.ToUpper(strR[0])
	return string(strR)
}

func addWordGrammerLookup() {
	AddFuncLookup("sentencesimple", Info{
		Display:     "Simple Sentence",
		Category:    "word",
		Description: "Group of words that expresses a complete thought",
		Example:     "A tribe fly the lemony kitchen.",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return sentenceSimple(r), nil
		},
	})
}
