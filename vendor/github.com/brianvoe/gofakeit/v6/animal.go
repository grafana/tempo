package gofakeit

import "math/rand"

// PetName will return a random fun pet name
func PetName() string {
	return petName(globalFaker.Rand)
}

// PetName will return a random fun pet name
func (f *Faker) PetName() string {
	return petName(f.Rand)
}

func petName(r *rand.Rand) string {
	return getRandValue(r, []string{"animal", "petname"})
}

// Animal will return a random animal
func Animal() string {
	return animal(globalFaker.Rand)
}

// Animal will return a random animal
func (f *Faker) Animal() string {
	return animal(f.Rand)
}

func animal(r *rand.Rand) string {
	return getRandValue(r, []string{"animal", "animal"})
}

// AnimalType will return a random animal type
func AnimalType() string {
	return animalType(globalFaker.Rand)
}

// AnimalType will return a random animal type
func (f *Faker) AnimalType() string {
	return animalType(f.Rand)
}

func animalType(r *rand.Rand) string {
	return getRandValue(r, []string{"animal", "type"})
}

// FarmAnimal will return a random animal that usually lives on a farm
func FarmAnimal() string {
	return farmAnimal(globalFaker.Rand)
}

// FarmAnimal will return a random animal that usually lives on a farm
func (f *Faker) FarmAnimal() string {
	return farmAnimal(f.Rand)
}

func farmAnimal(r *rand.Rand) string {
	return getRandValue(r, []string{"animal", "farm"})
}

// Cat will return a random cat breed
func Cat() string {
	return cat(globalFaker.Rand)
}

// Cat will return a random cat breed
func (f *Faker) Cat() string {
	return cat(f.Rand)
}

func cat(r *rand.Rand) string {
	return getRandValue(r, []string{"animal", "cat"})
}

// Dog will return a random dog breed
func Dog() string {
	return dog(globalFaker.Rand)
}

// Dog will return a random dog breed
func (f *Faker) Dog() string {
	return dog(f.Rand)
}

func dog(r *rand.Rand) string {
	return getRandValue(r, []string{"animal", "dog"})
}

// Bird will return a random bird species
func Bird() string {
	return bird(globalFaker.Rand)
}

// Bird will return a random bird species
func (f *Faker) Bird() string {
	return bird(f.Rand)
}

func bird(r *rand.Rand) string {
	return getRandValue(r, []string{"animal", "bird"})
}

func addAnimalLookup() {
	AddFuncLookup("petname", Info{
		Display:     "Pet Name",
		Category:    "animal",
		Description: "Affectionate nickname given to a pet",
		Example:     "Ozzy Pawsborne",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return petName(r), nil
		},
	})

	AddFuncLookup("animal", Info{
		Display:     "Animal",
		Category:    "animal",
		Description: "Living creature with the ability to move, eat, and interact with its environment",
		Example:     "elk",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return animal(r), nil
		},
	})

	AddFuncLookup("animaltype", Info{
		Display:     "Animal Type",
		Category:    "animal",
		Description: "Type of animal, such as mammals, birds, reptiles, etc.",
		Example:     "amphibians",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return animalType(r), nil
		},
	})

	AddFuncLookup("farmanimal", Info{
		Display:     "Farm Animal",
		Category:    "animal",
		Description: "Animal name commonly found on a farm",
		Example:     "Chicken",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return farmAnimal(r), nil
		},
	})

	AddFuncLookup("cat", Info{
		Display:     "Cat",
		Category:    "animal",
		Description: "Various breeds that define different cats",
		Example:     "Chausie",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return cat(r), nil
		},
	})

	AddFuncLookup("dog", Info{
		Display:     "Dog",
		Category:    "animal",
		Description: "Various breeds that define different dogs",
		Example:     "Norwich Terrier",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return dog(r), nil
		},
	})

	AddFuncLookup("bird", Info{
		Display:     "Bird",
		Category:    "animal",
		Description: "Distinct species of birds",
		Example:     "goose",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return bird(r), nil
		},
	})
}
