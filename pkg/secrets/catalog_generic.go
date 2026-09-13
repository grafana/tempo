package secrets

import (
	"regexp"
	"regexp/syntax"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// This is an ordinary additional catalog heuristic, not a fallback-priority
// policy. Only this rule applies the pinned value-based candidate filters;
// specific-provider and custom-rule acceptance remains independent.
var genericRuleSpecs = []catalogRuleSpec{
	{
		ID:              genericAPIKeyRuleID,
		Regex:           genericAPIKeyRegex,
		Keywords:        []string{"access", "api", "auth", keyField, "credential", "creds", passwdField, passwordField, secretField, tokenField},
		Entropy:         3.5,
		SecretGroup:     1,
		Source:          gitleaksBaselineSource,
		Description:     "Assigned API-key-like candidates with strict entropy and pinned generic-only nonsecret filters; not issuer validation.",
		ValidateContext: validateGenericAPIKey,
	},
}

var (
	genericSecretFilter = sync.OnceValue(func() genericExclusionFilter {
		return newGenericExclusionFilter(genericSecretFilterRegex)
	})
	genericMatchFilter = sync.OnceValue(func() genericExclusionFilter {
		return newGenericExclusionFilter(genericMatchFilterRegex)
	})
	genericLineFilter = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(genericLineFilterRegex)
	})
	genericStopMatcher = sync.OnceValue(func() *keywordMatcher {
		keywords := make([]matcherKeyword, len(genericStopwords))
		for i, word := range genericStopwords {
			// The shared builder simple-folds keywords. Restrict its input to
			// lowercase ASCII so it builds exactly the pinned stopword language.
			if word == "" || !isASCIIKeyword(word) || word != strings.ToLower(word) {
				panic("invalid native generic stopword catalog")
			}
			keywords[i] = matcherKeyword{value: word}
		}
		matcher, err := newKeywordMatcher(keywords)
		if err != nil {
			panic("cannot compile native generic stopword catalog")
		}
		return &matcher
	})
)

// genericExclusionFilter preserves the union of the exact exclusion branches.
// Splitting the parsed alternation keeps each DFA small and avoids executing
// the full exclusion regexp when only one branch remains possible. Assertions
// and case-folding flags stay in each branch; DFAs only reject impossible matches.
type genericExclusionFilter []genericExclusionBranch

type genericExclusionBranch struct {
	regex     *regexp.Regexp
	rejection *rejectionFilter
}

func newGenericExclusionFilter(pattern string) genericExclusionFilter {
	parsed, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		panic(err)
	}
	branches := []*syntax.Regexp{parsed}
	if parsed.Op == syntax.OpAlternate {
		branches = parsed.Sub
	}
	filters := make(genericExclusionFilter, 0, len(branches))
	for _, branch := range branches {
		rejection, _ := newRejectionFilter(branch, false)
		filters = append(filters, genericExclusionBranch{
			regex:     regexp.MustCompile(branch.String()),
			rejection: rejection,
		})
	}
	return filters
}

func (f genericExclusionFilter) MatchString(value string) bool {
	for _, branch := range f {
		if (branch.rejection == nil || branch.rejection.mayMatch(value)) && branch.regex.MatchString(value) {
			return true
		}
	}
	return false
}

func validateGenericAPIKey(value string, matchStart, matchEnd int, secret string) contextValidation {
	if genericSecretFilter().MatchString(secret) || containsGenericStopword(secret) {
		return contextValidation{}
	}
	fullMatch := strings.Trim(value[matchStart:matchEnd], "\n")
	if genericMatchFilter().MatchString(fullMatch) {
		return contextValidation{}
	}
	if !genericLineFilter().MatchString(genericMatchLines(value, matchStart, matchEnd)) {
		return contextValidation{accepted: true}
	}
	// A non-multiline match in a rejected line establishes the same rejection
	// for following matches on that line. Preserve the newline itself: a later
	// match may begin there and belong to the next, accepted line.
	if !strings.ContainsRune(value[matchStart:matchEnd], '\n') {
		if next := strings.IndexByte(value[matchEnd:], '\n'); next >= 0 {
			return contextValidation{rejectUntil: matchEnd + next}
		}
		// Pinned unterminated-last-line context retains the value prefix, so a
		// matching line filter also rejects every later candidate in that tail.
		return contextValidation{rejectUntil: len(value)}
	}
	return contextValidation{}
}

// containsGenericStopword reuses the immutable native Aho-Corasick transitions,
// stopping at the first output. It implements substring search after
// strings.ToLower without allocating a lowercase copy or rescanning once per
// stopword. In particular, long-s must NOT acquire the ASCII s equivalence used
// by the catalog's regexp keyword prefilter; Kelvin sign does lowercase to k.
func containsGenericStopword(value string) bool {
	matcher := genericStopMatcher()
	state := uint16(0)
	for i := 0; i < len(value); {
		b := value[i]
		i++
		if b >= utf8.RuneSelf {
			r, width := utf8.DecodeRuneInString(value[i-1:])
			i += width - 1
			r = unicode.ToLower(r)
			if r >= utf8.RuneSelf {
				state = 0
				continue
			}
			b = byte(r)
		} else {
			b = foldASCII(b)
		}
		state = matcher.states[state].next[b]
		if len(matcher.states[state].outputs) != 0 {
			return true
		}
	}
	return false
}

// genericMatchLines mirrors the pinned scanner's value-only location semantics,
// including its last-line artifacts: a match beginning on an unterminated last
// line retains the value prefix, while a multiline match ending there stops at
// the match end. Broadening that latter span could suppress candidates accepted
// by the baseline. Keep original offsets; never manufacture a candidate window.
func genericMatchLines(value string, matchStart, matchEnd int) string {
	for matchEnd > matchStart && value[matchEnd-1] == '\n' {
		matchEnd--
	}
	previousLine := strings.LastIndexByte(value[:matchStart+1], '\n')
	if strings.IndexByte(value[matchStart+1:], '\n') < 0 {
		if previousLine < 0 {
			return value
		}
		end := len(value)
		if offset := strings.IndexAny(value[matchEnd:], "\r\n"); offset >= 0 {
			end = matchEnd + offset
		}
		return value[:end]
	}
	start := max(0, previousLine)
	if offset := strings.IndexByte(value[matchEnd:], '\n'); offset >= 0 {
		return value[start : matchEnd+offset]
	}
	return value[start:matchEnd]
}
