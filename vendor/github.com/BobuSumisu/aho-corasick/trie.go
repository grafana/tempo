package ahocorasick

const (
	rootState int64 = 1
	nilState  int64 = 0
)

// Trie represents a trie of patterns with extra links as per the Aho-Corasick algorithm.
type Trie struct {
	dict     []int64
	trans    [][256]int64
	failLink []int64
	dictLink []int64
	pattern  []int64
}

// Walk calls this function on any match, giving the end position, length of the matched bytes,
// and the pattern number.
type WalkFn func(end, n, pattern int64) bool

// Walk runs the algorithm on a given output, calling the supplied callback function on every
// match. The algorithm will terminate if the callback function returns false.
func (tr *Trie) Walk(input []byte, fn WalkFn) {
	s := rootState

	for i, c := range input {
		t := tr.trans[s][c]

		if t == nilState {
			for u := tr.failLink[s]; u != rootState; u = tr.failLink[u] {
				if t = tr.trans[u][c]; t != nilState {
					break
				}
			}

			if t == nilState {
				if t = tr.trans[rootState][c]; t == nilState {
					t = rootState
				}
			}
		}

		s = t

		if tr.dict[s] != 0 {
			if !fn(int64(i), tr.dict[s], tr.pattern[s]) {
				return
			}
		}

		if tr.dictLink[s] != nilState {
			for u := tr.dictLink[s]; u != nilState; u = tr.dictLink[u] {
				if !fn(int64(i), tr.dict[u], tr.pattern[u]) {
					return
				}
			}
		}
	}
}

// Match runs the Aho-Corasick string-search algorithm on a byte input.
func (tr *Trie) Match(input []byte) []*Match {
	matches := make([]*Match, 0)
	tr.Walk(input, func(end, n, pattern int64) bool {
		pos := end - n + 1
		matches = append(matches, newMatch(pos, pattern, input[pos:pos+n]))
		return true
	})
	return matches
}

// MatchFirst is the same as Match, but returns after first successful match.
func (tr *Trie) MatchFirst(input []byte) *Match {
	var match *Match
	tr.Walk(input, func(end, n, pattern int64) bool {
		pos := end - n + 1
		match = &Match{pos: pos, match: input[pos : pos+n]}
		return false
	})
	return match
}

// MatchString runs the Aho-Corasick string-search algorithm on a string input.
func (tr *Trie) MatchString(input string) []*Match {
	return tr.Match([]byte(input))
}

// MatchFirstString is the same as MatchString, but returns after first successful match.
func (tr *Trie) MatchFirstString(input string) *Match {
	return tr.MatchFirst([]byte(input))
}
