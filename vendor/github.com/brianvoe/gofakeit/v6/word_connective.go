package gofakeit

import "math/rand"

// Connective will generate a random connective
func Connective() string { return connective(globalFaker.Rand) }

// Connective will generate a random connective
func (f *Faker) Connective() string { return connective(f.Rand) }

func connective(r *rand.Rand) string {
	var connectiveType = map[int]string{
		0: "connective_time",
		1: "connective_comparative",
		2: "connective_complaint",
		3: "connective_listing",
		4: "connective_casual",
		5: "connective_examplify",
	}
	return getRandValue(r, []string{"word", connectiveType[number(r, 0, 5)]})
}

// ConnectiveTime will generate a random connective time
func ConnectiveTime() string { return connectiveTime(globalFaker.Rand) }

// ConnectiveTime will generate a random connective time

func (f *Faker) ConnectiveTime() string { return connectiveTime(f.Rand) }

func connectiveTime(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "connective_time"})
}

// ConnectiveComparative will generate a random comparative connective
func ConnectiveComparative() string { return connectiveComparative(globalFaker.Rand) }

// ConnectiveComparative will generate a random comparative connective
func (f *Faker) ConnectiveComparative() string { return connectiveComparative(f.Rand) }

func connectiveComparative(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "connective_comparative"})
}

// ConnectiveComplaint will generate a random complaint connective
func ConnectiveComplaint() string { return connectiveComplaint(globalFaker.Rand) }

// ConnectiveComplaint will generate a random complaint connective
func (f *Faker) ConnectiveComplaint() string { return connectiveComplaint(f.Rand) }

func connectiveComplaint(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "connective_complaint"})
}

// ConnectiveListing will generate a random listing connective
func ConnectiveListing() string { return connectiveListing(globalFaker.Rand) }

// ConnectiveListing will generate a random listing connective
func (f *Faker) ConnectiveListing() string { return connectiveListing(f.Rand) }

func connectiveListing(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "connective_listing"})
}

// ConnectiveCasual will generate a random casual connective
func ConnectiveCasual() string { return connectiveCasual(globalFaker.Rand) }

// ConnectiveCasual will generate a random casual connective
func (f *Faker) ConnectiveCasual() string { return connectiveCasual(f.Rand) }

func connectiveCasual(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "connective_casual"})
}

// ConnectiveExamplify will generate a random examplify connective
func ConnectiveExamplify() string { return connectiveExamplify(globalFaker.Rand) }

// ConnectiveExamplify will generate a random examplify connective
func (f *Faker) ConnectiveExamplify() string { return connectiveExamplify(f.Rand) }

func connectiveExamplify(r *rand.Rand) string {
	return getRandValue(r, []string{"word", "connective_examplify"})
}

func addWordConnectiveLookup() {
	AddFuncLookup("connective", Info{
		Display:     "Connective",
		Category:    "word",
		Description: "Word used to connect words or sentences",
		Example:     "such as",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return connective(r), nil
		},
	})

	AddFuncLookup("connectivetime", Info{
		Display:     "Connective Time",
		Category:    "word",
		Description: "Connective word used to indicate a temporal relationship between events or actions",
		Example:     "finally",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return connectiveTime(r), nil
		},
	})

	AddFuncLookup("connectivecomparative", Info{
		Display:     "Connective Comparitive",
		Category:    "word",
		Description: "Connective word used to indicate a comparison between two or more things",
		Example:     "in addition",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return connectiveComparative(r), nil
		},
	})

	AddFuncLookup("connectivecomplaint", Info{
		Display:     "Connective Complaint",
		Category:    "word",
		Description: "Connective word used to express dissatisfaction or complaints about a situation",
		Example:     "besides",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return connectiveComplaint(r), nil
		},
	})

	AddFuncLookup("connectivelisting", Info{
		Display:     "Connective Listing",
		Category:    "word",
		Description: "Connective word used to list or enumerate items or examples",
		Example:     "firstly",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return connectiveListing(r), nil
		},
	})

	AddFuncLookup("connectivecasual", Info{
		Display:     "Connective Casual",
		Category:    "word",
		Description: "Connective word used to indicate a cause-and-effect relationship between events or actions",
		Example:     "an outcome of",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return connectiveCasual(r), nil
		},
	})

	AddFuncLookup("connectiveexamplify", Info{
		Display:     "Connective Examplify",
		Category:    "word",
		Description: "Connective word used to provide examples or illustrations of a concept or idea",
		Example:     "then",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return connectiveExamplify(r), nil
		},
	})
}
