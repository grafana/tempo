package gofakeit

import "math/rand"

// Language will return a random language
func Language() string { return language(globalFaker.Rand) }

// Language will return a random language
func (f *Faker) Language() string { return language(f.Rand) }

func language(r *rand.Rand) string { return getRandValue(r, []string{"language", "long"}) }

// LanguageAbbreviation will return a random language abbreviation
func LanguageAbbreviation() string { return languageAbbreviation(globalFaker.Rand) }

// LanguageAbbreviation will return a random language abbreviation
func (f *Faker) LanguageAbbreviation() string { return languageAbbreviation(f.Rand) }

func languageAbbreviation(r *rand.Rand) string { return getRandValue(r, []string{"language", "short"}) }

// LanguageBCP will return a random language BCP (Best Current Practices)
func LanguageBCP() string { return languageBCP(globalFaker.Rand) }

// LanguageBCP will return a random language BCP (Best Current Practices)
func (f *Faker) LanguageBCP() string { return languageBCP(f.Rand) }

func languageBCP(r *rand.Rand) string { return getRandValue(r, []string{"language", "bcp"}) }

// ProgrammingLanguage will return a random programming language
func ProgrammingLanguage() string { return programmingLanguage(globalFaker.Rand) }

// ProgrammingLanguage will return a random programming language
func (f *Faker) ProgrammingLanguage() string { return programmingLanguage(f.Rand) }

func programmingLanguage(r *rand.Rand) string {
	return getRandValue(r, []string{"language", "programming"})
}

// ProgrammingLanguageBest will return a random programming language
func ProgrammingLanguageBest() string { return programmingLanguageBest(globalFaker.Rand) }

// ProgrammingLanguageBest will return a random programming language
func (f *Faker) ProgrammingLanguageBest() string { return programmingLanguageBest(f.Rand) }

func programmingLanguageBest(r *rand.Rand) string { return "Go" }

func addLanguagesLookup() {
	AddFuncLookup("language", Info{
		Display:     "Language",
		Category:    "language",
		Description: "System of communication using symbols, words, and grammar to convey meaning between individuals",
		Example:     "Kazakh",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return language(r), nil
		},
	})

	AddFuncLookup("languageabbreviation", Info{
		Display:     "Language Abbreviation",
		Category:    "language",
		Description: "Shortened form of a language's name",
		Example:     "kk",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return languageAbbreviation(r), nil
		},
	})

	AddFuncLookup("languagebcp", Info{
		Display:     "Language BCP",
		Category:    "language",
		Description: "Set of guidelines and standards for identifying and representing languages in computing and internet protocols",
		Example:     "en-US",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return languageBCP(r), nil
		},
	})

	AddFuncLookup("programminglanguage", Info{
		Display:     "Programming Language",
		Category:    "language",
		Description: "Formal system of instructions used to create software and perform computational tasks",
		Example:     "Go",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return programmingLanguage(r), nil
		},
	})
}
