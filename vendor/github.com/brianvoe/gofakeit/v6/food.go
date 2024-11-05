package gofakeit

import (
	"math/rand"
	"strings"
)

// Fruit will return a random fruit name
func Fruit() string { return fruit(globalFaker.Rand) }

// Fruit will return a random fruit name
func (f *Faker) Fruit() string { return fruit(f.Rand) }

func fruit(r *rand.Rand) string { return getRandValue(r, []string{"food", "fruit"}) }

// Vegetable will return a random vegetable name
func Vegetable() string { return vegetable(globalFaker.Rand) }

// Vegetable will return a random vegetable name
func (f *Faker) Vegetable() string { return vegetable(f.Rand) }

func vegetable(r *rand.Rand) string { return getRandValue(r, []string{"food", "vegetable"}) }

// Breakfast will return a random breakfast name
func Breakfast() string { return breakfast(globalFaker.Rand) }

// Breakfast will return a random breakfast name
func (f *Faker) Breakfast() string { return breakfast(f.Rand) }

func breakfast(r *rand.Rand) string {
	v := getRandValue(r, []string{"food", "breakfast"})
	return strings.ToUpper(v[:1]) + v[1:]
}

// Lunch will return a random lunch name
func Lunch() string { return lunch(globalFaker.Rand) }

// Lunch will return a random lunch name
func (f *Faker) Lunch() string { return lunch(f.Rand) }

func lunch(r *rand.Rand) string {
	v := getRandValue(r, []string{"food", "lunch"})
	return strings.ToUpper(v[:1]) + v[1:]
}

// Dinner will return a random dinner name
func Dinner() string { return dinner(globalFaker.Rand) }

// Dinner will return a random dinner name
func (f *Faker) Dinner() string { return dinner(f.Rand) }

func dinner(r *rand.Rand) string {
	v := getRandValue(r, []string{"food", "dinner"})
	return strings.ToUpper(v[:1]) + v[1:]
}

// Drink will return a random drink name
func Drink() string { return drink(globalFaker.Rand) }

// Drink will return a random drink name
func (f *Faker) Drink() string { return drink(f.Rand) }

func drink(r *rand.Rand) string {
	v := getRandValue(r, []string{"food", "drink"})
	return strings.ToUpper(v[:1]) + v[1:]
}

// Snack will return a random snack name
func Snack() string { return snack(globalFaker.Rand) }

// Snack will return a random snack name
func (f *Faker) Snack() string { return snack(f.Rand) }

func snack(r *rand.Rand) string {
	v := getRandValue(r, []string{"food", "snack"})
	return strings.ToUpper(v[:1]) + v[1:]
}

// Dessert will return a random dessert name
func Dessert() string { return dessert(globalFaker.Rand) }

// Dessert will return a random dessert name
func (f *Faker) Dessert() string { return dessert(f.Rand) }

func dessert(r *rand.Rand) string {
	v := getRandValue(r, []string{"food", "dessert"})
	return strings.ToUpper(v[:1]) + v[1:]
}

func addFoodLookup() {
	AddFuncLookup("fruit", Info{
		Display:     "Fruit",
		Category:    "food",
		Description: "Edible plant part, typically sweet, enjoyed as a natural snack or dessert",
		Example:     "Peach",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return fruit(r), nil
		},
	})

	AddFuncLookup("vegetable", Info{
		Display:     "Vegetable",
		Category:    "food",
		Description: "Edible plant or part of a plant, often used in savory cooking or salads",
		Example:     "Amaranth Leaves",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return vegetable(r), nil
		},
	})

	AddFuncLookup("breakfast", Info{
		Display:     "Breakfast",
		Category:    "food",
		Description: "First meal of the day, typically eaten in the morning",
		Example:     "Blueberry banana happy face pancakes",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return breakfast(r), nil
		},
	})

	AddFuncLookup("lunch", Info{
		Display:     "Lunch",
		Category:    "food",
		Description: "Midday meal, often lighter than dinner, eaten around noon",
		Example:     "No bake hersheys bar pie",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return lunch(r), nil
		},
	})

	AddFuncLookup("dinner", Info{
		Display:     "Dinner",
		Category:    "food",
		Description: "Evening meal, typically the day's main and most substantial meal",
		Example:     "Wild addicting dip",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return dinner(r), nil
		},
	})

	AddFuncLookup("drink", Info{
		Display:     "Drink",
		Category:    "food",
		Description: "Liquid consumed for hydration, pleasure, or nutritional benefits",
		Example:     "Soda",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return drink(r), nil
		},
	})

	AddFuncLookup("snack", Info{
		Display:     "Snack",
		Category:    "food",
		Description: "Random snack",
		Example:     "Small, quick food item eaten between meals",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return snack(r), nil
		},
	})

	AddFuncLookup("dessert", Info{
		Display:     "Dessert",
		Category:    "food",
		Description: "Sweet treat often enjoyed after a meal",
		Example:     "French napoleons",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return dessert(r), nil
		},
	})
}
