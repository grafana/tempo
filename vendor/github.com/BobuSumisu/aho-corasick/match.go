package ahocorasick

import (
	"bytes"
	"fmt"
)

// Match represents a matched pattern in the input.
type Match struct {
	pos     int64
	pattern int64
	match   []byte
}

func newMatch(pos, pattern int64, match []byte) *Match {
	return &Match{pos, pattern, match}
}

func newMatchString(pos, pattern int64, match string) *Match {
	return &Match{pos: pos, pattern: pattern, match: []byte(match)}
}

func (m *Match) String() string {
	return fmt.Sprintf("{%d %d %q}", m.pos, m.pattern, m.match)
}

// Pos returns the byte position of the match.
func (m *Match) Pos() int64 {
	return m.pos
}

// Pattern returns the pattern id of the match.
func (m *Match) Pattern() int64 {
	return m.pattern
}

// Match returns the pattern matched.
func (m *Match) Match() []byte {
	return m.match
}

// MatchString returns the pattern matched as a string.
func (m *Match) MatchString() string {
	return string(m.match)
}

// MatchEqual check whether two matches are equal (i.e. at same position, pattern and same pattern).
func MatchEqual(a, b *Match) bool {
	return a.pos == b.pos && a.pattern == b.pattern && bytes.Equal(a.match, b.match)
}
