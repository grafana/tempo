package secrets

import (
	"math/bits"
	"regexp/syntax"
	"strings"
)

// asciiRequirement proves that every match consumes at least one strict-ASCII
// rune. Unknown expressions and Unicode alternatives remain eligible.
type asciiRequirement bool

func requiredASCII(re *syntax.Regexp) asciiRequirement {
	if re == nil {
		return false
	}
	switch re.Op {
	case syntax.OpLiteral:
		for i := range re.Rune {
			inst := syntax.Inst{Op: syntax.InstRune, Rune: re.Rune[i : i+1], Arg: uint32(re.Flags & syntax.FoldCase)}
			if !instructionMatchesNonASCII(&inst) {
				return true
			}
		}
	case syntax.OpCharClass:
		inst := syntax.Inst{Op: syntax.InstRune, Rune: re.Rune, Arg: uint32(re.Flags & syntax.FoldCase)}
		return asciiRequirement(!instructionMatchesNonASCII(&inst))
	case syntax.OpCapture, syntax.OpPlus:
		return requiredASCII(re.Sub[0])
	case syntax.OpRepeat:
		if re.Min > 0 {
			return requiredASCII(re.Sub[0])
		}
	case syntax.OpConcat:
		for _, sub := range re.Sub {
			if requiredASCII(sub) {
				return true
			}
		}
	case syntax.OpAlternate:
		for _, sub := range re.Sub {
			if !requiredASCII(sub) {
				return false
			}
		}
		return asciiRequirement(len(re.Sub) != 0)
	}
	return false
}

// possible skips the presence scan for ASCII inputs and rules without a proof.
// A byte below RuneSelf always denotes an ASCII rune, even in malformed UTF-8.
func (required asciiRequirement) possible(value string, hits *keywordHits) bool {
	if !bool(required) || hits == nil || !hits.nonASCII {
		return true
	}
	if !hits.asciiChecked {
		hits.asciiChecked = true
		for i := range len(value) {
			if value[i] < 0x80 {
				hits.asciiPresent = true
				break
			}
		}
	}
	return hits.asciiPresent
}

// asciiRun is a necessary (not sufficient) condition: width consecutive matching
// runes must occur. When allowsUnicode is false, those runes must all be ASCII,
// so the byte scanner is also sound on otherwise non-ASCII input.
type asciiRun struct {
	set             asciiSet
	width           int
	allowsUnicode   bool
	unicodeFoldOnly bool
}

// canReject reports whether every matching run is ASCII on this input. Unicode
// outside the run does not disable it when only absent simple-fold aliases could
// participate. Invalid keyword guidance may lack the alias inventory.
func (r *asciiRun) canReject(hits *keywordHits) bool {
	return !r.allowsUnicode || hits != nil && !hits.invalid &&
		(!hits.nonASCII || r.unicodeFoldOnly && !hits.asciiFoldAlias)
}

// exists searches for an ASCII run. An out-of-class byte at position p
// rules out every candidate containing p, so the next candidate ends at p+width.
// Unlike a DFA scan this can skip width bytes after a single failed membership check.
func (r *asciiRun) exists(value string) bool {
	for end := r.width - 1; end < len(value); {
		matched := 0
		for matched < r.width && r.set.has(value[end-matched]) {
			matched++
		}
		if matched == r.width {
			return true
		}
		end += r.width - matched
	}
	return false
}

// requiredASCIIRun derives one mandatory run from a repeated single-rune atom.
// Concatenations require every child; alternations require a run from every branch,
// so their classes are unioned and their widths reduced to the shortest branch.
// Optional expressions cannot provide a necessary condition. Short or broad runs
// are omitted because checking them costs more than they save.
func requiredASCIIRun(re *syntax.Regexp) asciiRun {
	var run asciiRun
	switch re.Op {
	case syntax.OpCapture, syntax.OpPlus:
		return requiredASCIIRun(re.Sub[0])
	case syntax.OpRepeat:
		if re.Min == 0 {
			return run
		}
		sub := re.Sub[0]
		for sub.Op == syntax.OpCapture {
			sub = sub.Sub[0]
		}
		if sub.Op == syntax.OpCharClass || sub.Op == syntax.OpLiteral && len(sub.Rune) == 1 {
			inst := syntax.Inst{Op: syntax.InstRune, Rune: sub.Rune, Arg: uint32(sub.Flags & syntax.FoldCase)}
			run = asciiRun{
				set:             instructionASCIISet(&inst),
				width:           re.Min,
				allowsUnicode:   instructionMatchesNonASCII(&inst),
				unicodeFoldOnly: regexpASCIIOrFoldOnly(sub),
			}
		} else {
			return requiredASCIIRun(sub)
		}
	case syntax.OpConcat:
		// Favor wider runs with fewer accepted bytes. This only chooses between
		// necessary conditions; it never changes which inputs can be accepted.
		members := 0
		for _, sub := range re.Sub {
			candidate := requiredASCIIRun(sub)
			candidateMembers := bits.OnesCount64(candidate.set[0]) + bits.OnesCount64(candidate.set[1])
			if candidate.width > 0 && (run.width == 0 || candidate.width*members > run.width*candidateMembers) {
				run = candidate
				members = candidateMembers
			}
		}
	case syntax.OpAlternate:
		run = requiredASCIIRun(re.Sub[0])
		for _, sub := range re.Sub[1:] {
			candidate := requiredASCIIRun(sub)
			run.width = min(run.width, candidate.width)
			run.set[0] |= candidate.set[0]
			run.set[1] |= candidate.set[1]
			run.allowsUnicode = run.allowsUnicode || candidate.allowsUnicode
			run.unicodeFoldOnly = run.unicodeFoldOnly && candidate.unicodeFoldOnly
		}
	}
	members := bits.OnesCount64(run.set[0]) + bits.OnesCount64(run.set[1])
	if (run.width < 8 && (run.width < 4 || members > 16)) || members > 64 {
		return asciiRun{}
	}
	return run
}

// requiredPunctuation derives exact ASCII punctuation present in every match.
// Punctuation has no Unicode case-fold aliases. Optional children contribute
// nothing; alternations retain only bytes required by every branch.
func requiredPunctuation(re *syntax.Regexp) asciiSet {
	var required asciiSet
	switch re.Op {
	case syntax.OpLiteral:
		for _, r := range re.Rune {
			if r > ' ' && r < 127 && !syntax.IsWordChar(r) {
				required[r>>6] |= uint64(1) << (r & 63)
			}
		}
	case syntax.OpCapture, syntax.OpPlus:
		return requiredPunctuation(re.Sub[0])
	case syntax.OpRepeat:
		if re.Min > 0 {
			return requiredPunctuation(re.Sub[0])
		}
	case syntax.OpConcat:
		for _, sub := range re.Sub {
			part := requiredPunctuation(sub)
			required[0] |= part[0]
			required[1] |= part[1]
		}
	case syntax.OpAlternate:
		required = requiredPunctuation(re.Sub[0])
		for _, sub := range re.Sub[1:] {
			part := requiredPunctuation(sub)
			required[0] &= part[0]
			required[1] &= part[1]
		}
	}
	return required
}

// requiredPunctuationChoice derives a set of which at least one byte must occur.
// This covers separators such as [:=] that have no individually mandatory byte.
// Requirements already proved by exact are omitted because they add no rejection.
func requiredPunctuationChoice(re *syntax.Regexp, exact asciiSet) asciiSet {
	var choice asciiSet
	switch re.Op {
	case syntax.OpLiteral:
		choice = requiredPunctuation(re)
		choice[0] &^= exact[0]
		choice[1] &^= exact[1]
	case syntax.OpCharClass:
		for i := 0; i < len(re.Rune); i += 2 {
			low, high := re.Rune[i], re.Rune[i+1]
			if low <= ' ' || high >= 127 {
				return asciiSet{}
			}
			for r := low; r <= high; r++ {
				if syntax.IsWordChar(r) {
					return asciiSet{}
				}
				choice[r>>6] |= uint64(1) << (r & 63)
			}
		}
		// Removing an alternative would be unsound: the class may choose it.
		if choice[0]&exact[0] != 0 || choice[1]&exact[1] != 0 {
			return asciiSet{}
		}
	case syntax.OpCapture, syntax.OpPlus:
		return requiredPunctuationChoice(re.Sub[0], exact)
	case syntax.OpRepeat:
		if re.Min > 0 {
			return requiredPunctuationChoice(re.Sub[0], exact)
		}
	case syntax.OpConcat:
		best := 129
		for _, sub := range re.Sub {
			part := requiredPunctuationChoice(sub, exact)
			count := bits.OnesCount64(part[0]) + bits.OnesCount64(part[1])
			if count != 0 && count < best {
				choice, best = part, count
			}
		}
	case syntax.OpAlternate:
		for _, sub := range re.Sub {
			part := requiredPunctuationChoice(sub, exact)
			if part[0] == 0 && part[1] == 0 {
				return asciiSet{}
			}
			choice[0] |= part[0]
			choice[1] |= part[1]
		}
	}
	return choice
}

// punctuationPresence lazily checks only bytes required by candidate rules.
// Each byte is searched at most once per value, including cached rejections.
// The cache belongs to one evaluation, never to the immutable compiled rules.
type punctuationPresence struct {
	checked asciiSet
	absent  asciiSet
}

func (p *punctuationPresence) contains(value string, required asciiSet) bool {
	if required[0]&p.absent[0] != 0 || required[1]&p.absent[1] != 0 {
		return false
	}
	for wordIndex, word := range required {
		for unchecked := word &^ p.checked[wordIndex]; unchecked != 0; unchecked &= unchecked - 1 {
			bit := bits.TrailingZeros64(unchecked)
			mask := uint64(1) << bit
			p.checked[wordIndex] |= mask
			if strings.IndexByte(value, byte(wordIndex*64+bit)) < 0 {
				p.absent[wordIndex] |= mask
				return false
			}
		}
	}
	return true
}

func (p *punctuationPresence) containsAny(value string, choice asciiSet) bool {
	if choice[0]&p.checked[0]&^p.absent[0] != 0 || choice[1]&p.checked[1]&^p.absent[1] != 0 {
		return true
	}
	for wordIndex, word := range choice {
		for unchecked := word &^ p.checked[wordIndex]; unchecked != 0; unchecked &= unchecked - 1 {
			bit := bits.TrailingZeros64(unchecked)
			mask := uint64(1) << bit
			p.checked[wordIndex] |= mask
			if strings.IndexByte(value, byte(wordIndex*64+bit)) >= 0 {
				return true
			}
			p.absent[wordIndex] |= mask
		}
	}
	return false
}
