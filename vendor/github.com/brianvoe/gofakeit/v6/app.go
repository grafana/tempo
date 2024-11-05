package gofakeit

import (
	"fmt"
	"math/rand"
)

// AppName will generate a random app name
func AppName() string {
	return appName(globalFaker.Rand)
}

// AppName will generate a random app name
func (f *Faker) AppName() string {
	return appName(f.Rand)
}

func appName(r *rand.Rand) string {
	name := ""
	switch number(r, 1, 3) {
	case 1:
		name = noun(r) + verb(r)
	case 2:
		name = color(r) + noun(r)
	case 3:
		name = animal(r) + verb(r)
	}

	return title(name)
}

// AppVersion will generate a random app version
func AppVersion() string {
	return appVersion(globalFaker.Rand)
}

// AppVersion will generate a random app version
func (f *Faker) AppVersion() string {
	return appVersion(f.Rand)
}

func appVersion(r *rand.Rand) string {
	return fmt.Sprintf("%d.%d.%d", number(r, 1, 5), number(r, 1, 20), number(r, 1, 20))
}

// AppAuthor will generate a random company or person name
func AppAuthor() string {
	return appAuthor(globalFaker.Rand)
}

// AppAuthor will generate a random company or person name
func (f *Faker) AppAuthor() string {
	return appAuthor(f.Rand)
}

func appAuthor(r *rand.Rand) string {
	if boolFunc(r) {
		return name(r)
	}

	return company(r)
}

func addAppLookup() {
	AddFuncLookup("appname", Info{
		Display:     "App Name",
		Category:    "app",
		Description: "Software program designed for a specific purpose or task on a computer or mobile device",
		Example:     "Parkrespond",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return appName(r), nil
		},
	})

	AddFuncLookup("appversion", Info{
		Display:     "App Version",
		Category:    "app",
		Description: "Particular release of an application in Semantic Versioning format",
		Example:     "1.12.14",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return appVersion(r), nil
		},
	})

	AddFuncLookup("appauthor", Info{
		Display:     "App Author",
		Category:    "app",
		Description: "Person or group creating and developing an application",
		Example:     "Qado Energy, Inc.",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return appAuthor(r), nil
		},
	})
}
