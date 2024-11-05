package gofakeit

import (
	crand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"unicode"

	"github.com/brianvoe/gofakeit/v6/data"
)

const lowerStr = "abcdefghijklmnopqrstuvwxyz"
const upperStr = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
const numericStr = "0123456789"
const specialStr = "@#$%&?|!(){}<>=*+-_:;,."
const specialSafeStr = "@#$&?!-_*."
const spaceStr = " "
const allStr = lowerStr + upperStr + numericStr + specialStr + spaceStr
const vowels = "aeiou"
const hashtag = '#'
const questionmark = '?'
const dash = '-'
const base58 = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
const minUint = 0
const maxUint = ^uint(0)
const minInt = -maxInt - 1
const maxInt = int(^uint(0) >> 1)

// Seed will set the global random value. Setting seed to 0 will use crypto/rand
func Seed(seed int64) {
	if seed == 0 {
		binary.Read(crand.Reader, binary.BigEndian, &seed)
		globalFaker.Rand.Seed(seed)
	} else {
		globalFaker.Rand.Seed(seed)
	}
}

// Check if in lib
func dataCheck(dataVal []string) bool {
	var checkOk bool

	if len(dataVal) == 2 {
		_, checkOk = data.Data[dataVal[0]]
		if checkOk {
			_, checkOk = data.Data[dataVal[0]][dataVal[1]]
		}
	}

	return checkOk
}

// Get Random Value
func getRandValue(r *rand.Rand, dataVal []string) string {
	if !dataCheck(dataVal) {
		return ""
	}
	return data.Data[dataVal[0]][dataVal[1]][r.Intn(len(data.Data[dataVal[0]][dataVal[1]]))]
}

// Replace # with numbers
func replaceWithNumbers(r *rand.Rand, str string) string {
	if str == "" {
		return str
	}
	bytestr := []byte(str)
	for i := 0; i < len(bytestr); i++ {
		if bytestr[i] == hashtag {
			bytestr[i] = byte(randDigit(r))
		}
	}
	if bytestr[0] == '0' {
		bytestr[0] = byte(r.Intn(8)+1) + '0'
	}

	return string(bytestr)
}

// Replace ? with ASCII lowercase letters
func replaceWithLetters(r *rand.Rand, str string) string {
	if str == "" {
		return str
	}
	bytestr := []byte(str)
	for i := 0; i < len(bytestr); i++ {
		if bytestr[i] == questionmark {
			bytestr[i] = byte(randLetter(r))
		}
	}

	return string(bytestr)
}

// Replace ? with ASCII lowercase letters between a and f
func replaceWithHexLetters(r *rand.Rand, str string) string {
	if str == "" {
		return str
	}
	bytestr := []byte(str)
	for i := 0; i < len(bytestr); i++ {
		if bytestr[i] == questionmark {
			bytestr[i] = byte(randHexLetter(r))
		}
	}

	return string(bytestr)
}

// Generate random lowercase ASCII letter
func randLetter(r *rand.Rand) rune {
	allLetters := upperStr + lowerStr
	return rune(allLetters[r.Intn(len(allLetters))])
}

func randCharacter(r *rand.Rand, s string) string {
	return string(s[r.Int63()%int64(len(s))])
}

// Generate random lowercase ASCII letter between a and f
func randHexLetter(r *rand.Rand) rune {
	return rune(byte(r.Intn(6)) + 'a')
}

// Generate random ASCII digit
func randDigit(r *rand.Rand) rune {
	return rune(byte(r.Intn(10)) + '0')
}

// Generate random integer between min and max
func randIntRange(r *rand.Rand, min, max int) int {
	// If they pass in the same number, just return that number
	if min == max {
		return min
	}

	// If they pass in a min that is bigger than max, swap them
	if min > max {
		ogmin := min
		min = max
		max = ogmin
	}

	// Figure out if the min/max numbers calculation
	// would cause a panic in the Int63() function.
	if max-min+1 > 0 {
		return min + int(r.Int63n(int64(max-min+1)))
	}

	// Loop through the range until we find a number that fits
	for {
		v := int(r.Uint64())
		if (v >= min) && (v <= max) {
			return v
		}
	}
}

// Generate random uint between min and max
func randUintRange(r *rand.Rand, min, max uint) uint {
	// If they pass in the same number, just return that number
	if min == max {
		return min
	}

	// If they pass in a min that is bigger than max, swap them
	if min > max {
		ogmin := min
		min = max
		max = ogmin
	}

	// Figure out if the min/max numbers calculation
	// would cause a panic in the Int63() function.
	if int(max)-int(min)+1 > 0 {
		return uint(r.Intn(int(max)-int(min)+1) + int(min))
	}

	// Loop through the range until we find a number that fits
	for {
		v := uint(r.Uint64())
		if (v >= min) && (v <= max) {
			return v
		}
	}
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(math.Floor(num*output)) / output
}

func equalSliceString(a, b []string) bool {
	sizeA, sizeB := len(a), len(b)
	if sizeA != sizeB {
		return false
	}

	for i, va := range a {
		vb := b[i]

		if va != vb {
			return false
		}
	}
	return true
}

func equalSliceInt(a, b []int) bool {
	sizeA, sizeB := len(a), len(b)
	if sizeA != sizeB {
		return false
	}

	for i, va := range a {
		vb := b[i]

		if va != vb {
			return false
		}
	}
	return true
}

func equalSliceInterface(a, b []any) bool {
	sizeA, sizeB := len(a), len(b)
	if sizeA != sizeB {
		return false
	}

	for i, va := range a {
		if !reflect.DeepEqual(va, b[i]) {
			return false
		}
	}
	return true
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func anyToString(a any) string {
	if a == nil {
		return ""
	}

	// If it's a slice of bytes or struct, unmarshal it into an interface
	if bytes, ok := a.([]byte); ok {
		return string(bytes)
	}

	// If it's a struct, map, or slice, convert to JSON
	switch reflect.TypeOf(a).Kind() {
	case reflect.Struct, reflect.Map, reflect.Slice:
		b, err := json.Marshal(a)
		if err == nil {
			return string(b)
		}
	}

	return fmt.Sprintf("%v", a)
}

// Title returns a copy of the string s with all Unicode letters that begin words
// mapped to their Unicode title case
func title(s string) string {
	// isSeparator reports whether the rune could mark a word boundary
	isSeparator := func(r rune) bool {
		// ASCII alphanumerics and underscore are not separators
		if r <= 0x7F {
			switch {
			case '0' <= r && r <= '9':
				return false
			case 'a' <= r && r <= 'z':
				return false
			case 'A' <= r && r <= 'Z':
				return false
			case r == '_':
				return false
			}
			return true
		}

		// Letters and digits are not separators
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return false
		}

		// Otherwise, all we can do for now is treat spaces as separators.
		return unicode.IsSpace(r)
	}

	prev := ' '
	return strings.Map(
		func(r rune) rune {
			if isSeparator(prev) {
				prev = r
				return unicode.ToTitle(r)
			}
			prev = r
			return r
		},
		s)
}

func funcLookupSplit(str string) []string {
	out := []string{}
	for str != "" {
		if strings.HasPrefix(str, "[") {
			startIndex := strings.Index(str, "[")
			endIndex := strings.Index(str, "]")
			val := str[(startIndex) : endIndex+1]
			out = append(out, strings.TrimSpace(val))
			str = strings.Replace(str, val, "", 1)

			// Trim off comma if it has it
			if strings.HasPrefix(str, ",") {
				str = strings.Replace(str, ",", "", 1)
			}
		} else {
			strSplit := strings.SplitN(str, ",", 2)
			strSplitLen := len(strSplit)
			if strSplitLen >= 1 {
				out = append(out, strings.TrimSpace(strSplit[0]))
			}
			if strSplitLen >= 2 {
				str = strSplit[1]
			} else {
				str = ""
			}
		}
	}

	return out
}

// Used for parsing the tag in a struct
func parseNameAndParamsFromTag(tag string) (string, string) {
	// Trim the curly on the beginning and end
	tag = strings.TrimLeft(tag, "{")
	tag = strings.TrimRight(tag, "}")
	// Check if has params separated by :
	fNameSplit := strings.SplitN(tag, ":", 2)
	fName := ""
	fParams := ""
	if len(fNameSplit) >= 1 {
		fName = fNameSplit[0]
	}
	if len(fNameSplit) >= 2 {
		fParams = fNameSplit[1]
	}
	return fName, fParams
}

// Used for parsing map params
func parseMapParams(info *Info, fParams string) *MapParams {
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
	if mapParams.Size() > 0 {
		return mapParams
	} else {
		return nil
	}
}

// Used for splitting the values
func addSplitValsToMapParams(splitVals []string, info *Info, mapParams *MapParams) *MapParams {
	for ii := 0; ii < len(splitVals); ii++ {
		if len(info.Params)-1 >= ii {
			if strings.HasPrefix(splitVals[ii], "[") {
				lookupSplits := funcLookupSplit(strings.TrimRight(strings.TrimLeft(splitVals[ii], "["), "]"))
				for _, v := range lookupSplits {
					mapParams.Add(info.Params[ii].Field, v)
				}
			} else {
				mapParams.Add(info.Params[ii].Field, splitVals[ii])
			}
		}
	}
	return mapParams
}
