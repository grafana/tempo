package gofakeit

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"regexp/syntax"
	"strings"
)

// Generate fake information from given string.
// Replaceable values should be within {}
//
// Functions
// Ex: {firstname} - billy
// Ex: {sentence:3} - Record river mind.
// Ex: {number:1,10} - 4
// Ex: {uuid} - 590c1440-9888-45b0-bd51-a817ee07c3f2
//
// Letters/Numbers
// Ex: ### - 481 - random numbers
// Ex: ??? - fda - random letters
//
// For a complete list of runnable functions use FuncsLookup
func Generate(dataVal string) string { return generate(globalFaker.Rand, dataVal) }

// Generate fake information from given string.
// Replaceable values should be within {}
//
// Functions
// Ex: {firstname} - billy
// Ex: {sentence:3} - Record river mind.
// Ex: {number:1,10} - 4
// Ex: {uuid} - 590c1440-9888-45b0-bd51-a817ee07c3f2
//
// Letters/Numbers
// Ex: ### - 481 - random numbers
// Ex: ??? - fda - random letters
//
// For a complete list of runnable functions use FuncsLookup
func (f *Faker) Generate(dataVal string) string { return generate(f.Rand, dataVal) }

func generate(r *rand.Rand, dataVal string) string {
	// Replace # with numbers and ? with letters
	dataVal = replaceWithNumbers(r, dataVal)
	dataVal = replaceWithLetters(r, dataVal)

	// Check if string has any replaceable values
	if !strings.Contains(dataVal, "{") && !strings.Contains(dataVal, "}") {
		return dataVal
	}

	// Variables to identify the index in which it exists
	startCurly := -1
	startCurlyIgnore := []int{}
	endCurly := -1
	endCurlyIgnore := []int{}

	// Loop through string characters
	for i := 0; i < len(dataVal); i++ {
		// Check for ignores if equal skip
		shouldSkip := false
		for _, igs := range startCurlyIgnore {
			if i == igs {
				shouldSkip = true
			}
		}
		for _, ige := range endCurlyIgnore {
			if i == ige {
				shouldSkip = true
			}
		}
		if shouldSkip {
			continue
		}

		// Identify items between brackets. Ex: {firstname}
		if string(dataVal[i]) == "{" {
			startCurly = i
			continue
		}
		if startCurly != -1 && string(dataVal[i]) == "}" {
			endCurly = i
		}
		if startCurly == -1 || endCurly == -1 {
			continue
		}

		// Get the value between brackets
		fParts := dataVal[startCurly+1 : endCurly]

		// Check if has params separated by :
		fNameSplit := strings.SplitN(fParts, ":", 2)
		fName := ""
		fParams := ""
		if len(fNameSplit) >= 1 {
			fName = fNameSplit[0]
		}
		if len(fNameSplit) >= 2 {
			fParams = fNameSplit[1]
		}

		// Check to see if its a replaceable lookup function
		if info := GetFuncLookup(fName); info != nil {
			// Get parameters, make sure params and the split both have values
			mapParams := NewMapParams()
			paramsLen := len(info.Params)

			// If just one param and its a string simply just pass it
			if paramsLen == 1 && info.Params[0].Type == "string" {
				mapParams.Add(info.Params[0].Field, fParams)
			} else if paramsLen > 0 && fParams != "" {
				splitVals := funcLookupSplit(fParams)
				mapParams = addSplitValsToMapParams(splitVals, info, mapParams)
			}
			if mapParams.Size() == 0 {
				mapParams = nil
			}

			// Call function
			fValue, err := info.Generate(r, mapParams, info)
			if err != nil {
				// If we came across an error just dont replace value
				dataVal = strings.Replace(dataVal, "{"+fParts+"}", err.Error(), 1)
			} else {
				// Successfully found, run replace with new value
				dataVal = strings.Replace(dataVal, "{"+fParts+"}", fmt.Sprintf("%v", fValue), 1)
			}

			// Reset the curly index back to -1 and reset ignores
			startCurly = -1
			startCurlyIgnore = []int{}
			endCurly = -1
			endCurlyIgnore = []int{}
			i = -1 // Reset back to the start of the string
			continue
		}

		// Couldnt find anything - mark curly brackets to skip and rerun
		startCurlyIgnore = append(startCurlyIgnore, startCurly)
		endCurlyIgnore = append(endCurlyIgnore, endCurly)

		// Reset the curly index back to -1
		startCurly = -1
		endCurly = -1
		i = -1 // Reset back to the start of the string
		continue
	}

	return dataVal
}

// FixedWidthOptions defines values needed for csv generation
type FixedWidthOptions struct {
	RowCount int     `json:"row_count" xml:"row_count" fake:"{number:1,10}"`
	Fields   []Field `json:"fields" xml:"fields" fake:"{fields}"`
}

// FixedWidth generates an table of random data in fixed width format
// A nil FixedWidthOptions returns a randomly structured FixedWidth.
func FixedWidth(co *FixedWidthOptions) (string, error) { return fixeWidthFunc(globalFaker.Rand, co) }

// FixedWidth generates an table of random data in fixed width format
// A nil FixedWidthOptions returns a randomly structured FixedWidth.
func (f *Faker) FixedWidth(co *FixedWidthOptions) (string, error) { return fixeWidthFunc(f.Rand, co) }

// Function to generate a fixed width document
func fixeWidthFunc(r *rand.Rand, co *FixedWidthOptions) (string, error) {
	// If we didn't get FixedWidthOptions, create a new random one
	if co == nil {
		co = &FixedWidthOptions{}
	}

	// Make sure you set a row count
	if co.RowCount <= 0 {
		co.RowCount = r.Intn(10) + 1
	}

	// Check fields
	if len(co.Fields) <= 0 {
		// Create random fields
		co.Fields = []Field{
			{Name: "Name", Function: "{firstname} {lastname}"},
			{Name: "Email", Function: "email"},
			{Name: "Password", Function: "password", Params: MapParams{"special": {"false"}, "space": {"false"}}},
		}
	}

	data := [][]string{}
	hasHeader := false

	// Loop through fields, generate data and add to data array
	for _, field := range co.Fields {
		// Start new row
		row := []string{}

		// Add name to first value
		if field.Name != "" {
			hasHeader = true
		}
		row = append(row, field.Name)

		// Get function
		funcInfo := GetFuncLookup(field.Function)
		var value any
		if funcInfo == nil {
			// Try to run the function through generate
			for i := 0; i < co.RowCount; i++ {
				row = append(row, generate(r, field.Function))
			}
		} else {
			// Generate function value
			var err error
			for i := 0; i < co.RowCount; i++ {
				value, err = funcInfo.Generate(r, &field.Params, funcInfo)
				if err != nil {
					value = ""
				}

				// Add value to row
				row = append(row, anyToString(value))
			}
		}

		// Add row to data
		data = append(data, row)
	}

	var result strings.Builder

	// Calculate column widths
	colWidths := make([]int, len(data))
	for i, row := range data {
		for _, value := range row {
			width := len(value) + 5
			if width > colWidths[i] {
				colWidths[i] = width
			}
		}
	}

	// Append table rows to the string, excluding the entire row if the first value is empty
	for i := 0; i < len(data[0]); i++ {
		if !hasHeader && i == 0 {
			continue // Skip the entire column if the first value is empty
		}

		var resultRow strings.Builder
		for j, row := range data {
			resultRow.WriteString(fmt.Sprintf("%-*s", colWidths[j], row[i]))
		}

		// Trim trailing spaces
		result.WriteString(strings.TrimRight(resultRow.String(), " "))

		// Only add new line if not the last row
		if i != len(data[0])-1 {
			result.WriteString("\n")
		}
	}

	return result.String(), nil
}

// Regex will generate a string based upon a RE2 syntax
func Regex(regexStr string) string { return regex(globalFaker.Rand, regexStr) }

// Regex will generate a string based upon a RE2 syntax
func (f *Faker) Regex(regexStr string) string { return regex(f.Rand, regexStr) }

func regex(r *rand.Rand, regexStr string) (gen string) {
	re, err := syntax.Parse(regexStr, syntax.Perl)
	if err != nil {
		return "Could not parse regex string"
	}

	// Panic catch
	defer func() {
		if r := recover(); r != nil {
			gen = fmt.Sprint(r)
			return

		}
	}()

	return regexGenerate(r, re, len(regexStr)*100)
}

func regexGenerate(ra *rand.Rand, re *syntax.Regexp, limit int) string {
	if limit <= 0 {
		panic("Length limit reached when generating output")
	}

	op := re.Op
	switch op {
	case syntax.OpNoMatch: // matches no strings
		// Do Nothing
	case syntax.OpEmptyMatch: // matches empty string
		return ""
	case syntax.OpLiteral: // matches Runes sequence
		var b strings.Builder
		for _, ru := range re.Rune {
			b.WriteRune(ru)
		}
		return b.String()
	case syntax.OpCharClass: // matches Runes interpreted as range pair list
		// number of possible chars
		sum := 0
		for i := 0; i < len(re.Rune); i += 2 {
			sum += int(re.Rune[i+1]-re.Rune[i]) + 1
			if re.Rune[i+1] == 0x10ffff { // rune range end
				sum = -1
				break
			}
		}

		// pick random char in range (inverse match group)
		if sum == -1 {
			chars := []uint8{}
			for j := 0; j < len(allStr); j++ {
				c := allStr[j]

				// Check c in range
				for i := 0; i < len(re.Rune); i += 2 {
					if rune(c) >= re.Rune[i] && rune(c) <= re.Rune[i+1] {
						chars = append(chars, c)
						break
					}
				}
			}
			if len(chars) > 0 {
				return string([]byte{chars[ra.Intn(len(chars))]})
			}
		}

		r := ra.Intn(int(sum))
		var ru rune
		sum = 0
		for i := 0; i < len(re.Rune); i += 2 {
			gap := int(re.Rune[i+1]-re.Rune[i]) + 1
			if sum+gap > r {
				ru = re.Rune[i] + rune(r-sum)
				break
			}
			sum += gap
		}

		return string(ru)
	case syntax.OpAnyCharNotNL, syntax.OpAnyChar: // matches any character(and except newline)
		return randCharacter(ra, allStr)
	case syntax.OpBeginLine: // matches empty string at beginning of line
	case syntax.OpEndLine: // matches empty string at end of line
	case syntax.OpBeginText: // matches empty string at beginning of text
	case syntax.OpEndText: // matches empty string at end of text
	case syntax.OpWordBoundary: // matches word boundary `\b`
	case syntax.OpNoWordBoundary: // matches word non-boundary `\B`
	case syntax.OpCapture: // capturing subexpression with index Cap, optional name Name
		return regexGenerate(ra, re.Sub0[0], limit)
	case syntax.OpStar: // matches Sub[0] zero or more times
		var b strings.Builder
		for i := 0; i < number(ra, 0, 10); i++ {
			for _, rs := range re.Sub {
				b.WriteString(regexGenerate(ra, rs, limit-b.Len()))
			}
		}
		return b.String()
	case syntax.OpPlus: // matches Sub[0] one or more times
		var b strings.Builder
		for i := 0; i < number(ra, 1, 10); i++ {
			for _, rs := range re.Sub {
				b.WriteString(regexGenerate(ra, rs, limit-b.Len()))
			}
		}
		return b.String()
	case syntax.OpQuest: // matches Sub[0] zero or one times
		var b strings.Builder
		for i := 0; i < number(ra, 0, 1); i++ {
			for _, rs := range re.Sub {
				b.WriteString(regexGenerate(ra, rs, limit-b.Len()))
			}
		}
		return b.String()
	case syntax.OpRepeat: // matches Sub[0] at least Min times, at most Max (Max == -1 is no limit)
		var b strings.Builder
		count := 0
		re.Max = int(math.Min(float64(re.Max), float64(10)))
		if re.Max > re.Min {
			count = ra.Intn(re.Max - re.Min + 1)
		}
		for i := 0; i < re.Min || i < (re.Min+count); i++ {
			for _, rs := range re.Sub {
				b.WriteString(regexGenerate(ra, rs, limit-b.Len()))
			}
		}
		return b.String()
	case syntax.OpConcat: // matches concatenation of Subs
		var b strings.Builder
		for _, rs := range re.Sub {
			b.WriteString(regexGenerate(ra, rs, limit-b.Len()))
		}
		return b.String()
	case syntax.OpAlternate: // matches alternation of Subs
		return regexGenerate(ra, re.Sub[number(ra, 0, len(re.Sub)-1)], limit)
	}

	return ""
}

// Map will generate a random set of map data
func Map() map[string]any { return mapFunc(globalFaker.Rand) }

// Map will generate a random set of map data
func (f *Faker) Map() map[string]any { return mapFunc(f.Rand) }

func mapFunc(r *rand.Rand) map[string]any {
	m := map[string]any{}

	randWordType := func() string {
		s := randomString(r, []string{"lorem", "bs", "job", "name", "address"})
		switch s {
		case "bs":
			return bs(r)
		case "job":
			return jobTitle(r)
		case "name":
			return name(r)
		case "address":
			return street(r) + ", " + city(r) + ", " + state(r) + " " + zip(r)
		}
		return word(r)
	}

	randSlice := func() []string {
		var sl []string
		for ii := 0; ii < number(r, 3, 10); ii++ {
			sl = append(sl, word(r))
		}
		return sl
	}

	for i := 0; i < number(r, 3, 10); i++ {
		t := randomString(r, []string{"string", "int", "float", "slice", "map"})
		switch t {
		case "string":
			m[word(r)] = randWordType()
		case "int":
			m[word(r)] = number(r, 1, 10000000)
		case "float":
			m[word(r)] = float32Range(r, 1, 1000000)
		case "slice":
			m[word(r)] = randSlice()
		case "map":
			mm := map[string]any{}
			tt := randomString(r, []string{"string", "int", "float", "slice"})
			switch tt {
			case "string":
				mm[word(r)] = randWordType()
			case "int":
				mm[word(r)] = number(r, 1, 10000000)
			case "float":
				mm[word(r)] = float32Range(r, 1, 1000000)
			case "slice":
				mm[word(r)] = randSlice()
			}
			m[word(r)] = mm
		}
	}

	return m
}

func addGenerateLookup() {
	AddFuncLookup("generate", Info{
		Display:     "Generate",
		Category:    "generate",
		Description: "Random string generated from string value based upon available data sets",
		Example:     "{firstname} {lastname} {email} - Markus Moen markusmoen@pagac.net",
		Output:      "string",
		Params: []Param{
			{Field: "str", Display: "String", Type: "string", Description: "String value to generate from"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			str, err := info.GetString(m, "str")
			if err != nil {
				return nil, err
			}

			// Limit the length of the string passed
			if len(str) > 1000 {
				return nil, errors.New("string length is too large. limit to 1000 characters")
			}

			return generate(r, str), nil
		},
	})

	AddFuncLookup("fixed_width", Info{
		Display:     "Fixed Width",
		Category:    "generate",
		Description: "Fixed width rows of output data based on input fields",
		Example: `Name               Email                          Password         Age
Markus Moen        sylvanmraz@murphy.net          6VlvH6qqXc7g     13
Alayna Wuckert     santinostanton@carroll.biz     g7sLrS0gEwLO     46
Lura Lockman       zacherykuhic@feil.name         S8gV7Z64KlHG     12`,
		Output:      "[]byte",
		ContentType: "text/plain",
		Params: []Param{
			{Field: "rowcount", Display: "Row Count", Type: "int", Default: "10", Description: "Number of rows"},
			{Field: "fields", Display: "Fields", Type: "[]Field", Description: "Fields name, function and params"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			co := FixedWidthOptions{}

			rowCount, err := info.GetInt(m, "rowcount")
			if err != nil {
				return nil, err
			}

			co.RowCount = rowCount

			fields, _ := info.GetStringArray(m, "fields")

			// Check to make sure fields has length
			if len(fields) > 0 {
				co.Fields = make([]Field, len(fields))
				for i, f := range fields {
					// Unmarshal fields string into fields array
					err = json.Unmarshal([]byte(f), &co.Fields[i])
					if err != nil {
						return nil, err
					}
				}
			} else {
				return nil, errors.New("missing fields")
			}

			out, err := fixeWidthFunc(r, &co)
			if err != nil {
				return nil, err
			}

			return out, nil
		},
	})

	AddFuncLookup("regex", Info{
		Display:     "Regex",
		Category:    "generate",
		Description: "Pattern-matching tool used in text processing to search and manipulate strings",
		Example:     "[abcdef]{5} - affec",
		Output:      "string",
		Params: []Param{
			{Field: "str", Display: "String", Type: "string", Description: "Regex RE2 syntax string"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			str, err := info.GetString(m, "str")
			if err != nil {
				return nil, err
			}

			// Limit the length of the string passed
			if len(str) > 500 {
				return nil, errors.New("string length is too large. limit to 500 characters")
			}

			return regex(r, str), nil
		},
	})

	AddFuncLookup("map", Info{
		Display:     "Map",
		Category:    "generate",
		Description: "Data structure that stores key-value pairs",
		Example: `{
	"software": 7518355,
	"that": ["despite", "pack", "whereas", "recently", "there", "anyone", "time", "read"],
	"use": 683598,
	"whom": "innovate",
	"yourselves": 1987784
}`,
		Output:      "map[string]any",
		ContentType: "application/json",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return mapFunc(r), nil
		},
	})
}
