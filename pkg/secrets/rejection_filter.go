package secrets

import (
	"math/bits"
	"regexp/syntax"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	rejectionRelaxSpread = 16
	rejectionNFACap      = 4096
	rejectionDFACap      = 1024
	rejectionNFAWords    = rejectionNFACap / 64
	// Small values cost less to scan directly than to search for a prefix first.
	rejectionPrefixMinBytes = 256
	// Anchored probes must not rescan whole suffixes for every keyword when
	// repeat relaxation leaves the DFA alive. An unfinished probe is inconclusive.
	rejectionPrefixProbeBytes = 256
	// One conservative symbol represents every non-ASCII rune, including RuneError.
	// ASCII bytes retain exact classes; no UTF-8 byte is mistaken for an ASCII byte.
	rejectionUnicodeSymbol = 128
	rejectionAlphabetSize  = 129
)

// rejectionFilter is a DFA for an over-approximation of the regex. ASCII character
// classes are exact; a Unicode transition includes every instruction that can match any
// non-ASCII rune. Only the accepting regexp decides actual findings.
type rejectionFilter struct {
	classOf     [rejectionAlphabetSize]uint8
	start       uint16
	classes     int
	prefix      string   // exact ASCII literal prefix of the original regex, if available
	next        []uint16 // state*classes + class -> state; 0 is dead, 1 is the accepting sink
	acceptAtEnd []bool   // end-of-text assertions are resolved only after input is consumed
}

// mayMatch reports whether an unanchored filter can match any substring of value.
func (f *rejectionFilter) mayMatch(value string) bool {
	return f.run(value, len(value))
}

// mayMatchAfterPrefix skips leading bytes using an exact ASCII literal prefix.
// Every match starts with that prefix, even within otherwise non-ASCII input, so
// none can precede its first occurrence. The suffix retains every possible match
// for the unanchored DFA; the accepting regexp still sees the original whole value.
func (f *rejectionFilter) mayMatchAfterPrefix(value string) bool {
	start := strings.Index(value, f.prefix)
	return start >= 0 && f.run(value[start:], len(value)-start)
}

// mayMatchPrefix rejects only within bounded lookahead. If the DFA remains
// alive at the limit, the exact anchored regex decides the candidate.
func (f *rejectionFilter) mayMatchPrefix(value string) bool {
	return f.run(value, rejectionPrefixProbeBytes)
}

func (f *rejectionFilter) run(value string, limit int) bool {
	state := f.start
	if state == 1 {
		return true
	}
	stop := min(len(value), limit)
	for index := 0; index < stop; {
		if state == 0 {
			return false
		}
		symbol := value[index]
		index++
		if symbol >= utf8.RuneSelf {
			_, width := utf8.DecodeRuneInString(value[index-1:])
			index += width - 1
			symbol = rejectionUnicodeSymbol
		}
		state = f.next[int(state)*f.classes+int(f.classOf[symbol])]
		if state == 1 {
			return true
		}
	}
	if stop < len(value) {
		return state != 0
	}
	return f.acceptAtEnd[state]
}

type rejectionNFASet [rejectionNFAWords]uint64

func (s *rejectionNFASet) add(state uint32) bool {
	word := state >> 6
	mask := uint64(1) << (state & 63)
	if s[word]&mask != 0 {
		return false
	}
	s[word] |= mask
	return true
}

func (s *rejectionNFASet) empty(words int) bool {
	for index := range words {
		if s[index] != 0 {
			return false
		}
	}
	return true
}

func (s *rejectionNFASet) hash(words int) uint64 {
	hash := uint64(1469598103934665603)
	for index := range words {
		hash = (hash ^ s[index]) * 1099511628211
	}
	return hash
}

func (s *rejectionNFASet) equal(other *rejectionNFASet, words int) bool {
	for index := range words {
		if s[index] != other[index] {
			return false
		}
	}
	return true
}

// relaxedRegexp widens only counted repetitions whose optional spread would
// otherwise multiply automaton states. Replacing x{m,n} with x{m,} adds strings and
// preserves L(re) within L(relaxedRegexp(re)). Unchanged syntax and rune backing
// stay shared: this pass, Simplify and syntax.Compile do not mutate their inputs.
func relaxedRegexp(re *syntax.Regexp) *syntax.Regexp {
	out := re
	for index, sub := range re.Sub {
		relaxed := relaxedRegexp(sub)
		if relaxed != sub {
			if out == re {
				cloned := *re
				cloned.Sub = slices.Clone(re.Sub)
				out = &cloned
			}
			out.Sub[index] = relaxed
		}
	}
	if out.Op == syntax.OpRepeat && out.Max >= 0 && out.Max-out.Min > rejectionRelaxSpread {
		if out == re {
			cloned := *re
			out = &cloned
		}
		out.Max = -1
	}
	return out
}

func instructionASCIISet(instruction *syntax.Inst) asciiSet {
	switch instruction.Op {
	case syntax.InstRuneAny:
		return asciiSet{^uint64(0), ^uint64(0)}
	case syntax.InstRuneAnyNotNL:
		set := asciiSet{^uint64(0), ^uint64(0)}
		set[0] &^= uint64(1) << '\n'
		return set
	case syntax.InstRune, syntax.InstRune1:
	default:
		return asciiSet{}
	}

	var set asciiSet
	if len(instruction.Rune) == 1 {
		if syntax.Flags(instruction.Arg)&syntax.FoldCase != 0 {
			for value := range 128 {
				if instruction.MatchRune(rune(value)) {
					set.add(byte(value))
				}
			}
		} else if instruction.Rune[0] < 0x80 {
			set.add(byte(instruction.Rune[0]))
		}
		return set
	}
	for index := 0; index+1 < len(instruction.Rune); index += 2 {
		low := max(rune(0), instruction.Rune[index])
		high := min(rune(0x7f), instruction.Rune[index+1])
		for value := low; value <= high; value++ {
			set.add(byte(value))
		}
	}
	return set
}

// instructionMatchesNonASCII proves when an instruction needs the widened Unicode
// transition. Fold-case ASCII literals must include aliases such as Kelvin sign
// and long-s. All malformed input bytes decode as RuneError, a non-ASCII rune.
func instructionMatchesNonASCII(instruction *syntax.Inst) bool {
	switch instruction.Op {
	case syntax.InstRuneAny, syntax.InstRuneAnyNotNL:
		return true
	case syntax.InstRune, syntax.InstRune1:
	default:
		return false
	}
	if len(instruction.Rune) == 1 {
		r := instruction.Rune[0]
		if r >= utf8.RuneSelf {
			return true
		}
		if syntax.Flags(instruction.Arg)&syntax.FoldCase != 0 {
			for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
				if next >= utf8.RuneSelf {
					return true
				}
			}
		}
		return false
	}
	for i := 1; i < len(instruction.Rune); i += 2 {
		if instruction.Rune[i] >= utf8.RuneSelf {
			return true
		}
	}
	return false
}

func newRejectionFilter(re *syntax.Regexp, anchored bool) (*rejectionFilter, bool) {
	return newRejectionFilterWithLimits(re, anchored, rejectionNFACap, rejectionDFACap)
}

// newRejectionFilterWithLimits builds a DFA for a relaxed regexp/syntax Thompson
// program. End-of-text assertions are resolved at EOF; other assertions become
// epsilon transitions, and broad repeats are widened.
// All non-ASCII runes share a symbol whose transitions union their possible rune
// instructions. Decoding once per rune preserves repetition counts, including for
// malformed UTF-8. Each widening preserves every original match.
func newRejectionFilterWithLimits(
	re *syntax.Regexp,
	anchored bool,
	maxNFAStates int,
	maxDFAStates int,
) (*rejectionFilter, bool) {
	program, err := syntax.Compile(relaxedRegexp(re).Simplify())
	if err != nil {
		return nil, false
	}
	return newRejectionFilterFromProgram(program, anchored, maxNFAStates, maxDFAStates)
}

// The anchored and unanchored DFAs can share this immutable construction input.
// Neither retains the program after its transition tables are built.
func newRejectionFilterFromProgram(program *syntax.Prog, anchored bool, maxNFAStates, maxDFAStates int) (*rejectionFilter, bool) {
	if program == nil || len(program.Inst) > maxNFAStates || len(program.Inst) > rejectionNFACap {
		return nil, false
	}
	nfaWords := (len(program.Inst) + 63) / 64

	edges := make([]asciiSet, len(program.Inst))
	var signatures [rejectionAlphabetSize]rejectionNFASet
	for state := range program.Inst {
		edges[state] = instructionASCIISet(&program.Inst[state])
		for value := range 128 {
			if edges[state].has(byte(value)) {
				signatures[value].add(uint32(state))
			}
		}
		if instructionMatchesNonASCII(&program.Inst[state]) {
			signatures[rejectionUnicodeSymbol].add(uint32(state))
		}
	}

	filter := &rejectionFilter{}
	classBuckets := make(map[uint64][]uint8, 8)
	var representatives [rejectionAlphabetSize]byte
	for value := range rejectionAlphabetSize {
		signature := &signatures[value]
		hash := signature.hash(nfaWords)
		class := uint8(0)
		found := false
		for _, candidate := range classBuckets[hash] {
			if signature.equal(&signatures[representatives[candidate]], nfaWords) {
				class = candidate
				found = true
				break
			}
		}
		if !found {
			class = uint8(filter.classes)
			filter.classes++
			classBuckets[hash] = append(classBuckets[hash], class)
			representatives[class] = byte(value)
		}
		filter.classOf[value] = class
	}
	filter.next = make([]uint16, 2*filter.classes)
	for class := range filter.classes {
		filter.next[filter.classes+class] = 1
	}
	filter.acceptAtEnd = []bool{false, true}

	accepting := rejectionNFASet{}
	for state := range program.Inst {
		if program.Inst[state].Op == syntax.InstMatch {
			accepting.add(uint32(state))
		}
	}

	// Reuse a worklist across closures. Removing each pending state from a
	// bitset would repeatedly scan all preceding words of large programs.
	pending := make([]uint32, 0, len(program.Inst))
	epsilonClosure := func(states *rejectionNFASet, atEnd bool) {
		pending = pending[:0]
		for wordIndex, word := range states[:nfaWords] {
			for word != 0 {
				bit := bits.TrailingZeros64(word)
				pending = append(pending, uint32(wordIndex*64+bit))
				word &= word - 1
			}
		}
		for len(pending) != 0 {
			last := len(pending) - 1
			instruction := &program.Inst[pending[last]]
			pending = pending[:last]
			add := func(next uint32) {
				if states.add(next) {
					pending = append(pending, next)
				}
			}
			switch instruction.Op {
			case syntax.InstAlt, syntax.InstAltMatch:
				add(instruction.Out)
				add(instruction.Arg)
			case syntax.InstCapture, syntax.InstNop:
				add(instruction.Out)
			case syntax.InstEmptyWidth:
				if syntax.EmptyOp(instruction.Arg)&syntax.EmptyEndText == 0 || atEnd {
					add(instruction.Out)
				}
			}
		}
	}

	isAccepting := func(states *rejectionNFASet) bool {
		for wordIndex := range nfaWords {
			if states[wordIndex]&accepting[wordIndex] != 0 {
				return true
			}
		}
		return false
	}

	var start rejectionNFASet
	start.add(uint32(program.Start))
	epsilonClosure(&start, false)
	if isAccepting(&start) {
		filter.start = 1
		return filter, true
	}
	if start.empty(nfaWords) {
		return filter, true
	}
	if maxDFAStates <= 2 {
		return nil, false
	}

	// Store only each program's live words, not a maximum-sized set per state.
	// Compact hash chains also avoid allocating a slice for every hash bucket.
	states := append([]uint64(nil), start[:nfaWords]...)
	stateLinks := []uint16{0}
	stateBuckets := map[uint64]uint16{start.hash(nfaWords): 2}
	filter.start = 2
	// The reachable-state queue grows in the loop; range would freeze its length.
	for queueIndex := 0; queueIndex < len(stateLinks); queueIndex++ {
		stateNumber := uint16(queueIndex + 2)
		filter.next = append(filter.next, make([]uint16, filter.classes)...)
		var ending rejectionNFASet
		copy(ending[:nfaWords], states[queueIndex*nfaWords:(queueIndex+1)*nfaWords])
		epsilonClosure(&ending, true)
		filter.acceptAtEnd = append(filter.acceptAtEnd, isAccepting(&ending))
		for class := range filter.classes {
			var moved rejectionNFASet
			representative := representatives[class]
			for wordIndex := range nfaWords {
				word := states[queueIndex*nfaWords+wordIndex]
				for word != 0 {
					bit := word & -word
					state := wordIndex*64 + bits.TrailingZeros64(bit)
					word &^= bit
					if edges[state].has(representative) ||
						representative == rejectionUnicodeSymbol && signatures[rejectionUnicodeSymbol][wordIndex]&bit != 0 {
						moved.add(program.Inst[state].Out)
					}
				}
			}
			if !anchored {
				moved.add(uint32(program.Start))
			}
			epsilonClosure(&moved, false)

			target := uint16(0)
			switch {
			case isAccepting(&moved):
				target = 1
			case moved.empty(nfaWords):
			default:
				hash := moved.hash(nfaWords)
				for candidate := stateBuckets[hash]; candidate != 0; candidate = stateLinks[candidate-2] {
					low := int(candidate-2) * nfaWords
					if slices.Equal(moved[:nfaWords], states[low:low+nfaWords]) {
						target = candidate
						break
					}
				}
				if target == 0 {
					nextNumber := len(stateLinks) + 2
					if nextNumber >= maxDFAStates || nextNumber >= 1<<16 {
						return nil, false
					}
					target = uint16(nextNumber)
					stateLinks = append(stateLinks, stateBuckets[hash])
					stateBuckets[hash] = target
					states = append(states, moved[:nfaWords]...)
				}
			}
			filter.next[int(stateNumber)*filter.classes+class] = target
		}
	}
	// Construction grows these tables incrementally. Drop substantial spare
	// capacity once, without changing the flat lookup layout used by run.
	if cap(filter.next)-len(filter.next) >= (len(filter.next)+7)/8 {
		next := make([]uint16, len(filter.next))
		copy(next, filter.next)
		filter.next = next
	}
	if cap(filter.acceptAtEnd)-len(filter.acceptAtEnd) >= (len(filter.acceptAtEnd)+7)/8 {
		ending := make([]bool, len(filter.acceptAtEnd))
		copy(ending, filter.acceptAtEnd)
		filter.acceptAtEnd = ending
	}
	return filter, true
}
