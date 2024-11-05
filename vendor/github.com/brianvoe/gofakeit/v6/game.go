package gofakeit

import (
	"fmt"
	"math/rand"
	"strings"
)

// Gamertag will generate a random video game username
func Gamertag() string { return gamertag(globalFaker.Rand) }

// Gamertag will generate a random video game username
func (f *Faker) Gamertag() string { return gamertag(f.Rand) }

func gamertag(r *rand.Rand) string {
	str := ""
	num := number(r, 1, 4)
	switch num {
	case 1:
		str = fmt.Sprintf("%s%ser", title(nounConcrete(r)), title(verbAction(r)))
	case 2:
		str = fmt.Sprintf("%s%s", title(adjectiveDescriptive(r)), title(animal(r)))
	case 3:
		str = fmt.Sprintf("%s%s", title(adjectiveDescriptive(r)), title(nounConcrete(r)))
	case 4:
		str = fmt.Sprintf("%s%s", title(fruit(r)), title(adjectiveDescriptive(r)))
	}

	// Randomly determine if we should add a number
	if r.Intn(3) == 1 {
		str += digitN(r, uint(number(r, 1, 3)))
	}

	// Remove any spaces
	str = strings.Replace(str, " ", "", -1)

	return str
}

// Dice will generate a random set of dice
func Dice(numDice uint, sides []uint) []uint { return dice(globalFaker.Rand, numDice, sides) }

// Dice will generate a random set of dice
func (f *Faker) Dice(numDice uint, sides []uint) []uint { return dice(f.Rand, numDice, sides) }

func dice(r *rand.Rand, numDice uint, sides []uint) []uint {
	dice := make([]uint, numDice)

	// If we dont have any sides well set the sides to 6
	if len(sides) == 0 {
		sides = []uint{6}
	}

	for i := range dice {
		// If sides[i] doesnt exist use the first side
		if len(sides)-1 < i {
			dice[i] = uint(number(r, 1, int(sides[0])))
		} else {
			dice[i] = uint(number(r, 1, int(sides[i])))
		}
	}

	return dice
}

func addGameLookup() {
	AddFuncLookup("gamertag", Info{
		Display:     "Gamertag",
		Category:    "game",
		Description: "User-selected online username or alias used for identification in games",
		Example:     "footinterpret63",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return gamertag(r), nil
		},
	})

	AddFuncLookup("dice", Info{
		Display:     "Dice",
		Category:    "game",
		Description: "Small, cube-shaped objects used in games of chance for random outcomes",
		Example:     "[5, 2, 3]",
		Output:      "[]uint",
		Params: []Param{
			{Field: "numdice", Display: "Number of Dice", Type: "uint", Default: "1", Description: "Number of dice to roll"},
			{Field: "sides", Display: "Number of Sides", Type: "[]uint", Default: "[6]", Description: "Number of sides on each dice"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			numDice, err := info.GetUint(m, "numdice")
			if err != nil {
				return nil, err
			}

			sides, err := info.GetUintArray(m, "sides")
			if err != nil {
				return nil, err
			}

			return dice(r, numDice, sides), nil
		},
	})
}
