package secrets

import (
	"regexp/syntax"
	"slices"
	"strings"
	"unicode/utf8"
)

// ruleStrategy selects how a rule is evaluated around verified keyword offsets.
// Non-ASCII values require a separate proof for anchored byte-offset guidance.
type ruleStrategy uint8

const (
	// strategyFullScan runs the regex over the whole value. It is the reference behaviour and
	// the fallback whenever the analysis below cannot prove a cheaper strategy equivalent.
	strategyFullScan ruleStrategy = iota
	// strategyWindow runs the regex over merged windows when every match contains
	// a keyword and the regex has a bounded rune width. Byte bounds expand on
	// Unicode input, with complete real runes retained at both window edges.
	strategyWindow
	// strategyAnchored runs the regex anchored at keyword occurrences. It is exact when every
	// match begins with a keyword, optionally preceded by an optional run of a fixed byte class
	// (for example, an optional identifier prefix). Walking back from the keyword over that
	// class yields the leftmost start the unanchored search would have chosen, and the engine
	// anchored there reproduces the leftmost-first match, including its captures.
	strategyAnchored
)

const (
	// analysisMaxStrings bounds the finite languages enumerated while proving keyword coverage.
	analysisMaxStrings = 128
	// analysisMaxStringLength bounds the enumerated strings so the cross products stay cheap.
	analysisMaxStringLength = 64
	// analysisMinFactorLength drops single-byte factors: they are present in almost every
	// window and cost more to check than they save.
	analysisMinFactorLength = 2
	// analysisMaxFactors bounds the per-window factor checks.
	analysisMaxFactors = 4
)

// asciiSet is a 128-bit membership set over ASCII bytes.
type asciiSet [2]uint64

func (s *asciiSet) add(value byte) {
	s[value>>6] |= 1 << (value & 63)
}

func (s *asciiSet) has(value byte) bool {
	return value < 0x80 && s[value>>6]&(1<<(value&63)) != 0
}

// ruleEvaluationPlan is derived from a rule's regex and keywords. Widths count
// runes; guided evaluation expands them to byte bounds on non-ASCII values.
type ruleEvaluationPlan struct {
	strategy ruleStrategy
	// unicodeGuided and unicodeFoldGuided prove the selected strategy over
	// every Unicode branch, not just its ASCII projection. The latter requires
	// that the value contain no non-ASCII aliases of ASCII simple-fold classes.
	unicodeGuided     bool
	unicodeFoldGuided bool
	// keywordRequired is true only when every regex match contains at least one configured
	// keyword. Rules without that proof bypass candidate selection and scan every value.
	keywordRequired bool
	// maxWidth bounds the rune count of a match, or is -1 when unbounded.
	maxWidth int
	// minWidth is a lower bound on the number of runes every match consumes. Since every
	// rune occupies at least one byte, a shorter byte string cannot match.
	minWidth int
	// prefixMax is the largest number of prefixClass bytes that may precede the keyword-led
	// remainder of an anchored match.
	prefixMax   int
	prefixClass asciiSet
	// leadingBoundary records that the keyword-led remainder starts with \b, which lets the
	// evaluator reject keyword occurrences glued to a word character without running the regex.
	leadingBoundary bool
	// factors are folded literals every match contains that no keyword occurrence guarantees,
	// longest first. The window strategy skips windows lacking one of them without running
	// the regex, which matters when a keyword is ordinary text (lob's "test_" and "live_")
	// while the regex also requires a rarer literal ("lob").
	factors []string
}

// effectiveKeywords returns the configured keywords when the regex proves one is required.
// Otherwise it derives the longest ASCII literal of at least three bytes that every ASCII
// match contains. Such a literal is a necessary match condition by requiredLiterals'
// construction; if no suitable literal exists, the unproven configured keywords are retained
// and analyzeRule keeps the rule on the keywordless path.
func effectiveKeywords(spec catalogRuleSpec) []string {
	re, err := syntax.Parse(spec.Regex, syntax.Perl)
	if err != nil {
		return spec.Keywords
	}
	return effectiveKeywordsForRegexp(re, spec.Keywords)
}

func effectiveKeywordsForRegexp(re *syntax.Regexp, keywords []string) []string {
	folded := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		if keyword == "" || !isASCIIKeyword(keyword) {
			return keywords
		}
		folded = append(folded, strings.ToLower(keyword))
	}
	if len(folded) > 0 && consumesKeyword(re, folded) {
		return keywords
	}

	literals := requiredLiterals(re)
	slices.SortStableFunc(literals, func(left, right string) int { return len(right) - len(left) })
	for _, literal := range literals {
		if len(literal) >= 3 && isASCIIKeyword(literal) {
			return []string{literal}
		}
	}
	return keywords
}

// exactASCIIStart ignores assertions but never folds or widens characters.
// Every match starts with the returned literal, including on Unicode input.
func exactASCIIStart(re *syntax.Regexp) string {
	switch re.Op {
	case syntax.OpCapture:
		return exactASCIIStart(re.Sub[0])
	case syntax.OpConcat:
		for _, sub := range re.Sub {
			if isZeroWidth(sub) {
				continue
			}
			return exactASCIIStart(sub)
		}
	case syntax.OpLiteral:
		literal := string(re.Rune)
		if re.Flags&syntax.FoldCase != 0 {
			return ""
		}
		if isASCIIKeyword(literal) {
			return literal
		}
	}
	return ""
}

func analyzeRule(re *syntax.Regexp, keywords []string) ruleEvaluationPlan {
	plan := ruleEvaluationPlan{maxWidth: -1}
	plan.minWidth = minRuneWidth(re)
	asciiOrFoldOnly := regexpASCIIOrFoldOnly(re)
	if len(keywords) == 0 {
		return plan
	}
	folded := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		if keyword == "" || !isASCIIKeyword(keyword) {
			return plan
		}
		folded = append(folded, strings.ToLower(keyword))
	}
	plan.maxWidth = asciiMaxWidth(re)
	literals := requiredLiterals(re)
	plan.keywordRequired = consumesKeyword(re, folded) || slices.ContainsFunc(folded, func(keyword string) bool {
		return impliedByFactors(keyword, literals)
	})
	class, prefixMax, rest, prefixProof := splitOptionalPrefix(re)
	switch {
	case beginsWithKeyword(rest, folded):
		plan.strategy = strategyAnchored
		plan.prefixClass = class
		plan.prefixMax = prefixMax
		plan.leadingBoundary = requiresLeadingWordBoundary(rest)
		plan.unicodeGuided = prefixProof == unicodeKeywordProof && unicodeKeywordProof.beginsWithKeyword(rest, folded)
		plan.unicodeFoldGuided = !plan.unicodeGuided &&
			(asciiOrFoldOnly || prefixProof != asciiKeywordProof && unicodeNoAliasKeywordProof.beginsWithKeyword(rest, folded))
	case plan.maxWidth >= 0 && plan.keywordRequired:
		plan.strategy = strategyWindow
		plan.factors = requiredFactors(re, folded)
		plan.unicodeGuided = unicodeKeywordProof.consumesKeyword(re, folded)
		plan.unicodeFoldGuided = !plan.unicodeGuided &&
			(asciiOrFoldOnly || unicodeNoAliasKeywordProof.consumesKeyword(re, folded))
	}
	return plan
}

func regexpASCIIOrFoldOnly(re *syntax.Regexp) bool {
	allowed := func(r rune) bool {
		if r < 128 {
			return true
		}
		for _, alias := range unicodeASCIIFolds {
			if alias.from == r {
				return true
			}
		}
		return false
	}
	switch re.Op {
	case syntax.OpAnyChar, syntax.OpAnyCharNotNL:
		return false
	case syntax.OpLiteral:
		for _, r := range re.Rune {
			if !allowed(r) {
				return false
			}
		}
	case syntax.OpCharClass:
		for i := 0; i+1 < len(re.Rune); i += 2 {
			lo, hi := max(re.Rune[i], 128), re.Rune[i+1]
			if hi-lo+1 > rune(len(unicodeASCIIFolds)) {
				return false
			}
			for r := lo; r <= hi; r++ {
				if !allowed(r) {
					return false
				}
			}
		}
	}
	for _, child := range re.Sub {
		if !regexpASCIIOrFoldOnly(child) {
			return false
		}
	}
	return true
}

// requiresLeftContext reports whether an assertion can inspect the rune before
// the match starts. Once a rune is necessarily consumed, later assertions see
// that rune even when the anchored regexp runs on the suffix alone.
func requiresLeftContext(re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpBeginLine, syntax.OpBeginText, syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return true
	case syntax.OpConcat:
		for _, sub := range re.Sub {
			if requiresLeftContext(sub) {
				return true
			}
			if minRuneWidth(sub) > 0 {
				return false
			}
		}
	case syntax.OpRepeat:
		return re.Max != 0 && requiresLeftContext(re.Sub[0])
	default:
		for _, sub := range re.Sub {
			if requiresLeftContext(sub) {
				return true
			}
		}
	}
	return false
}

// literalRunMatcher covers only one captured ASCII literal and greedy class run.
// Its trailing delimiter is disjoint from the body (or a word boundary after a
// word-only body), so a failed tail can never succeed by shortening the run.
// The original regexp remains the unanchored reference and fallback.
type literalRunMatcher struct {
	prefix          string
	body            asciiSet
	min             int
	max             int
	leadingBoundary bool
	trailing        syntax.Op
	delimiter       syntax.Inst
}

func newLiteralRunMatcher(re *syntax.Regexp) *literalRunMatcher {
	if re.Op != syntax.OpConcat || len(re.Sub) == 0 {
		return nil
	}
	parts := re.Sub
	var matcher literalRunMatcher
	if parts[0].Op == syntax.OpWordBoundary {
		matcher.leadingBoundary = true
		parts = parts[1:]
	}
	if len(parts) < 1 || len(parts) > 2 || parts[0].Op != syntax.OpCapture || parts[0].Cap != 1 {
		return nil
	}
	capture := parts[0].Sub[0]
	if capture.Op != syntax.OpConcat || len(capture.Sub) != 2 || capture.Sub[0].Op != syntax.OpLiteral {
		return nil
	}
	matcher.prefix = exactASCIIStart(capture.Sub[0])
	if matcher.prefix == "" {
		return nil
	}
	repeat := capture.Sub[1]
	if repeat.Flags&syntax.NonGreedy != 0 {
		return nil
	}
	switch repeat.Op {
	case syntax.OpRepeat:
		matcher.min, matcher.max = repeat.Min, repeat.Max
	case syntax.OpPlus:
		matcher.min, matcher.max = 1, -1
	default:
		return nil
	}
	if matcher.min < 1 || repeat.Sub[0].Op != syntax.OpCharClass {
		return nil
	}
	body := syntax.Inst{Op: syntax.InstRune, Rune: repeat.Sub[0].Rune, Arg: uint32(repeat.Sub[0].Flags & syntax.FoldCase)}
	if instructionMatchesNonASCII(&body) {
		return nil
	}
	matcher.body = instructionASCIISet(&body)
	matcher.trailing = syntax.OpEmptyMatch
	if len(parts) == 2 {
		tail := parts[1]
		matcher.trailing = tail.Op
		switch tail.Op {
		case syntax.OpWordBoundary, syntax.OpEndText:
		case syntax.OpAlternate:
			if len(tail.Sub) != 2 {
				return nil
			}
			left, right := tail.Sub[0], tail.Sub[1]
			if right.Op == syntax.OpEndText {
				left, right = right, left
			}
			if left.Op != syntax.OpEndText || right.Op != syntax.OpCharClass {
				return nil
			}
			matcher.delimiter = syntax.Inst{Op: syntax.InstRune, Rune: right.Rune, Arg: uint32(right.Flags & syntax.FoldCase)}
			delimiter := instructionASCIISet(&matcher.delimiter)
			if matcher.body[0]&delimiter[0] != 0 || matcher.body[1]&delimiter[1] != 0 {
				return nil
			}
		default:
			return nil
		}
	}
	if matcher.trailing == syntax.OpWordBoundary {
		for value := range 128 {
			if matcher.body.has(byte(value)) && !isASCIIWordByte(byte(value)) {
				return nil
			}
		}
	}
	return &matcher
}

// requiredFactors returns the folded literals of at least analysisMinFactorLength bytes that
// every ASCII match of the expression contains, longest first, dropping those a longer factor
// or every keyword already guarantees.
func requiredFactors(re *syntax.Regexp, keywords []string) []string {
	literals := requiredLiterals(re)
	slices.SortStableFunc(literals, func(left, right string) int { return len(right) - len(left) })
	var factors []string
	for _, literal := range literals {
		if len(literal) < analysisMinFactorLength || impliedByKeywords(literal, keywords) || impliedByFactors(literal, factors) {
			continue
		}
		factors = append(factors, literal)
	}
	if len(factors) > analysisMaxFactors {
		factors = factors[:analysisMaxFactors]
	}
	return factors
}

type literalAlternative struct {
	text string
	fold bool
}

// requiredLiteralAlternatives proves a bounded disjunction: every match contains
// at least one returned literal. Unlike requiredLiterals, branches need not share
// a common substring. An unprovable branch disables the entire rejection gate.
func requiredLiteralAlternatives(re *syntax.Regexp) []literalAlternative {
	return literalAlternativesForRegexp(re, false)
}

// requiredExactLiteralAlternatives retains case-sensitive ASCII factors, including
// finite prefixes spanning adjacent atoms. Their bytes remain necessary on Unicode
// and malformed input. A branch without an exact proof disables this extra gate;
// callers retain the original factor gate so this cannot weaken its selectivity.
func requiredExactLiteralAlternatives(re *syntax.Regexp) []literalAlternative {
	return literalAlternativesForRegexp(re, true)
}

func literalAlternativesForRegexp(re *syntax.Regexp, exactOnly bool) []literalAlternative {
	switch re.Op {
	case syntax.OpLiteral:
		if len(re.Rune) < analysisMinFactorLength || exactOnly && re.Flags&syntax.FoldCase != 0 {
			return nil
		}
		for _, r := range re.Rune {
			if r >= 128 {
				return nil
			}
		}
		text := string(re.Rune[:min(len(re.Rune), analysisMaxStringLength)])
		fold := re.Flags&syntax.FoldCase != 0
		if fold {
			text = strings.ToLower(text)
		}
		return []literalAlternative{{text: text, fold: fold}}
	case syntax.OpCapture, syntax.OpPlus:
		return literalAlternativesForRegexp(re.Sub[0], exactOnly)
	case syntax.OpRepeat:
		if re.Min > 0 {
			return literalAlternativesForRegexp(re.Sub[0], exactOnly)
		}
	case syntax.OpConcat:
		var best []literalAlternative
		bestWidth := 0
		for _, sub := range re.Sub {
			alternatives := literalAlternativesForRegexp(sub, exactOnly)
			width := analysisMaxStringLength
			for _, literal := range alternatives {
				width = min(width, len(literal.text))
			}
			if len(alternatives) != 0 && (width > bestWidth || width == bestWidth && len(alternatives) < len(best)) {
				best, bestWidth = alternatives, width
			}
		}
		if exactOnly {
			for i := range re.Sub {
				// Starting before a zero-width atom repeats the same byte prefix
				// as starting after it; long assertion chains add no evidence.
				if isZeroWidth(re.Sub[i]) {
					continue
				}
				prefixes, _ := exactLiteralPrefixes(&syntax.Regexp{Op: syntax.OpConcat, Sub: re.Sub[i:]})
				width := analysisMaxStringLength
				for _, prefix := range prefixes {
					width = min(width, len(prefix))
				}
				if len(prefixes) != 0 && width >= analysisMinFactorLength &&
					(width > bestWidth || width == bestWidth && len(prefixes) < len(best)) {
					best = make([]literalAlternative, len(prefixes))
					for j, prefix := range prefixes {
						best[j] = literalAlternative{text: prefix}
					}
					bestWidth = width
				}
			}
		}
		return best
	case syntax.OpAlternate:
		var alternatives []literalAlternative
		for _, sub := range re.Sub {
			required := literalAlternativesForRegexp(sub, exactOnly)
			if len(required) == 0 {
				return nil
			}
			for _, literal := range required {
				if !slices.Contains(alternatives, literal) {
					if len(alternatives) == analysisMaxStrings {
						return nil
					}
					alternatives = append(alternatives, literal)
				}
			}
		}
		return alternatives
	}
	return nil
}

// exactLiteralPrefixes returns a bounded set covering the prefixes of every match,
// and whether it enumerates complete strings. Assertions only constrain the language
// and therefore contribute no bytes. Unknown or folded atoms contribute the universal
// prefix "", never a discarded branch. A variable repetition contributes its required
// first copies but stops concatenation, so e.g. a+[bc] cannot invent an adjacent "ab".
func exactLiteralPrefixes(re *syntax.Regexp) ([]string, bool) {
	switch re.Op {
	case syntax.OpNoMatch:
		return nil, true
	case syntax.OpEmptyMatch, syntax.OpBeginLine, syntax.OpEndLine, syntax.OpBeginText,
		syntax.OpEndText, syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return []string{""}, true
	case syntax.OpLiteral:
		if re.Flags&syntax.FoldCase != 0 {
			return []string{""}, false
		}
		length := min(len(re.Rune), analysisMaxStringLength)
		for _, r := range re.Rune[:length] {
			if r < 0 || r >= 128 {
				return []string{""}, false
			}
		}
		return []string{string(re.Rune[:length])}, length == len(re.Rune)
	case syntax.OpCharClass:
		if re.Flags&syntax.FoldCase != 0 {
			return []string{""}, false
		}
		// Expanding a broad alphabet only multiplies searches for nearly the
		// same prefix. Small classes can add useful framing without that cost.
		var out []string
		for i := 0; i+1 < len(re.Rune); i += 2 {
			low, high := re.Rune[i], re.Rune[i+1]
			if low < 0 || high >= 128 || int(high-low+1) > analysisMaxFactors-len(out) {
				return []string{""}, false
			}
			for r := low; r <= high; r++ {
				out = append(out, string(r))
			}
		}
		return out, true
	case syntax.OpCapture:
		return exactLiteralPrefixes(re.Sub[0])
	case syntax.OpQuest:
		prefixes, complete := exactLiteralPrefixes(re.Sub[0])
		joined, ok := unionStrings([]string{""}, prefixes)
		if !ok {
			return []string{""}, false
		}
		return joined, complete
	case syntax.OpPlus:
		prefixes, _ := exactLiteralPrefixes(re.Sub[0])
		return prefixes, false
	case syntax.OpRepeat:
		if re.Max == 0 {
			return []string{""}, true
		}
		if re.Min == 0 {
			return []string{""}, false
		}
		prefixes, complete := exactLiteralPrefixes(re.Sub[0])
		if !complete || len(prefixes) == 0 || len(prefixes) == 1 && prefixes[0] == "" {
			return prefixes, complete
		}
		acc := []string{""}
		for range re.Min {
			joined, ok := crossStrings(acc, prefixes)
			if !ok {
				return acc, false
			}
			acc = joined
		}
		return acc, re.Max == re.Min
	case syntax.OpConcat:
		acc := []string{""}
		for _, sub := range re.Sub {
			prefixes, complete := exactLiteralPrefixes(sub)
			joined, ok := crossStrings(acc, prefixes)
			if !ok {
				return acc, false
			}
			acc = joined
			if !complete {
				return acc, false
			}
		}
		return acc, true
	case syntax.OpAlternate:
		var acc []string
		complete := true
		for _, sub := range re.Sub {
			prefixes, branchComplete := exactLiteralPrefixes(sub)
			joined, ok := unionStrings(acc, prefixes)
			if !ok {
				return []string{""}, false
			}
			acc = joined
			complete = complete && branchComplete
		}
		return acc, complete
	}
	return []string{""}, false
}

// requiredLiterals returns folded strings every ASCII match of the expression contains. For an
// alternation these are the substrings shared by literals every branch requires, which follows
// the parser's prefix factoring ("new(?:-relic|relic|_relic)" yields "new" and "relic").
func requiredLiterals(re *syntax.Regexp) []string {
	switch re.Op {
	case syntax.OpLiteral:
		var builder strings.Builder
		for _, r := range re.Rune {
			if r >= 0x80 {
				return nil
			}
			builder.WriteByte(foldASCII(byte(r)))
		}
		return []string{builder.String()}
	case syntax.OpCapture, syntax.OpPlus:
		return requiredLiterals(re.Sub[0])
	case syntax.OpRepeat:
		if re.Min > 0 {
			return requiredLiterals(re.Sub[0])
		}
	case syntax.OpConcat:
		var out []string
		for _, sub := range re.Sub {
			for _, literal := range requiredLiterals(sub) {
				out = appendUniqueString(out, literal)
			}
		}
		return out
	case syntax.OpAlternate:
		common := substringsOf(requiredLiterals(re.Sub[0]))
		for _, sub := range re.Sub[1:] {
			branch := requiredLiterals(sub)
			common = slices.DeleteFunc(common, func(candidate string) bool {
				return !slices.ContainsFunc(branch, func(literal string) bool { return strings.Contains(literal, candidate) })
			})
		}
		return common
	}
	return nil
}

// substringsOf returns a bounded subset of the distinct substrings of at least
// analysisMinFactorLength bytes. Dropping proof candidates only weakens a
// rejection gate; it must not turn a large tenant literal into quadratic storage.
func substringsOf(literals []string) []string {
	var out []string
	for _, literal := range literals {
		for start := 0; start < len(literal); start++ {
			for end := start + analysisMinFactorLength; end <= len(literal); end++ {
				out = appendUniqueString(out, literal[start:end])
				if len(out) == analysisMaxStrings {
					return out
				}
			}
		}
	}
	return out
}

func impliedByKeywords(literal string, keywords []string) bool {
	for _, keyword := range keywords {
		if !strings.Contains(keyword, literal) {
			return false
		}
	}
	return true
}

func impliedByFactors(literal string, factors []string) bool {
	return slices.ContainsFunc(factors, func(factor string) bool { return strings.Contains(factor, literal) })
}

// minRuneWidth returns a lower bound on the number of runes every match consumes. It takes
// the shortest branch and the minimum repetition count, so it cannot exceed any actual
// match's rune count; because every rune uses at least one byte, a value with fewer bytes
// cannot match.
func minRuneWidth(re *syntax.Regexp) int {
	switch re.Op {
	case syntax.OpNoMatch, syntax.OpEmptyMatch, syntax.OpBeginLine, syntax.OpEndLine,
		syntax.OpBeginText, syntax.OpEndText, syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return 0
	case syntax.OpLiteral:
		return len(re.Rune)
	case syntax.OpCharClass, syntax.OpAnyCharNotNL, syntax.OpAnyChar:
		return 1
	case syntax.OpCapture, syntax.OpPlus:
		return minRuneWidth(re.Sub[0])
	case syntax.OpQuest, syntax.OpStar:
		return 0
	case syntax.OpRepeat:
		return re.Min * minRuneWidth(re.Sub[0])
	case syntax.OpConcat:
		total := 0
		for _, sub := range re.Sub {
			total += minRuneWidth(sub)
		}
		return total
	case syntax.OpAlternate:
		narrowest := minRuneWidth(re.Sub[0])
		for _, sub := range re.Sub[1:] {
			narrowest = min(narrowest, minRuneWidth(sub))
		}
		return narrowest
	}
	return 0
}

// regexpInstructionEstimate bounds the expanded Thompson program without
// simplifying counted repetitions. Captures add two instructions, optional
// copies and loops add branches, and nullable stars may require two branches.
// Saturation keeps nested arithmetic safe and preserves rejection at the cap.
func regexpInstructionEstimate(re *syntax.Regexp) int {
	switch re.Op {
	case syntax.OpLiteral:
		return max(1, len(re.Rune))
	case syntax.OpCapture:
		return addRegexpInstructions(regexpInstructionEstimate(re.Sub[0]), 2)
	case syntax.OpStar:
		return addRegexpInstructions(regexpInstructionEstimate(re.Sub[0]), 2)
	case syntax.OpPlus, syntax.OpQuest:
		return addRegexpInstructions(regexpInstructionEstimate(re.Sub[0]), 1)
	case syntax.OpRepeat:
		if re.Max == 0 {
			return 1
		}
		body := regexpInstructionEstimate(re.Sub[0])
		if re.Max < 0 {
			if re.Min == 0 {
				return addRegexpInstructions(body, 2)
			}
			return addRegexpInstructions(multiplyRegexpInstructions(body, re.Min), 1)
		}
		return addRegexpInstructions(multiplyRegexpInstructions(body, re.Max), re.Max-re.Min)
	case syntax.OpConcat, syntax.OpAlternate:
		total := 0
		if re.Op == syntax.OpAlternate {
			total = max(0, len(re.Sub)-1)
		}
		for _, sub := range re.Sub {
			total = addRegexpInstructions(total, regexpInstructionEstimate(sub))
		}
		return max(1, total)
	default:
		// Empty matches, assertions, no-match, and rune classes each need
		// at most one instruction.
		return 1
	}
}

func addRegexpInstructions(left, right int) int {
	if left > maxCustomPolicyInstructions || right > maxCustomPolicyInstructions-left {
		return maxCustomPolicyInstructions + 1
	}
	return left + right
}

func multiplyRegexpInstructions(instructions, copies int) int {
	if copies == 0 {
		return 0
	}
	if instructions > maxCustomPolicyInstructions/copies {
		return maxCustomPolicyInstructions + 1
	}
	return instructions * copies
}

// asciiMaxWidth counts one unit per consumed rune, or returns -1 when unbounded.
// It is a byte bound on ASCII input; arbitrary UTF-8 needs at most UTFMax bytes per unit.
func asciiMaxWidth(re *syntax.Regexp) int {
	switch re.Op {
	case syntax.OpNoMatch, syntax.OpEmptyMatch, syntax.OpBeginLine, syntax.OpEndLine,
		syntax.OpBeginText, syntax.OpEndText, syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return 0
	case syntax.OpLiteral:
		return len(re.Rune)
	case syntax.OpCharClass, syntax.OpAnyCharNotNL, syntax.OpAnyChar:
		return 1
	case syntax.OpCapture, syntax.OpQuest:
		return asciiMaxWidth(re.Sub[0])
	case syntax.OpRepeat:
		if re.Max < 0 {
			return -1
		}
		width := asciiMaxWidth(re.Sub[0])
		if width < 0 {
			return -1
		}
		return re.Max * width
	case syntax.OpConcat:
		total := 0
		for _, sub := range re.Sub {
			width := asciiMaxWidth(sub)
			if width < 0 {
				return -1
			}
			total += width
		}
		return total
	case syntax.OpAlternate:
		widest := 0
		for _, sub := range re.Sub {
			width := asciiMaxWidth(sub)
			if width < 0 {
				return -1
			}
			widest = max(widest, width)
		}
		return widest
	}
	return -1
}

// splitOptionalPrefix peels optional repeats of one projected ASCII class off a
// concatenation. The extra proof records when walking that class backwards by
// bytes is also sound on Unicode input; arbitrary Unicode members disable it.
func splitOptionalPrefix(re *syntax.Regexp) (asciiSet, int, *syntax.Regexp, keywordProof) {
	if re.Op != syntax.OpConcat {
		return asciiSet{}, 0, re, unicodeKeywordProof
	}
	var class asciiSet
	prefixMax := 0
	count := 0
	proof := unicodeKeywordProof
	for count < len(re.Sub)-1 {
		sub := re.Sub[count]
		if sub.Op != syntax.OpRepeat || sub.Min != 0 || sub.Max <= 0 || sub.Sub[0].Op != syntax.OpCharClass {
			break
		}
		set := asciiClassSet(sub.Sub[0])
		if count > 0 && set != class {
			break
		}
		if proof != asciiKeywordProof && !regexpASCIIOrFoldOnly(sub) {
			proof = asciiKeywordProof
		} else if proof == unicodeKeywordProof && len(sub.Sub[0].Rune) > 0 && sub.Sub[0].Rune[len(sub.Sub[0].Rune)-1] >= 128 {
			proof = unicodeNoAliasKeywordProof
		}
		class = set
		prefixMax += sub.Max
		count++
	}
	switch {
	case count == 0:
		return asciiSet{}, 0, re, unicodeKeywordProof
	case count == len(re.Sub)-1:
		return class, prefixMax, re.Sub[count], proof
	default:
		return class, prefixMax, &syntax.Regexp{Op: syntax.OpConcat, Sub: re.Sub[count:]}, proof
	}
}

func asciiClassSet(class *syntax.Regexp) asciiSet {
	var set asciiSet
	for i := 0; i+1 < len(class.Rune); i += 2 {
		for r := class.Rune[i]; r <= class.Rune[i+1] && r < 0x80; r++ {
			set.add(byte(r))
		}
	}
	return set
}

// keywordProof selects the input language whose keyword coverage is being
// proved. Unicode proofs retain a non-ASCII marker instead of dropping branches:
// no ASCII keyword can cross that marker. A folded ASCII rune with a Unicode
// alias is also masked unless the input excludes every such alias.
type keywordProof uint8

const (
	asciiKeywordProof keywordProof = iota
	unicodeKeywordProof
	unicodeNoAliasKeywordProof
)

// beginsWithKeyword is the ASCII-only proof used to select an execution strategy.
func beginsWithKeyword(re *syntax.Regexp, keywords []string) bool {
	return asciiKeywordProof.beginsWithKeyword(re, keywords)
}

func (proof keywordProof) beginsWithKeyword(re *syntax.Regexp, keywords []string) bool {
	strs, _ := proof.leadingStrings(re)
	return allHaveKeywordPrefix(strs, keywords)
}

// consumesKeyword reports whether every ASCII match of the expression contains one of the
// folded keywords.
func consumesKeyword(re *syntax.Regexp, keywords []string) bool {
	return asciiKeywordProof.consumesKeyword(re, keywords)
}

func (proof keywordProof) consumesKeyword(re *syntax.Regexp, keywords []string) bool {
	if strs, ok := proof.strings(re); ok {
		return allContainKeyword(strs, keywords)
	}
	switch re.Op {
	case syntax.OpCapture, syntax.OpPlus:
		return proof.consumesKeyword(re.Sub[0], keywords)
	case syntax.OpRepeat:
		return re.Min > 0 && proof.consumesKeyword(re.Sub[0], keywords)
	case syntax.OpAlternate:
		for _, sub := range re.Sub {
			if !proof.consumesKeyword(sub, keywords) {
				return false
			}
		}
		return true
	case syntax.OpConcat:
		for i, sub := range re.Sub {
			if proof.consumesKeyword(sub, keywords) {
				return true
			}
			// The keyword may be spelled across children, e.g. "xox[bp]-" for xoxb and xoxp,
			// or "fo(?:o|b)" after the parser factors "foo|fob".
			strs, _ := proof.leadingStrings(&syntax.Regexp{Op: syntax.OpConcat, Sub: re.Sub[i:]})
			if allContainKeyword(strs, keywords) {
				return true
			}
		}
	}
	return false
}

// leadingStrings returns folded prefixes covering every match in the proof's
// language, and whether they cover complete strings. Unknown structure yields
// the universal prefix "", never a discarded branch. Enumeration caps only
// weaken the proof.
func (proof keywordProof) leadingStrings(re *syntax.Regexp) ([]string, bool) {
	if strs, ok := proof.strings(re); ok {
		return strs, true
	}
	switch re.Op {
	case syntax.OpCapture:
		return proof.leadingStrings(re.Sub[0])
	case syntax.OpConcat:
		acc := []string{""}
		for _, sub := range re.Sub {
			strs, complete := proof.leadingStrings(sub)
			joined, ok := crossStrings(acc, strs)
			if !ok {
				return acc, false
			}
			acc = joined
			if !complete {
				return acc, false
			}
		}
		return acc, true
	case syntax.OpAlternate:
		var acc []string
		for _, sub := range re.Sub {
			strs, _ := proof.leadingStrings(sub)
			joined, ok := unionStrings(acc, strs)
			if !ok {
				return []string{""}, false
			}
			acc = joined
		}
		return acc, false
	}
	// Repeats that were not enumerable above would only contribute unhelpful single
	// characters, so they end the prefix.
	return []string{""}, false
}

func requiresLeadingWordBoundary(re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpWordBoundary:
		return true
	case syntax.OpCapture:
		return requiresLeadingWordBoundary(re.Sub[0])
	case syntax.OpConcat:
		for _, sub := range re.Sub {
			if requiresLeadingWordBoundary(sub) {
				return true
			}
			if !isZeroWidth(sub) {
				return false
			}
		}
	case syntax.OpAlternate:
		for _, sub := range re.Sub {
			if !requiresLeadingWordBoundary(sub) {
				return false
			}
		}
		return len(re.Sub) > 0
	}
	return false
}

func isZeroWidth(re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpEmptyMatch, syntax.OpBeginLine, syntax.OpEndLine, syntax.OpBeginText,
		syntax.OpEndText, syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return true
	case syntax.OpCapture:
		return isZeroWidth(re.Sub[0])
	}
	return false
}

// strings enumerates a bounded, folded language for keyword proofs. Assertions
// can only remove matches, so they contribute the empty string. Unicode markers
// consume a position but can never establish an ASCII keyword. A nil, true
// result is reserved for an actually empty language under the selected input
// assumptions; arbitrary Unicode alternatives must not disappear.
func (proof keywordProof) strings(re *syntax.Regexp) ([]string, bool) {
	switch re.Op {
	case syntax.OpNoMatch:
		return nil, true
	case syntax.OpEmptyMatch, syntax.OpBeginLine, syntax.OpEndLine, syntax.OpBeginText,
		syntax.OpEndText, syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return []string{""}, true
	case syntax.OpLiteral:
		var builder strings.Builder
		for _, r := range re.Rune {
			if r >= 0x80 {
				if proof == asciiKeywordProof {
					return nil, true
				}
				builder.WriteRune(utf8.RuneError)
				continue
			}
			if proof == unicodeKeywordProof && re.Flags&syntax.FoldCase != 0 &&
				slices.ContainsFunc(unicodeASCIIFolds, func(alias unicodeASCIIFold) bool { return alias.to == foldASCII(byte(r)) }) {
				builder.WriteRune(utf8.RuneError)
				continue
			}
			builder.WriteByte(foldASCII(byte(r)))
		}
		return []string{builder.String()}, true
	case syntax.OpCharClass:
		var out []string
		for i := 0; i+1 < len(re.Rune); i += 2 {
			for r := re.Rune[i]; r <= re.Rune[i+1] && r < 0x80; r++ {
				out = appendUniqueString(out, string(rune(foldASCII(byte(r)))))
			}
		}
		if proof != asciiKeywordProof && len(re.Rune) > 0 && re.Rune[len(re.Rune)-1] >= 128 &&
			(proof != unicodeNoAliasKeywordProof || !regexpASCIIOrFoldOnly(re)) {
			out = append(out, string(utf8.RuneError))
		}
		if len(out) > analysisMaxStrings {
			return nil, false
		}
		return out, true
	case syntax.OpCapture:
		return proof.strings(re.Sub[0])
	case syntax.OpQuest:
		sub, ok := proof.strings(re.Sub[0])
		if !ok {
			return nil, false
		}
		return unionStrings([]string{""}, sub)
	case syntax.OpRepeat:
		if re.Max < 0 {
			return nil, false
		}
		sub, ok := proof.strings(re.Sub[0])
		if !ok {
			return nil, false
		}
		var out []string
		power := []string{""}
		for count := 0; ; count++ {
			if count >= re.Min {
				if out, ok = unionStrings(out, power); !ok {
					return nil, false
				}
			}
			if count == re.Max {
				return out, true
			}
			if power, ok = crossStrings(power, sub); !ok {
				return nil, false
			}
		}
	case syntax.OpConcat:
		out := []string{""}
		for _, sub := range re.Sub {
			strs, ok := proof.strings(sub)
			if !ok {
				return nil, false
			}
			if out, ok = crossStrings(out, strs); !ok {
				return nil, false
			}
		}
		return out, true
	case syntax.OpAlternate:
		var out []string
		for _, sub := range re.Sub {
			strs, ok := proof.strings(sub)
			if !ok {
				return nil, false
			}
			if out, ok = unionStrings(out, strs); !ok {
				return nil, false
			}
		}
		return out, true
	}
	return nil, false
}

func crossStrings(left, right []string) ([]string, bool) {
	if len(left) == 0 || len(right) == 0 {
		return nil, true
	}
	if len(left)*len(right) > analysisMaxStrings {
		return nil, false
	}
	out := make([]string, 0, len(left)*len(right))
	for _, prefix := range left {
		for _, suffix := range right {
			if len(prefix)+len(suffix) > analysisMaxStringLength {
				return nil, false
			}
			out = append(out, prefix+suffix)
		}
	}
	return out, true
}

func unionStrings(left, right []string) ([]string, bool) {
	out := append(make([]string, 0, len(left)+len(right)), left...)
	for _, value := range right {
		out = appendUniqueString(out, value)
	}
	if len(out) > analysisMaxStrings {
		return nil, false
	}
	return out, true
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func allContainKeyword(values, keywords []string) bool {
	for _, value := range values {
		if !containsAnyKeyword(value, keywords) {
			return false
		}
	}
	return true
}

func containsAnyKeyword(value string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(value, keyword) {
			return true
		}
	}
	return false
}

func allHaveKeywordPrefix(values, keywords []string) bool {
	for _, value := range values {
		if !hasAnyKeywordPrefix(value, keywords) {
			return false
		}
	}
	return true
}

func hasAnyKeywordPrefix(value string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.HasPrefix(value, keyword) {
			return true
		}
	}
	return false
}
