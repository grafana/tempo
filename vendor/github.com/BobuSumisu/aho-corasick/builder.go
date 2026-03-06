package ahocorasick

import (
	"bufio"
	"encoding/hex"
	"os"
	"strings"
)

type state struct {
	id       int64
	value    byte
	parent   *state
	trans    map[byte]*state
	dict     int64
	failLink *state
	dictLink *state
	pattern  int64
}

// TrieBuilder is used to build Tries.
type TrieBuilder struct {
	states      []*state
	root        *state
	numPatterns int64
}

// NewTrieBuilder creates and initializes a new TrieBuilder.
func NewTrieBuilder() *TrieBuilder {
	tb := &TrieBuilder{
		states:      make([]*state, 0),
		root:        nil,
		numPatterns: 0,
	}
	tb.addState(0, nil)
	tb.addState(0, nil)
	tb.root = tb.states[1]
	return tb
}

func (tb *TrieBuilder) addState(value byte, parent *state) *state {
	s := &state{
		id:       int64(len(tb.states)),
		value:    value,
		parent:   parent,
		trans:    make(map[byte]*state),
		dict:     0,
		failLink: nil,
		dictLink: nil,
		pattern:  0,
	}
	tb.states = append(tb.states, s)
	return s
}

// AddPattern adds a byte pattern to the Trie under construction.
func (tb *TrieBuilder) AddPattern(pattern []byte) *TrieBuilder {
	s := tb.root
	var t *state
	var ok bool

	for _, c := range pattern {
		if t, ok = s.trans[c]; !ok {
			t = tb.addState(c, s)
			s.trans[c] = t
		}
		s = t
	}

	s.dict = int64(len(pattern))
	s.pattern = tb.numPatterns
	tb.numPatterns++

	return tb
}

// AddPatterns adds multiple byte patterns to the Trie.
func (tb *TrieBuilder) AddPatterns(patterns [][]byte) *TrieBuilder {
	for _, pattern := range patterns {
		tb.AddPattern(pattern)
	}
	return tb
}

// AddString adds a string pattern to the Trie under construction.
func (tb *TrieBuilder) AddString(pattern string) *TrieBuilder {
	return tb.AddPattern([]byte(pattern))
}

// AddStrings add multiple strings to the Trie.
func (tb *TrieBuilder) AddStrings(patterns []string) *TrieBuilder {
	for _, pattern := range patterns {
		tb.AddString(pattern)
	}
	return tb
}

// LoadPatterns loads byte patterns from a file. Expects one pattern per line in hexadecimal form.
func (tb *TrieBuilder) LoadPatterns(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(f)

	for s.Scan() {
		str := strings.TrimSpace(s.Text())
		if len(str) != 0 {
			pattern, err := hex.DecodeString(str)
			if err != nil {
				return err
			}
			tb.AddPattern(pattern)
		}
	}

	return s.Err()
}

// LoadStrings loads string patterns from a file. Expects one pattern per line.
func (tb *TrieBuilder) LoadStrings(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(f)

	for s.Scan() {
		str := strings.TrimSpace(s.Text())
		if len(str) != 0 {
			tb.AddString(str)
		}
	}

	return s.Err()
}

// Build constructs the Trie.
func (tb *TrieBuilder) Build() *Trie {

	tb.computeFailLinks(tb.root)
	tb.computeDictLinks(tb.root)

	numStates := len(tb.states)

	dict := make([]int64, numStates)
	trans := make([][256]int64, numStates)
	failLink := make([]int64, numStates)
	dictLink := make([]int64, numStates)
	pattern := make([]int64, numStates)

	for i, s := range tb.states {
		dict[i] = s.dict
		pattern[i] = s.pattern
		for c, t := range s.trans {
			trans[i][c] = t.id
		}
		if s.failLink != nil {
			failLink[i] = s.failLink.id
		}
		if s.dictLink != nil {
			dictLink[i] = s.dictLink.id
		}
	}

	return &Trie{dict, trans, failLink, dictLink, pattern}
}

func (tb *TrieBuilder) computeFailLinks(s *state) {
	if s.failLink != nil {
		return
	}

	if s == tb.root || s.parent == tb.root {
		s.failLink = tb.root
	} else {
		var ok bool

		for t := s.parent.failLink; t != tb.root; t = t.failLink {
			if t.failLink == nil {
				tb.computeFailLinks(t)
			}

			if s.failLink, ok = t.trans[s.value]; ok {
				break
			}
		}

		if s.failLink == nil {
			if s.failLink, ok = tb.root.trans[s.value]; !ok {
				s.failLink = tb.root
			}
		}
	}

	for _, t := range s.trans {
		tb.computeFailLinks(t)
	}
}

func (tb *TrieBuilder) computeDictLinks(s *state) {
	if s != tb.root {
		for t := s.failLink; t != tb.root; t = t.failLink {
			if t.dict != 0 {
				s.dictLink = t
				break
			}
		}
	}

	for _, t := range s.trans {
		tb.computeDictLinks(t)
	}
}
