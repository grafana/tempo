package gofakeit

import "math/rand"

// Username will generate a random username based upon picking a random lastname and random numbers at the end
func Username() string {
	return username(globalFaker.Rand)
}

// Username will generate a random username based upon picking a random lastname and random numbers at the end
func (f *Faker) Username() string {
	return username(f.Rand)
}

func username(r *rand.Rand) string {
	return getRandValue(r, []string{"person", "last"}) + replaceWithNumbers(r, "####")
}

// Password will generate a random password.
// Minimum number length of 5 if less than.
func Password(lower bool, upper bool, numeric bool, special bool, space bool, num int) string {
	return password(globalFaker.Rand, lower, upper, numeric, special, space, num)
}

// Password will generate a random password.
// Minimum number length of 5 if less than.
func (f *Faker) Password(lower bool, upper bool, numeric bool, special bool, space bool, num int) string {
	return password(f.Rand, lower, upper, numeric, special, space, num)
}

func password(r *rand.Rand, lower bool, upper bool, numeric bool, special bool, space bool, num int) string {
	// Make sure the num minimum is at least 5
	if num < 5 {
		num = 5
	}
	i := 0
	b := make([]byte, num)
	var passString string

	if lower {
		passString += lowerStr
		b[i] = lowerStr[r.Int63()%int64(len(lowerStr))]
		i++
	}
	if upper {
		passString += upperStr
		b[i] = upperStr[r.Int63()%int64(len(upperStr))]
		i++
	}
	if numeric {
		passString += numericStr
		b[i] = numericStr[r.Int63()%int64(len(numericStr))]
		i++
	}
	if special {
		passString += specialStr
		b[i] = specialStr[r.Int63()%int64(len(specialStr))]
		i++
	}
	if space {
		passString += spaceStr
		b[i] = spaceStr[r.Int63()%int64(len(spaceStr))]
		i++
	}

	// Set default if empty
	if passString == "" {
		passString = lowerStr + numericStr
	}

	// Loop through and add it up
	for i <= num-1 {
		b[i] = passString[r.Int63()%int64(len(passString))]
		i++
	}

	// Shuffle bytes
	for i := range b {
		j := r.Intn(i + 1)
		b[i], b[j] = b[j], b[i]
	}

	return string(b)
}

func addAuthLookup() {
	AddFuncLookup("username", Info{
		Display:     "Username",
		Category:    "auth",
		Description: "Generates a random username",
		Example:     "Daniel1364",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return username(r), nil
		},
	})

	AddFuncLookup("password", Info{
		Display:     "Password",
		Category:    "auth",
		Description: "Generates a random password",
		Example:     "EEP+wwpk 4lU-eHNXlJZ4n K9%v&TZ9e",
		Output:      "string",
		Params: []Param{
			{Field: "lower", Display: "Lower", Type: "bool", Default: "true", Description: "Whether or not to add lower case characters"},
			{Field: "upper", Display: "Upper", Type: "bool", Default: "true", Description: "Whether or not to add upper case characters"},
			{Field: "numeric", Display: "Numeric", Type: "bool", Default: "true", Description: "Whether or not to add numeric characters"},
			{Field: "special", Display: "Special", Type: "bool", Default: "true", Description: "Whether or not to add special characters"},
			{Field: "space", Display: "Space", Type: "bool", Default: "false", Description: "Whether or not to add spaces"},
			{Field: "length", Display: "Length", Type: "int", Default: "12", Description: "Number of characters in password"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			lower, err := info.GetBool(m, "lower")
			if err != nil {
				return nil, err
			}

			upper, err := info.GetBool(m, "upper")
			if err != nil {
				return nil, err
			}

			numeric, err := info.GetBool(m, "numeric")
			if err != nil {
				return nil, err
			}

			special, err := info.GetBool(m, "special")
			if err != nil {
				return nil, err
			}

			space, err := info.GetBool(m, "space")
			if err != nil {
				return nil, err
			}

			length, err := info.GetInt(m, "length")
			if err != nil {
				return nil, err
			}

			return password(r, lower, upper, numeric, special, space, length), nil
		},
	})
}
