package secrets

import (
	"fmt"
	"strings"
	"unicode"
)

// foldKeyword uses Go regexp's simple-fold equivalence classes. Lowercasing
// alone leaves aliases such as long-s distinct from ASCII s and can incorrectly
// discard a rule before its case-insensitive regexp sees the original value.
// ASCII hot-path scanning does not call this; it is used for construction and
// the Unicode fallback, where byte positions are deliberately unavailable.
func foldKeyword(value string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x80 {
			return rune(foldASCII(byte(r)))
		}
		smallest := r
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			smallest = min(smallest, next)
		}
		return unicode.ToLower(smallest)
	}, value)
}

type unicodeASCIIFold struct {
	from    rune
	to      byte
	encoded string
}

// Only Unicode members of an ASCII simple-fold class can participate in an
// ASCII keyword. Derive those aliases from Go's tables instead of special-casing
// benchmark characters or performing full Unicode lowercasing on every rune.
var unicodeASCIIFolds = func() []unicodeASCIIFold {
	var aliases []unicodeASCIIFold
	for b := range 128 {
		if foldASCII(byte(b)) != byte(b) {
			continue
		}
		r := rune(b)
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			if next >= 128 {
				aliases = append(aliases, unicodeASCIIFold{from: next, to: byte(b), encoded: string(next)})
			}
		}
	}
	return aliases
}()

func foldASCIIKeywordInput(value string) string {
	return strings.Map(func(r rune) rune {
		if r < 128 {
			return rune(foldASCII(byte(r)))
		}
		for _, alias := range unicodeASCIIFolds {
			if r == alias.from {
				return rune(alias.to)
			}
		}
		return r
	}, value)
}

func containsUnicodeASCIIFold(value string) bool {
	for _, alias := range unicodeASCIIFolds {
		if strings.Contains(value, alias.encoded) {
			return true
		}
	}
	return false
}

const catalogRuleWords = 17 // 1088 bounded slots for the Betterleaks-inclusive catalog and per-policy rule sets.

type ruleSet [catalogRuleWords]uint64

func (s *ruleSet) add(index uint16) {
	s[index>>6] |= uint64(1) << (index & 63)
}

func (s ruleSet) has(index uint16) bool {
	return s[index>>6]&(uint64(1)<<(index&63)) != 0
}

func (s ruleSet) intersect(other ruleSet) ruleSet {
	for i := range s {
		s[i] &= other[i]
	}
	return s
}

func (s *ruleSet) merge(other ruleSet) {
	for i := range s {
		s[i] |= other[i]
	}
}

type matcherState struct {
	next    [128]uint16
	outputs []uint16
}

type unicodeKeyword struct {
	value string
	rule  uint16
}

type keywordMatcher struct {
	states          []matcherState
	unicodeKeywords []unicodeKeyword
}

type matcherBuilderState struct {
	next    map[byte]uint16
	fail    uint16
	outputs []uint16
}

type matcherKeyword struct {
	value string
	rule  uint16
}

func newKeywordMatcher(keywords []matcherKeyword) (keywordMatcher, error) {
	build := []matcherBuilderState{{next: make(map[byte]uint16)}}
	unicodeKeywords := make([]unicodeKeyword, 0)

	for _, keyword := range keywords {
		value := foldKeyword(keyword.value)
		if value == "" {
			continue
		}
		if !isASCIIKeyword(value) {
			unicodeKeywords = append(unicodeKeywords, unicodeKeyword{value: value, rule: keyword.rule})
			continue
		}
		state := uint16(0)
		for i := range len(value) {
			b := value[i]
			next, ok := build[state].next[b]
			if !ok {
				if len(build) >= 1<<16 {
					return keywordMatcher{}, fmt.Errorf("secret keyword automaton exceeds %d states", 1<<16-1)
				}
				next = uint16(len(build))
				build[state].next[b] = next
				build = append(build, matcherBuilderState{next: make(map[byte]uint16)})
			}
			state = next
		}
		build[state].outputs = appendUniqueOutput(build[state].outputs, keyword.rule)
	}

	queue := make([]uint16, 0, len(build)-1)
	for _, child := range build[0].next {
		queue = append(queue, child)
	}
	for head := 0; head < len(queue); head++ {
		state := queue[head]
		for b, child := range build[state].next {
			queue = append(queue, child)
			fallback := build[state].fail
			for {
				if target, ok := build[fallback].next[b]; ok && target != child {
					build[child].fail = target
					break
				}
				if fallback == 0 {
					break
				}
				fallback = build[fallback].fail
			}
			for _, output := range build[build[child].fail].outputs {
				build[child].outputs = appendUniqueOutput(build[child].outputs, output)
			}
		}
	}

	states := make([]matcherState, len(build))
	for b, child := range build[0].next {
		states[0].next[b] = child
	}
	states[0].outputs = build[0].outputs
	for _, state := range queue {
		states[state].outputs = build[state].outputs
		fallback := build[state].fail
		for b := range 128 {
			if child, ok := build[state].next[byte(b)]; ok {
				states[state].next[b] = child
			} else {
				states[state].next[b] = states[fallback].next[b]
			}
		}
	}

	return keywordMatcher{states: states, unicodeKeywords: unicodeKeywords}, nil
}

func (m *keywordMatcher) match(value string, candidates *ruleSet) {
	if value == "" || len(m.states) == 0 {
		return
	}
	state := uint16(0)
	for i := range len(value) {
		b := value[i]
		if b >= 0x80 {
			m.matchUnicode(value, candidates)
			return
		}
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		state = m.states[state].next[b]
		for _, output := range m.states[state].outputs {
			candidates.add(output)
		}
	}
}

func (m *keywordMatcher) matchUnicode(value string, candidates *ruleSet) {
	if len(m.unicodeKeywords) == 0 {
		value = foldASCIIKeywordInput(value)
	} else {
		value = foldKeyword(value)
	}
	state := uint16(0)
	for i := range len(value) {
		b := value[i]
		if b >= 0x80 {
			state = 0
			continue
		}
		state = m.states[state].next[b]
		for _, output := range m.states[state].outputs {
			candidates.add(output)
		}
	}
	for _, keyword := range m.unicodeKeywords {
		if strings.Contains(value, keyword.value) {
			candidates.add(keyword.rule)
		}
	}
}

func appendUniqueOutput(outputs []uint16, output uint16) []uint16 {
	for _, existing := range outputs {
		if existing == output {
			return outputs
		}
	}
	return append(outputs, output)
}

func isASCIIKeyword(value string) bool {
	for i := range len(value) {
		if value[i] >= 0x80 {
			return false
		}
	}
	return true
}
