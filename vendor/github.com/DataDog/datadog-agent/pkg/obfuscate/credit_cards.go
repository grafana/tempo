// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package obfuscate

// IsCardNumber checks if b could be a credit card number by checking the digit count and IIN prefix.
// If validateLuhn is true, the Luhn checksum is also applied to potential candidates.
func IsCardNumber(b string, validateLuhn bool) (ok bool) {
	//
	// Just credit card numbers for now, based on:
	// • https://baymard.com/checkout-usability/credit-card-patterns
	// • https://www.regular-expressions.info/creditcard.html
	//
	if len(b) == 0 {
		return false
	}
	if len(b) < 12 {
		// fast path: can not be a credit card
		return false
	}
	if b[0] != ' ' && b[0] != '-' && (b[0] < '0' || b[0] > '9') {
		// fast path: only valid characters are 0-9, space (" ") and dash("-")
		return false
	}
	prefix := 0                 // holds up to b[:6] digits as a numeric value (for example []byte{"523"} becomes int(523)) for checking prefixes
	count := 0                  // counts digits encountered
	foundPrefix := false        // reports whether we've detected a valid prefix
	recdigit := func(_ byte) {} // callback on each found digit; no-op by default (we only need this for Luhn)
	if validateLuhn {
		// we need Luhn checksum validation, so we have to take additional action
		// and record all digits found
		buf := make([]byte, 0, len(b))
		recdigit = func(b byte) { buf = append(buf, b) }
		defer func() {
			if !ok {
				// if isCardNumber returned false, it means that b can not be
				// a credit card number
				return
			}
			// potentially a credit card number, run the Luhn checksum
			ok = luhnValid(buf)
		}()
	}
loop:
	for i := range b {
		// We traverse and search b for a valid IIN credit card prefix based
		// on the digits found, ignoring spaces and dashes.
		// Source: https://www.regular-expressions.info/creditcard.html
		switch b[i] {
		case ' ', '-':
			// ignore space (' ') and dash ('-')
			continue loop
		}
		if b[i] < '0' || b[i] > '9' {
			// not a 0 to 9 digit; can not be a credit card number; abort
			return false
		}
		count++
		recdigit(b[i])
		if !foundPrefix {
			// we have not yet found a valid prefix so we convert the digits
			// that we have so far into a numeric value:
			prefix = prefix*10 + (int(b[i]) - '0')
			maybe, yes := validCardPrefix(prefix)
			if yes {
				// we've found a valid prefix; continue counting
				foundPrefix = true
			} else if !maybe {
				// this is not a valid prefix and we should not continue looking
				return false
			}
		}
		if count > 16 {
			// too many digits
			return false
		}
	}
	if count < 12 {
		// too few digits
		return false
	}
	return foundPrefix
}

// luhnValid checks that the number represented in the given string validates the Luhn Checksum algorithm.
// str is expected to contain exclusively digits at all positions.
//
// See:
// • https://en.wikipedia.org/wiki/Luhn_algorithm
// • https://dev.to/shiraazm/goluhn-a-simple-library-for-generating-calculating-and-verifying-luhn-numbers-588j
func luhnValid(str []byte) bool {
	var (
		sum int
		alt bool
	)
	n := len(str)
	for i := n - 1; i > -1; i-- {
		if str[i] < '0' || str[i] > '9' {
			return false // not a number!
		}
		mod := int(str[i] - 0x30) // convert byte to int
		if alt {
			mod *= 2
			if mod > 9 {
				mod = (mod % 10) + 1
			}
		}
		alt = !alt
		sum += mod
	}
	return sum%10 == 0
}

// validCardPrefix validates whether b is a valid card prefix. Maybe returns true if
// the prefix could be an IIN once more digits are revealed and yes reports whether
// b is a fully valid IIN.
//
// If yes is false and maybe is false, there is no reason to continue searching. The
// prefix is invalid.
//
// IMPORTANT: If adding new prefixes to this algorithm, make sure that you update
// the "maybe" clauses above, in the shorter prefixes than the one you are adding.
// This refers to the cases which return true, false.
//
// TODO(x): this whole code could be code generated from a prettier data structure.
// Ultimately, it could even be user-configurable.
func validCardPrefix(n int) (maybe, yes bool) {
	// Validates IIN prefix possibilities
	// Source: https://www.regular-expressions.info/creditcard.html
	if n > 699999 {
		// too long for any known prefix; stop looking
		return false, false
	}
	if n < 10 {
		switch n {
		case 1, 4:
			// 1 & 4 are valid IIN
			return false, true
		case 2, 3, 5, 6:
			// 2, 3, 5, 6 could be the start of valid IIN
			return true, false
		default:
			// invalid IIN
			return false, false
		}
	}
	if n < 100 {
		if (n >= 34 && n <= 39) ||
			(n >= 51 && n <= 55) ||
			n == 62 ||
			n == 65 {
			// 34-39, 51-55, 62, 65 are valid IIN
			return false, true
		}
		if n == 30 || n == 63 || n == 64 || n == 50 || n == 60 ||
			(n >= 22 && n <= 27) || (n >= 56 && n <= 58) || (n >= 60 && n <= 69) {
			// 30, 63, 64, 50, 60, 22-27, 56-58, 60-69 may end up as valid IIN
			return true, false
		}
	}
	if n < 1000 {
		if (n >= 300 && n <= 305) ||
			(n >= 644 && n <= 649) ||
			n == 309 ||
			n == 636 {
			// 300‑305, 309, 636, 644‑649 are valid IIN
			return false, true
		}
		if (n >= 352 && n <= 358) || n == 501 || n == 601 ||
			(n >= 222 && n <= 272) || (n >= 500 && n <= 509) ||
			(n >= 560 && n <= 589) || (n >= 600 && n <= 699) {
			// 352-358, 501, 601, 222-272, 500-509, 560-589, 600-699 may be a 4 or 6 digit IIN prefix
			return true, false
		}
	}
	if n < 10000 {
		if (n >= 3528 && n <= 3589) ||
			n == 5019 ||
			n == 6011 {
			// 3528‑3589, 5019, 6011 are valid IINs
			return false, true
		}
		if (n >= 2221 && n <= 2720) || (n >= 5000 && n <= 5099) ||
			(n >= 5600 && n <= 5899) || (n >= 6000 && n <= 6999) {
			// maybe a 6-digit IIN
			return true, false
		}
	}
	if n < 100000 {
		if (n >= 22210 && n <= 27209) ||
			(n >= 50000 && n <= 50999) ||
			(n >= 56000 && n <= 58999) ||
			(n >= 60000 && n <= 69999) {
			// maybe a 6-digit IIN
			return true, false
		}
	}
	if n < 1000000 {
		if (n >= 222100 && n <= 272099) ||
			(n >= 500000 && n <= 509999) ||
			(n >= 560000 && n <= 589999) ||
			(n >= 600000 && n <= 699999) {
			// 222100‑272099, 500000‑509999, 560000‑589999, 600000‑699999 are valid IIN
			return false, true
		}
	}
	// unknown IIN
	return false, false
}
