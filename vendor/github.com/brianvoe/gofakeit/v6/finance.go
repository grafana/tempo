package gofakeit

import (
	"math/rand"
	"strconv"
	"unicode"
)

const cusipStr = upperStr + numericStr

// CUSIP
func Cusip() string {
	return cusip(globalFaker.Rand)
}

func (f *Faker) Cusip() string {
	return cusip(f.Rand)
}

func cusip(r *rand.Rand) string {
	cusipBytes := make([]byte, 8)
	for i := 0; i < len(cusipBytes); i++ {
		cusipBytes[i] = byte(cusipStr[r.Intn(len(cusipStr))])
	}

	baseCusip := string(cusipBytes)

	chkDigit := cusipChecksumDigit(baseCusip)
	return baseCusip + chkDigit
}

// ISIN
func Isin() string {
	return isin(globalFaker.Rand)
}

func (f *Faker) Isin() string {
	return isin(f.Rand)
}

func isin(r *rand.Rand) string {
	countryCode := CountryAbr()
	nsin := cusip(r)
	isinChkDig := isinChecksumDigit(countryCode + nsin)
	return countryCode + nsin + isinChkDig
}

// cusipChecksumDigit returns the checksum digit for a CUSIP
func cusipChecksumDigit(cusip string) string {
	sum := 0
	for i, c := range cusip {
		v := 0
		if unicode.IsDigit(c) {
			v = int(c - '0')
		}
		if unicode.IsLetter(c) {
			//0-indexed ordinal position of Letter + 10
			v = int(c-'A') + 10
		}
		if i%2 != 0 {
			// Multiply odd digits by two
			v = v * 2
		}

		sum = sum + int(v/10) + v%10
	}

	return strconv.Itoa((10 - (sum % 10)) % 10)
}

// isinChecksumDigit returns the checksum digit for an ISIN
func isinChecksumDigit(isin string) string {
	isinDigits := make([]int, 0)
	for _, c := range isin {
		if unicode.IsLetter(c) {
			letterVal := int(c) - 55
			// Each digit is added as a separate value
			isinDigits = append(isinDigits, letterVal/10)
			isinDigits = append(isinDigits, letterVal%10)
		}
		if unicode.IsDigit(c) {
			isinDigits = append(isinDigits, int(c-'0'))
		}
	}

	oddSum := 0
	evenSum := 0

	// Take the per digit sum of the digitized ISIN, doubling even indexed digits
	for i, d := range isinDigits {
		if i%2 == 0 {
			elem := 2 * d
			if elem > 9 {
				// If the element now has two digits, sum those digits
				elem = (elem % 10) + (elem / 10)
			}
			evenSum += elem
		} else {
			oddSum += d
		}
	}

	return strconv.Itoa((10 - (oddSum+evenSum)%10) % 10)
}

// Lookup Adds
func addFinanceLookup() {
	AddFuncLookup("cusip", Info{
		Display:     "CUSIP",
		Category:    "finance",
		Description: "Unique identifier for securities, especially bonds, in the United States and Canada",
		Example:     "38259P508",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return cusip(r), nil
		},
	})
	AddFuncLookup("isin", Info{
		Display:     "ISIN",
		Category:    "finance",
		Description: "International standard code for uniquely identifying securities worldwide",
		Example:     "CVLRQCZBXQ97",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return isin(r), nil
		},
	})
}
