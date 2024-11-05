package gofakeit

import "math/rand"

// Emoji will return a random fun emoji
func Emoji() string { return emoji(globalFaker.Rand) }

// Emoji will return a random fun emoji
func (f *Faker) Emoji() string { return emoji(f.Rand) }

func emoji(r *rand.Rand) string { return getRandValue(r, []string{"emoji", "emoji"}) }

// EmojiDescription will return a random fun emoji description
func EmojiDescription() string { return emojiDescription(globalFaker.Rand) }

// EmojiDescription will return a random fun emoji description
func (f *Faker) EmojiDescription() string { return emojiDescription(f.Rand) }

func emojiDescription(r *rand.Rand) string { return getRandValue(r, []string{"emoji", "description"}) }

// EmojiCategory will return a random fun emoji category
func EmojiCategory() string { return emojiCategory(globalFaker.Rand) }

// EmojiCategory will return a random fun emoji category
func (f *Faker) EmojiCategory() string { return emojiCategory(f.Rand) }

func emojiCategory(r *rand.Rand) string { return getRandValue(r, []string{"emoji", "category"}) }

// EmojiAlias will return a random fun emoji alias
func EmojiAlias() string { return emojiAlias(globalFaker.Rand) }

// EmojiAlias will return a random fun emoji alias
func (f *Faker) EmojiAlias() string { return emojiAlias(f.Rand) }

func emojiAlias(r *rand.Rand) string { return getRandValue(r, []string{"emoji", "alias"}) }

// EmojiTag will return a random fun emoji tag
func EmojiTag() string { return emojiTag(globalFaker.Rand) }

// EmojiTag will return a random fun emoji tag
func (f *Faker) EmojiTag() string { return emojiTag(f.Rand) }

func emojiTag(r *rand.Rand) string { return getRandValue(r, []string{"emoji", "tag"}) }

func addEmojiLookup() {
	AddFuncLookup("emoji", Info{
		Display:     "Emoji",
		Category:    "emoji",
		Description: "Digital symbol expressing feelings or ideas in text messages and online chats",
		Example:     "🤣",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return emoji(r), nil
		},
	})

	AddFuncLookup("emojidescription", Info{
		Display:     "Emoji Description",
		Category:    "emoji",
		Description: "Brief explanation of the meaning or emotion conveyed by an emoji",
		Example:     "face vomiting",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return emojiDescription(r), nil
		},
	})

	AddFuncLookup("emojicategory", Info{
		Display:     "Emoji Category",
		Category:    "emoji",
		Description: "Group or classification of emojis based on their common theme or use, like 'smileys' or 'animals'",
		Example:     "Smileys & Emotion",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return emojiCategory(r), nil
		},
	})

	AddFuncLookup("emojialias", Info{
		Display:     "Emoji Alias",
		Category:    "emoji",
		Description: "Alternative name or keyword used to represent a specific emoji in text or code",
		Example:     "smile",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return emojiAlias(r), nil
		},
	})

	AddFuncLookup("emojitag", Info{
		Display:     "Emoji Tag",
		Category:    "emoji",
		Description: "Label or keyword associated with an emoji to categorize or search for it easily",
		Example:     "happy",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return emojiTag(r), nil
		},
	})
}
