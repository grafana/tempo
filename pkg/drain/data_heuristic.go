package drain

import "unicode"

type IsDataHeuristic func(token string) bool

// hexWithAtLeastOneDigit checks if the token contains at least one digit and is hex-like or guid-like.
func hexWithAtLeastOneDigit(token string) bool {
	atLeastOneDigit := false
	for _, c := range token {
		if unicode.IsNumber(c) {
			atLeastOneDigit = true
			continue
		}
		if unicode.Is(unicode.Hex_Digit, c) || unicode.IsPunct(c) {
			continue
		}
		// Non-hex, non-punct found.
		return false
	}
	// All hex-like or guid-like values
	// But still require at least one digit.
	return atLeastOneDigit
}

// significantNumbers checks if the token is comprised of at least 25% numbers.
func significantNumbers(token string) bool {
	numberCount := 0
	for _, c := range token {
		if unicode.IsNumber(c) {
			numberCount++
		}
	}
	return numberCount > max(len(token)/4, 1)
}

// defaultIsDataHeuristic is the default implementation of the IsDataHeuristic
// interface. It checks if the token is likely data based on a set of heuristics.
// This is the second main point of customization for the DRAIN algorithm, and is
// largely domain-specific.
func defaultIsDataHeuristic(token string) bool {
	return hexWithAtLeastOneDigit(token) || significantNumbers(token)
}
