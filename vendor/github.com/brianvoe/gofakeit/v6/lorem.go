package gofakeit

import (
	"errors"
	"math/rand"
)

// LoremIpsumWord will generate a random word
func LoremIpsumWord() string { return loremIpsumWord(globalFaker.Rand) }

// LoremIpsumWord will generate a random word
func (f *Faker) LoremIpsumWord() string { return loremIpsumWord(f.Rand) }

func loremIpsumWord(r *rand.Rand) string { return getRandValue(r, []string{"lorem", "word"}) }

// LoremIpsumSentence will generate a random sentence
func LoremIpsumSentence(wordCount int) string {
	return loremIpsumSentence(globalFaker.Rand, wordCount)
}

// LoremIpsumSentence will generate a random sentence
func (f *Faker) LoremIpsumSentence(wordCount int) string {
	return loremIpsumSentence(f.Rand, wordCount)
}

func loremIpsumSentence(r *rand.Rand, wordCount int) string {
	return sentenceGen(r, wordCount, loremIpsumWord)
}

// LoremIpsumParagraph will generate a random paragraphGenerator
func LoremIpsumParagraph(paragraphCount int, sentenceCount int, wordCount int, separator string) string {
	return loremIpsumParagraph(globalFaker.Rand, paragraphCount, sentenceCount, wordCount, separator)
}

// LoremIpsumParagraph will generate a random paragraphGenerator
func (f *Faker) LoremIpsumParagraph(paragraphCount int, sentenceCount int, wordCount int, separator string) string {
	return loremIpsumParagraph(f.Rand, paragraphCount, sentenceCount, wordCount, separator)
}

func loremIpsumParagraph(r *rand.Rand, paragraphCount int, sentenceCount int, wordCount int, separator string) string {
	return paragraphGen(r, paragrapOptions{paragraphCount, sentenceCount, wordCount, separator}, loremIpsumSentence)
}

func addLoremLookup() {
	AddFuncLookup("loremipsumword", Info{
		Display:     "Lorem Ipsum Word",
		Category:    "word",
		Description: "Word of the Lorem Ipsum placeholder text used in design and publishing",
		Example:     "quia",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return loremIpsumWord(r), nil
		},
	})

	AddFuncLookup("loremipsumsentence", Info{
		Display:     "Lorem Ipsum Sentence",
		Category:    "word",
		Description: "Sentence of the Lorem Ipsum placeholder text used in design and publishing",
		Example:     "Quia quae repellat consequatur quidem.",
		Output:      "string",
		Params: []Param{
			{Field: "wordcount", Display: "Word Count", Type: "int", Default: "5", Description: "Number of words in a sentence"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			wordCount, err := info.GetInt(m, "wordcount")
			if err != nil {
				return nil, err
			}
			if wordCount <= 0 || wordCount > 50 {
				return nil, errors.New("invalid word count, must be greater than 0, less than 50")
			}

			return loremIpsumSentence(r, wordCount), nil
		},
	})

	AddFuncLookup("loremipsumparagraph", Info{
		Display:     "Lorem Ipsum Paragraph",
		Category:    "word",
		Description: "Paragraph of the Lorem Ipsum placeholder text used in design and publishing",
		Example: `Quia quae repellat consequatur quidem nisi quo qui voluptatum accusantium quisquam amet. Quas et ut non dolorem ipsam aut enim assumenda mollitia harum ut. Dicta similique veniam nulla voluptas at excepturi non ad maxime at non. Eaque hic repellat praesentium voluptatem qui consequuntur dolor iusto autem velit aut. Fugit tempore exercitationem harum consequatur voluptatum modi minima aut eaque et et.

Aut ea voluptatem dignissimos expedita odit tempore quod aut beatae ipsam iste. Minus voluptatibus dolorem maiores eius sed nihil vel enim odio voluptatem accusamus. Natus quibusdam temporibus tenetur cumque sint necessitatibus dolorem ex ducimus iusto ex. Voluptatem neque dicta explicabo officiis et ducimus sit ut ut praesentium pariatur. Illum molestias nisi at dolore ut voluptatem accusantium et fugiat et ut.

Explicabo incidunt reprehenderit non quia dignissimos recusandae vitae soluta quia et quia. Aut veniam voluptas consequatur placeat sapiente non eveniet voluptatibus magni velit eum. Nobis vel repellendus sed est qui autem laudantium quidem quam ullam consequatur. Aut iusto ut commodi similique quae voluptatem atque qui fugiat eum aut. Quis distinctio consequatur voluptatem vel aliquid aut laborum facere officiis iure tempora.`,
		Output: "string",
		Params: []Param{
			{Field: "paragraphcount", Display: "Paragraph Count", Type: "int", Default: "2", Description: "Number of paragraphs"},
			{Field: "sentencecount", Display: "Sentence Count", Type: "int", Default: "2", Description: "Number of sentences in a paragraph"},
			{Field: "wordcount", Display: "Word Count", Type: "int", Default: "5", Description: "Number of words in a sentence"},
			{Field: "paragraphseparator", Display: "Paragraph Separator", Type: "string", Default: "<br />", Description: "String value to add between paragraphs"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			paragraphCount, err := info.GetInt(m, "paragraphcount")
			if err != nil {
				return nil, err
			}
			if paragraphCount <= 0 || paragraphCount > 20 {
				return nil, errors.New("invalid paragraph count, must be greater than 0, less than 20")
			}

			sentenceCount, err := info.GetInt(m, "sentencecount")
			if err != nil {
				return nil, err
			}
			if sentenceCount <= 0 || sentenceCount > 20 {
				return nil, errors.New("invalid sentence count, must be greater than 0, less than 20")
			}

			wordCount, err := info.GetInt(m, "wordcount")
			if err != nil {
				return nil, err
			}
			if wordCount <= 0 || wordCount > 50 {
				return nil, errors.New("invalid word count, must be greater than 0, less than 50")
			}

			paragraphSeparator, err := info.GetString(m, "paragraphseparator")
			if err != nil {
				return nil, err
			}

			return loremIpsumParagraph(r, paragraphCount, sentenceCount, wordCount, paragraphSeparator), nil
		},
	})
}
