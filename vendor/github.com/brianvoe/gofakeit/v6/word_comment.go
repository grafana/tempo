package gofakeit

import (
	"math/rand"
	"strings"
)

// Comment will generate a random statement or remark expressing an opinion, observation, or reaction
func Comment() string { return comment(globalFaker.Rand) }

// Comment will generate a random statement or remark expressing an opinion, observation, or reaction
func (f *Faker) Comment() string { return comment(f.Rand) }

func comment(r *rand.Rand) string {
	structures := [][]string{
		{"interjection", "adjective", "noun", "verb", "adverb"},
		{"noun", "verb", "preposition", "determiner", "adjective", "noun"},
		{"noun", "verb", "adverb"},
		{"adjective", "noun", "verb"},
		{"noun", "verb", "preposition", "noun"},
	}

	// Randomly select a structure
	structure := structures[number(r, 0, len(structures)-1)]

	// Build the sentence
	var commentParts []string
	for _, wordType := range structure {
		switch wordType {
		case "noun":
			commentParts = append(commentParts, noun(r))
		case "verb":
			commentParts = append(commentParts, verb(r))
		case "adjective":
			commentParts = append(commentParts, adjective(r))
		case "adverb":
			commentParts = append(commentParts, adverb(r))
		case "interjection":
			commentParts = append(commentParts, interjection(r))
		case "preposition":
			commentParts = append(commentParts, preposition(r))
		case "determiner":
			commentParts = append(commentParts, nounDeterminer(r))
		default:
			// Should never hit
			panic("Invalid word type")
		}
	}

	// Combine the words into a sentence
	sentence := strings.Join(commentParts, " ")

	// Capitalize the first letter
	sentence = title(sentence)

	// Add a period to the end of the sentence
	sentence = sentence + "."

	return sentence
}

func addWordCommentLookup() {
	AddFuncLookup("comment", Info{
		Display:     "Comment",
		Category:    "word",
		Description: "Statement or remark expressing an opinion, observation, or reaction",
		Example:     "wow",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return interjection(r), nil
		},
	})
}
