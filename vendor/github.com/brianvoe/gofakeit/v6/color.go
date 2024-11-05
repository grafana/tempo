package gofakeit

import (
	"math/rand"

	"github.com/brianvoe/gofakeit/v6/data"
)

// Color will generate a random color string
func Color() string { return color(globalFaker.Rand) }

// Color will generate a random color string
func (f *Faker) Color() string { return color(f.Rand) }

func color(r *rand.Rand) string { return getRandValue(r, []string{"color", "full"}) }

// NiceColor will generate a random safe color string
func NiceColors() []string { return niceColors(globalFaker.Rand) }

// NiceColor will generate a random safe color string
func (f *Faker) NiceColors() []string { return niceColors(f.Rand) }

func niceColors(r *rand.Rand) []string {
	return data.ColorsNice[randIntRange(r, 0, len(data.ColorsNice)-1)]
}

// SafeColor will generate a random safe color string
func SafeColor() string { return safeColor(globalFaker.Rand) }

// SafeColor will generate a random safe color string
func (f *Faker) SafeColor() string { return safeColor(f.Rand) }

func safeColor(r *rand.Rand) string { return getRandValue(r, []string{"color", "safe"}) }

// HexColor will generate a random hexadecimal color string
func HexColor() string { return hexColor(globalFaker.Rand) }

// HexColor will generate a random hexadecimal color string
func (f *Faker) HexColor() string { return hexColor(f.Rand) }

func hexColor(r *rand.Rand) string {
	color := make([]byte, 6)
	hashQuestion := []byte("?#")
	for i := 0; i < 6; i++ {
		color[i] = hashQuestion[r.Intn(2)]
	}

	return "#" + replaceWithHexLetters(r, replaceWithNumbers(r, string(color)))
}

// RGBColor will generate a random int slice color
func RGBColor() []int { return rgbColor(globalFaker.Rand) }

// RGBColor will generate a random int slice color
func (f *Faker) RGBColor() []int { return rgbColor(f.Rand) }

func rgbColor(r *rand.Rand) []int {
	return []int{randIntRange(r, 0, 255), randIntRange(r, 0, 255), randIntRange(r, 0, 255)}
}

func addColorLookup() {
	AddFuncLookup("color", Info{
		Display:     "Color",
		Category:    "color",
		Description: "Hue seen by the eye, returns the name of the color like red or blue",
		Example:     "MediumOrchid",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return color(r), nil
		},
	})

	AddFuncLookup("nicecolors", Info{
		Display:     "Nice Colors",
		Category:    "color",
		Description: "Attractive and appealing combinations of colors, returns an list of color hex codes",
		Example:     `["#cfffdd","#b4dec1","#5c5863","#a85163","#ff1f4c"]`,
		Output:      "[]string",
		ContentType: "application/json",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return color(r), nil
		},
	})

	AddFuncLookup("safecolor", Info{
		Display:     "Safe Color",
		Category:    "color",
		Description: "Colors displayed consistently on different web browsers and devices",
		Example:     "black",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return safeColor(r), nil
		},
	})

	AddFuncLookup("hexcolor", Info{
		Display:     "Hex Color",
		Category:    "color",
		Description: "Six-digit code representing a color in the color model",
		Example:     "#a99fb4",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return hexColor(r), nil
		},
	})

	AddFuncLookup("rgbcolor", Info{
		Display:     "RGB Color",
		Category:    "color",
		Description: "Color defined by red, green, and blue light values",
		Example:     "[85, 224, 195]",
		Output:      "[]int",
		ContentType: "application/json",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return rgbColor(r), nil
		},
	})
}
