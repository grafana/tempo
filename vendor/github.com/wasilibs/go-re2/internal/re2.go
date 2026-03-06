package internal

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
	"unicode/utf8"
)

// Use same max size as regexp package.
// https://github.com/golang/go/blob/master/src/regexp/syntax/parse.go#L95
const maxSize = 128 << 20

type Regexp struct {
	ptr wasmPtr

	opts CompileOptions

	expr string

	numMatches     int
	groupNames     []string
	groupNamesOnce sync.Once

	abi *libre2ABI

	released uint32
}

// Copy returns a new Regexp object copied from re.
// Calling Longest on one copy does not affect another.
//
// Deprecated: In earlier releases, when using a Regexp in multiple goroutines,
// giving each goroutine its own copy helped to avoid lock contention.
// As of Go 1.12, using Copy is no longer necessary to avoid lock contention.
// Copy may still be appropriate if the reason for its use is to make
// two copies with different Longest settings.
func (re *Regexp) Copy() *Regexp {
	// Recompiling is slower than this should be but for a deprecated method it
	// is probably fine. The alternative would be to have reference counting to
	// make sure regex is only deleted when the last reference is gone.

	// Recompiling regex, no chance of err so don't bother checking it.
	c, _ := Compile(re.expr, re.opts)
	return c
}

type CompileOptions struct {
	Posix           bool
	Longest         bool
	CaseInsensitive bool
	Latin1          bool
}

func Compile(expr string, opts CompileOptions) (*Regexp, error) {
	abi := newABI()
	alloc := abi.startOperation(len(expr) + 2)
	defer abi.endOperation(alloc)

	cs := alloc.newCString(expr)

	rePtr := newRE(abi, cs, opts)
	errCode, errArg := reError(abi, rePtr)
	switch errCode {
	case 0:
	// No error.
	case 1:
		return nil, fmt.Errorf("error parsing regexp: unexpected error: %#q", errArg)
	case 2:
		return nil, fmt.Errorf("error parsing regexp: invalid escape sequence: %#q", errArg)
	case 3:
		return nil, fmt.Errorf("error parsing regexp: bad character class: %#q", errArg)
	case 4:
		return nil, fmt.Errorf("error parsing regexp: invalid character class range: %#q", errArg)
	case 5:
		return nil, fmt.Errorf("error parsing regexp: missing closing ]: %#q", errArg)
	case 6:
		return nil, fmt.Errorf("error parsing regexp: missing closing ): %#q", errArg)
	case 7:
		return nil, fmt.Errorf("error parsing regexp: unexpected ): %#q", errArg)
	case 8:
		return nil, fmt.Errorf("error parsing regexp: trailing backslash at end of expression: %#q", errArg)
	case 9:
		return nil, fmt.Errorf("error parsing regexp: missing argument to repetition operator: %#q", errArg)
	case 10:
		return nil, fmt.Errorf("error parsing regexp: bad repitition argument: %#q", errArg)
	case 11:
		return nil, fmt.Errorf("error parsing regexp: invalid nested repetition operator: %#q", errArg)
	case 12:
		return nil, fmt.Errorf("error parsing regexp: bad perl operator: %#q", errArg)
	case 13:
		return nil, fmt.Errorf("error parsing regexp: invalid UTF-8 in regexp: %#q", errArg)
	case 14:
		return nil, fmt.Errorf("error parsing regexp: bad named capture group: %#q", errArg)
	case 15:
		// TODO(anuraaga): While the unit test passes, it is likely that the actual limit is currently
		// different than regexp.
		return nil, fmt.Errorf("error parsing regexp: expression too large")
	}

	// Does not include whole expression match, e.g. $0
	numGroups := numCapturingGroups(abi, rePtr)

	re := &Regexp{
		ptr:        rePtr,
		opts:       opts,
		expr:       expr,
		numMatches: numGroups + 1,
		abi:        abi,
	}

	// Use func(interface{}) form for nottinygc compatibility.
	runtime.SetFinalizer(re, func(obj interface{}) {
		obj.(*Regexp).release()
	})

	return re, nil
}

// Expand appends template to dst and returns the result; during the
// append, Expand replaces variables in the template with corresponding
// matches drawn from src. The match slice should have been returned by
// FindSubmatchIndex.
//
// In the template, a variable is denoted by a substring of the form
// $name or ${name}, where name is a non-empty sequence of letters,
// digits, and underscores. A purely numeric name like $1 refers to
// the submatch with the corresponding index; other names refer to
// capturing parentheses named with the (?P<name>...) syntax. A
// reference to an out of range or unmatched index or a name that is not
// present in the regular expression is replaced with an empty slice.
//
// In the $name form, name is taken to be as long as possible: $1x is
// equivalent to ${1x}, not ${1}x, and, $10 is equivalent to ${10}, not ${1}0.
//
// To insert a literal $ in the output, use $$ in the template.
func (re *Regexp) Expand(dst []byte, template []byte, src []byte, match []int) []byte {
	return re.expand(dst, string(template), src, "", match)
}

// ExpandString is like Expand but the template and source are strings.
// It appends to and returns a byte slice in order to give the calling
// code control over allocation.
func (re *Regexp) ExpandString(dst []byte, template string, src string, match []int) []byte {
	return re.expand(dst, template, nil, src, match)
}

func (re *Regexp) expand(dst []byte, template string, bsrc []byte, src string, match []int) []byte {
	for len(template) > 0 {
		before, after, ok := strings.Cut(template, "$")
		if !ok {
			break
		}
		dst = append(dst, before...)
		template = after
		if template != "" && template[0] == '$' {
			// Treat $$ as $.
			dst = append(dst, '$')
			template = template[1:]
			continue
		}
		name, num, rest, ok := extract(template)
		if !ok {
			// Malformed; treat $ as raw text.
			dst = append(dst, '$')
			continue
		}
		template = rest
		if num >= 0 {
			if 2*num+1 < len(match) && match[2*num] >= 0 {
				if bsrc != nil {
					dst = append(dst, bsrc[match[2*num]:match[2*num+1]]...)
				} else {
					dst = append(dst, src[match[2*num]:match[2*num+1]]...)
				}
			}
		} else {
			for i, namei := range re.SubexpNames() {
				if name == namei && 2*i+1 < len(match) && match[2*i] >= 0 {
					if bsrc != nil {
						dst = append(dst, bsrc[match[2*i]:match[2*i+1]]...)
					} else {
						dst = append(dst, src[match[2*i]:match[2*i+1]]...)
					}
					break
				}
			}
		}
	}

	runtime.KeepAlive(re) // don't allow finalizer to run during method

	dst = append(dst, template...)
	return dst
}

// Find returns a slice holding the text of the leftmost match in b of the regular expression.
// A return value of nil indicates no match.
func (re *Regexp) Find(b []byte) []byte {
	alloc := re.abi.startOperation(len(b) + 8)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCStringFromBytes(b)

	var dstCap [2]int

	dst := re.find(&alloc, cs, dstCap[:0])
	return matchedBytes(b, dst)
}

// FindIndex returns a two-element slice of integers defining the location of
// the leftmost match in b of the regular expression. The match itself is at
// b[loc[0]:loc[1]].
// A return value of nil indicates no match.
func (re *Regexp) FindIndex(b []byte) (loc []int) {
	alloc := re.abi.startOperation(len(b) + 8)
	defer re.abi.endOperation(alloc)
	cs := alloc.newCStringFromBytes(b)

	res := re.find(&alloc, cs, nil)
	runtime.KeepAlive(b)
	return res
}

// FindString returns a string holding the text of the leftmost match in s of the regular
// expression. If there is no match, the return value is an empty string,
// but it will also be empty if the regular expression successfully matches
// an empty string. Use FindStringIndex or FindStringSubmatch if it is
// necessary to distinguish these cases.
func (re *Regexp) FindString(s string) string {
	alloc := re.abi.startOperation(len(s) + 8)
	defer re.abi.endOperation(alloc)
	cs := alloc.newCString(s)

	var dstCap [2]int

	dst := re.find(&alloc, cs, dstCap[:0])
	return matchedString(s, dst)
}

// FindStringIndex returns a two-element slice of integers defining the
// location of the leftmost match in s of the regular expression. The match
// itself is at s[loc[0]:loc[1]].
// A return value of nil indicates no match.
func (re *Regexp) FindStringIndex(s string) (loc []int) {
	alloc := re.abi.startOperation(len(s) + 8)
	defer re.abi.endOperation(alloc)
	cs := alloc.newCString(s)

	res := re.find(&alloc, cs, nil)
	runtime.KeepAlive(s)
	return res
}

func (re *Regexp) find(alloc *allocation, cs cString, dstCap []int) []int {
	matchArr := alloc.newCStringArray(1)
	defer matchArr.free()

	res := match(re, cs, matchArr.ptr, 1)
	if !res {
		return nil
	}

	m := readMatch(alloc, cs, matchArr.ptr, dstCap)
	runtime.KeepAlive(matchArr)

	runtime.KeepAlive(re) // don't allow finalizer to run during method

	return m
}

// FindAll is the 'All' version of Find; it returns a slice of all successive
// matches of the expression, as defined by the 'All' description in the
// package comment.
// A return value of nil indicates no match.
func (re *Regexp) FindAll(b []byte, n int) [][]byte {
	alloc := re.abi.startOperation(len(b) + 16)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCStringFromBytes(b)

	var matches [][]byte

	re.findAll(&alloc, b, "", cs, n, func(match []int) {
		matches = append(matches, matchedBytes(b, match))
	})

	return matches
}

// FindAllIndex is the 'All' version of FindIndex; it returns a slice of all
// successive matches of the expression, as defined by the 'All' description
// in the package comment.
// A return value of nil indicates no match.
func (re *Regexp) FindAllIndex(b []byte, n int) [][]int {
	alloc := re.abi.startOperation(len(b) + 16)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCStringFromBytes(b)

	var matches [][]int

	re.findAll(&alloc, b, "", cs, n, func(match []int) {
		matches = append(matches, append([]int(nil), match...))
	})

	res := matches
	runtime.KeepAlive(b)
	return res
}

// FindAllString is the 'All' version of FindString; it returns a slice of all
// successive matches of the expression, as defined by the 'All' description
// in the package comment.
// A return value of nil indicates no match.
func (re *Regexp) FindAllString(s string, n int) []string {
	alloc := re.abi.startOperation(len(s) + 16)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCString(s)

	var matches []string

	re.findAll(&alloc, nil, s, cs, n, func(match []int) {
		matches = append(matches, matchedString(s, match))
	})

	return matches
}

// FindAllStringIndex is the 'All' version of FindStringIndex; it returns a
// slice of all successive matches of the expression, as defined by the 'All'
// description in the package comment.
// A return value of nil indicates no match.
func (re *Regexp) FindAllStringIndex(s string, n int) [][]int {
	alloc := re.abi.startOperation(len(s) + 16)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCString(s)

	var matches [][]int

	re.findAll(&alloc, nil, s, cs, n, func(match []int) {
		matches = append(matches, append([]int(nil), match...))
	})

	res := matches
	runtime.KeepAlive(s)
	return res
}

func (re *Regexp) findAll(alloc *allocation, bsrc []byte, src string, cs cString, n int, deliver func(match []int)) {
	var dstCap [2]int

	if n < 0 {
		n = cs.length + 1
	}

	matchArr := alloc.newCStringArray(1)
	defer matchArr.free()

	count := 0
	prevMatchEnd := -1
	pos := 0
	for pos < cs.length+1 {
		if !matchFrom(re, cs, pos, matchArr.ptr, 1) {
			break
		}

		match := readMatch(alloc, cs, matchArr.ptr, dstCap[:0])
		accept := true
		// Check if it's an empty match following a match, which we ignore.
		if match[0] == match[1] && match[0] == prevMatchEnd {
			// We don't allow an empty match right
			// after a previous match, so ignore it.
			accept = false
		}

		pos = nextPos(bsrc, src, pos, match[1])

		if accept {
			deliver(match)
			count++
		}
		prevMatchEnd = match[1]

		if count == n {
			break
		}
	}

	runtime.KeepAlive(matchArr)

	runtime.KeepAlive(re) // don't allow finalizer to run during method
}

// FindAllSubmatch is the 'All' version of FindSubmatch; it returns a slice
// of all successive matches of the expression, as defined by the 'All'
// description in the package comment.
// A return value of nil indicates no match.
func (re *Regexp) FindAllSubmatch(b []byte, n int) [][][]byte {
	alloc := re.abi.startOperation(len(b) + 8*re.numMatches + 8)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCStringFromBytes(b)

	var matches [][][]byte

	re.findAllSubmatch(&alloc, b, "", cs, re.numMatches, n, func(match []int) {
		matched := make([][]byte, len(match)/2)
		for i := 0; i < len(match); i += 2 {
			matched[i/2] = matchedBytes(b, match[i:i+2])
		}
		matches = append(matches, matched)
	})

	return matches
}

// FindAllSubmatchIndex is the 'All' version of FindSubmatchIndex; it returns
// a slice of all successive matches of the expression, as defined by the
// 'All' description in the package comment.
// A return value of nil indicates no match.
func (re *Regexp) FindAllSubmatchIndex(b []byte, n int) [][]int {
	alloc := re.abi.startOperation(len(b) + 8*re.numMatches + 8)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCStringFromBytes(b)

	var matches [][]int

	re.findAllSubmatch(&alloc, b, "", cs, re.numMatches, n, func(match []int) {
		matches = append(matches, match)
	})

	res := matches
	runtime.KeepAlive(b)
	return res
}

// FindAllStringSubmatch is the 'All' version of FindStringSubmatch; it
// returns a slice of all successive matches of the expression, as defined by
// the 'All' description in the package comment.
// A return value of nil indicates no match.
func (re *Regexp) FindAllStringSubmatch(s string, n int) [][]string {
	alloc := re.abi.startOperation(len(s) + 8*re.numMatches + 8)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCString(s)

	var matches [][]string

	re.findAllSubmatch(&alloc, nil, s, cs, re.numMatches, n, func(match []int) {
		matched := make([]string, len(match)/2)
		for i := 0; i < len(match); i += 2 {
			matched[i/2] = matchedString(s, match[i:i+2])
		}
		matches = append(matches, matched)
	})

	return matches
}

// FindAllStringSubmatchIndex is the 'All' version of
// FindStringSubmatchIndex; it returns a slice of all successive matches of
// the expression, as defined by the 'All' description in the package
// comment.
// A return value of nil indicates no match.
func (re *Regexp) FindAllStringSubmatchIndex(s string, n int) [][]int {
	alloc := re.abi.startOperation(len(s) + 8*re.numMatches + 8)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCString(s)

	var matches [][]int

	re.findAllSubmatch(&alloc, nil, s, cs, re.numMatches, n, func(match []int) {
		matches = append(matches, match)
	})

	res := matches
	runtime.KeepAlive(s)
	return res
}

func (re *Regexp) findAllSubmatch(alloc *allocation, bsrc []byte, src string, cs cString, nmatch, n int, deliver func(match []int)) {
	if n < 0 {
		n = cs.length + 1
	}

	matchArr := alloc.newCStringArray(nmatch)
	defer matchArr.free()

	count := 0
	prevMatchEnd := -1
	pos := 0
	for pos < cs.length+1 {
		if !matchFrom(re, cs, pos, matchArr.ptr, uint32(nmatch)) {
			break
		}

		var matches []int
		accept := true
		readMatches(alloc, cs, matchArr.ptr, nmatch, func(match []int) bool {
			if len(matches) == 0 {
				// First match, check if it's an empty match following a match, which we ignore.
				if match[0] == match[1] && match[0] == prevMatchEnd {
					accept = false
				}

				pos = nextPos(bsrc, src, pos, match[1])
				prevMatchEnd = match[1]
			}
			if accept {
				matches = append(matches, match...)
				return true
			} else {
				return false
			}
		})
		if accept {
			deliver(matches)
		}
		count++

		if count == n {
			break
		}
	}

	runtime.KeepAlive(matchArr)

	runtime.KeepAlive(re) // don't allow finalizer to run during method
}

// FindSubmatch returns a slice of slices holding the text of the leftmost
// match of the regular expression in b and the matches, if any, of its
// subexpressions, as defined by the 'Submatch' descriptions in the package
// comment.
// A return value of nil indicates no match.
func (re *Regexp) FindSubmatch(b []byte) [][]byte {
	alloc := re.abi.startOperation(len(b) + 8*re.numMatches)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCStringFromBytes(b)

	var matches [][]byte

	re.findSubmatch(&alloc, cs, func(match []int) bool {
		matches = append(matches, matchedBytes(b, match))
		return true
	})

	return matches
}

// FindSubmatchIndex returns a slice holding the index pairs identifying the
// leftmost match of the regular expression in b and the matches, if any, of
// its subexpressions, as defined by the 'Submatch' and 'Index' descriptions
// in the package comment.
// A return value of nil indicates no match.
func (re *Regexp) FindSubmatchIndex(b []byte) []int {
	alloc := re.abi.startOperation(len(b) + 8*re.numMatches)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCStringFromBytes(b)

	var matches []int

	re.findSubmatch(&alloc, cs, func(match []int) bool {
		matches = append(matches, match...)
		return true
	})

	res := matches
	runtime.KeepAlive(b)
	return res
}

func (re *Regexp) FindStringSubmatch(s string) []string {
	alloc := re.abi.startOperation(len(s) + 8*re.numMatches)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCString(s)

	var matches []string

	re.findSubmatch(&alloc, cs, func(match []int) bool {
		matches = append(matches, matchedString(s, match))
		return true
	})

	return matches
}

// FindStringSubmatchIndex returns a slice holding the index pairs
// identifying the leftmost match of the regular expression in s and the
// matches, if any, of its subexpressions, as defined by the 'Submatch' and
// 'Index' descriptions in the package comment.
// A return value of nil indicates no match.
func (re *Regexp) FindStringSubmatchIndex(s string) []int {
	alloc := re.abi.startOperation(len(s) + 8*re.numMatches)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCString(s)

	var matches []int

	re.findSubmatch(&alloc, cs, func(match []int) bool {
		matches = append(matches, match...)
		return true
	})

	res := matches
	runtime.KeepAlive(s)
	return res
}

func (re *Regexp) findSubmatch(alloc *allocation, cs cString, deliver func(match []int) bool) {
	numGroups := re.numMatches
	matchArr := alloc.newCStringArray(numGroups)
	defer matchArr.free()

	if !match(re, cs, matchArr.ptr, uint32(numGroups)) {
		return
	}

	readMatches(alloc, cs, matchArr.ptr, numGroups, deliver)

	runtime.KeepAlive(matchArr)

	runtime.KeepAlive(re) // don't allow finalizer to run during method
}

// Longest makes future searches prefer the leftmost-longest match.
// That is, when matching against text, the regexp returns a match that
// begins as early as possible in the input (leftmost), and among those
// it chooses a match that is as long as possible.
// This method modifies the Regexp and may not be called concurrently
// with any other methods.
func (re *Regexp) Longest() {
	alloc := re.abi.startOperation(len(re.expr) + 2)
	defer re.abi.endOperation(alloc)

	if re.opts.Longest {
		return
	}

	// longest is not a mutable option in re2 so we must release and recompile.
	deleteRE(re.abi, re.ptr)

	cs := alloc.newCString(re.expr)
	newOpts := re.opts
	newOpts.Longest = true
	re.ptr = newRE(re.abi, cs, newOpts)
}

// NumSubexp returns the number of parenthesized subexpressions in this Regexp.
func (re *Regexp) NumSubexp() int {
	return re.numMatches - 1
}

// Split slices s into substrings separated by the expression and returns a slice of
// the substrings between those expression matches.
//
// The slice returned by this method consists of all the substrings of s
// not contained in the slice returned by FindAllString. When called on an expression
// that contains no metacharacters, it is equivalent to strings.SplitN.
//
// Example:
//
//	s := regexp.MustCompile("a*").Split("abaabaccadaaae", 5)
//	// s: ["", "b", "b", "c", "cadaaae"]
//
// The count determines the number of substrings to return:
//
//	n > 0: at most n substrings; the last substring will be the unsplit remainder.
//	n == 0: the result is nil (zero substrings)
//	n < 0: all substrings
func (re *Regexp) Split(s string, n int) []string {
	// Copied as is from
	// https://github.com/golang/go/blob/78472603c6bac7a52d42d565558b9c0cb12c3f9a/src/regexp/regexp.go#L1253
	// The logic in this function is only for taking match indexes to split the string, regex itself
	// delegates to our implementation.

	if n == 0 {
		return nil
	}

	if len(re.expr) > 0 && len(s) == 0 {
		return []string{""}
	}

	matches := re.FindAllStringIndex(s, n)
	strings := make([]string, 0, len(matches))

	beg := 0
	end := 0
	for _, match := range matches {
		if n > 0 && len(strings) >= n-1 {
			break
		}

		end = match[0]
		if match[1] != 0 {
			strings = append(strings, s[beg:end])
		}
		beg = match[1]
	}

	if end != len(s) {
		strings = append(strings, s[beg:])
	}

	return strings
}

// SubexpNames returns the names of the parenthesized subexpressions
// in this Regexp. The name for the first sub-expression is names[1],
// so that if m is a match slice, the name for m[i] is SubexpNames()[i].
// Since the Regexp as a whole cannot be named, names[0] is always
// the empty string. The slice should not be modified.
func (re *Regexp) SubexpNames() []string {
	re.groupNamesOnce.Do(func() {
		re.groupNames = subexpNames(re.abi, re.ptr, re.numMatches)
	})
	return re.groupNames
}

// SubexpIndex returns the index of the first subexpression with the given name,
// or -1 if there is no subexpression with that name.
//
// Note that multiple subexpressions can be written using the same name, as in
// (?P<bob>a+)(?P<bob>b+), which declares two subexpressions named "bob".
// In this case, SubexpIndex returns the index of the leftmost such subexpression
// in the regular expression.
func (re *Regexp) SubexpIndex(name string) int {
	if name != "" {
		for i, s := range re.SubexpNames() {
			if name == s {
				return i
			}
		}
	}
	return -1
}

// Match reports whether the byte slice b
// contains any match of the regular expression re.
func (re *Regexp) Match(b []byte) bool {
	alloc := re.abi.startOperation(len(b))
	defer re.abi.endOperation(alloc)

	cs := alloc.newCStringFromBytes(b)
	res := match(re, cs, nilWasmPtr, 0)
	runtime.KeepAlive(b)

	runtime.KeepAlive(re) // don't allow finalizer to run during method

	return res
}

// MatchString reports whether the string s
// contains any match of the regular expression re.
func (re *Regexp) MatchString(s string) bool {
	alloc := re.abi.startOperation(len(s))
	defer re.abi.endOperation(alloc)

	cs := alloc.newCString(s)
	res := match(re, cs, nilWasmPtr, 0)
	runtime.KeepAlive(s)

	runtime.KeepAlive(re) // don't allow finalizer to run during method

	return res
}

func (re *Regexp) release() {
	if !atomic.CompareAndSwapUint32(&re.released, 0, 1) {
		return
	}
	release(re)
}

func Release(re *Regexp) {
	re.release()
}

// ReplaceAll returns a copy of src, replacing matches of the Regexp
// with the replacement text repl. Inside repl, $ signs are interpreted as
// in Expand, so for instance $1 represents the text of the first submatch.
func (re *Regexp) ReplaceAll(src, repl []byte) []byte {
	alloc := re.abi.startOperation(len(src) + 8*re.numMatches + 8)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCStringFromBytes(src)

	n := 2
	if bytes.IndexByte(repl, '$') >= 0 {
		n = re.numMatches
	}

	srepl := ""
	b := re.replaceAll(&alloc, src, "", cs, n, func(dst []byte, m []int) []byte {
		if len(srepl) != len(repl) {
			srepl = string(repl)
		}
		return re.expand(dst, srepl, src, "", m)
	})
	return b
}

// ReplaceAllFunc returns a copy of src in which all matches of the
// [Regexp] have been replaced by the return value of function repl applied
// to the matched byte slice. The replacement returned by repl is substituted
// directly, without using [Regexp.Expand].
func (re *Regexp) ReplaceAllFunc(src []byte, repl func([]byte) []byte) []byte {
	alloc := re.abi.startOperation(len(src) + 8*re.numMatches + 8)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCStringFromBytes(src)

	return re.replaceAll(&alloc, src, "", cs, 2, func(dst []byte, m []int) []byte {
		return append(dst, repl(src[m[0]:m[1]])...)
	})
}

// ReplaceAllLiteral returns a copy of src, replacing matches of the Regexp
// with the replacement bytes repl. The replacement repl is substituted directly,
// without using Expand.
func (re *Regexp) ReplaceAllLiteral(src, repl []byte) []byte {
	alloc := re.abi.startOperation(len(src) + 8*re.numMatches + 8)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCStringFromBytes(src)

	return re.replaceAll(&alloc, src, "", cs, 2, func(dst []byte, m []int) []byte {
		return append(dst, repl...)
	})
}

// ReplaceAllLiteralString returns a copy of src, replacing matches of the Regexp
// with the replacement string repl. The replacement repl is substituted directly,
// without using Expand.
func (re *Regexp) ReplaceAllLiteralString(src, repl string) string {
	alloc := re.abi.startOperation(len(src) + 8*re.numMatches + 8)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCString(src)

	b := re.replaceAll(&alloc, nil, src, cs, 2, func(dst []byte, m []int) []byte {
		return append(dst, repl...)
	})

	return string(b)
}

// ReplaceAllString returns a copy of src, replacing matches of the Regexp
// with the replacement string repl. Inside repl, $ signs are interpreted as
// in Expand, so for instance $1 represents the text of the first submatch.
func (re *Regexp) ReplaceAllString(src, repl string) string {
	alloc := re.abi.startOperation(len(src) + 8*re.numMatches + 8)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCString(src)

	n := 2
	if strings.Contains(repl, "$") {
		n = re.numMatches
	}

	b := re.replaceAll(&alloc, nil, src, cs, n, func(dst []byte, m []int) []byte {
		return re.expand(dst, repl, nil, src, m)
	})

	return string(b)
}

// ReplaceAllStringFunc returns a copy of src in which all matches of the
// [Regexp] have been replaced by the return value of function repl applied
// to the matched substring. The replacement returned by repl is substituted
// directly, without using [Regexp.Expand].
func (re *Regexp) ReplaceAllStringFunc(src string, repl func(string) string) string {
	alloc := re.abi.startOperation(len(src) + 8*re.numMatches + 8)
	defer re.abi.endOperation(alloc)

	cs := alloc.newCString(src)

	b := re.replaceAll(&alloc, nil, src, cs, 2, func(dst []byte, m []int) []byte {
		return append(dst, repl(src[m[0]:m[1]])...)
	})

	return string(b)
}

func (re *Regexp) replaceAll(alloc *allocation, bsrc []byte, src string, cs cString, nmatch int, repl func(dst []byte, m []int) []byte) []byte {
	lastMatchEnd := 0
	var buf []byte

	re.findAllSubmatch(alloc, bsrc, src, cs, nmatch, -1, func(a []int) {
		// Copy the unmatched characters before this match.
		if bsrc != nil {
			buf = append(buf, bsrc[lastMatchEnd:a[0]]...)
		} else {
			buf = append(buf, src[lastMatchEnd:a[0]]...)
		}

		if a[1] > lastMatchEnd || a[0] == 0 {
			buf = repl(buf, a)
		}
		lastMatchEnd = a[1]
	})

	if bsrc != nil {
		buf = append(buf, bsrc[lastMatchEnd:]...)
	} else {
		buf = append(buf, src[lastMatchEnd:]...)
	}

	return buf
}

// String returns the source text used to compile the regular expression.
func (re *Regexp) String() string {
	return re.expr
}

func subexpNames(abi *libre2ABI, rePtr wasmPtr, numMatches int) []string {
	res := make([]string, numMatches)

	iter := namedGroupsIter(abi, rePtr)
	defer namedGroupsIterDelete(abi, iter)

	for {
		name, index, ok := namedGroupsIterNext(abi, iter)
		if !ok {
			break
		}
		res[index] = name
	}

	return res
}

// extract returns the name from a leading "name" or "{name}" in str.
// (The $ has already been removed by the caller.)
// If it is a number, extract returns num set to that number; otherwise num = -1.
// Copied as is from
// https://github.com/golang/go/blob/0fd7be7ee5f36215b5d6b8f23f35d60bf749805a/src/regexp/regexp.go#L981
func extract(str string) (name string, num int, rest string, ok bool) {
	if str == "" {
		return
	}
	brace := false
	if str[0] == '{' {
		brace = true
		str = str[1:]
	}
	i := 0
	for i < len(str) {
		rune, size := utf8.DecodeRuneInString(str[i:])
		if !unicode.IsLetter(rune) && !unicode.IsDigit(rune) && rune != '_' {
			break
		}
		i += size
	}
	if i == 0 {
		// empty name is not okay
		return
	}
	name = str[:i]
	if brace {
		if i >= len(str) || str[i] != '}' {
			// missing closing brace
			return
		}
		i++
	}

	// Parse number.
	num = 0
	for i := 0; i < len(name); i++ {
		if name[i] < '0' || '9' < name[i] || num >= 1e8 {
			num = -1
			break
		}
		num = num*10 + int(name[i]) - '0'
	}
	// Disallow leading zeros.
	if name[0] == '0' && len(name) > 1 {
		num = -1
	}

	rest = str[i:]
	ok = true
	return
}

func matchedBytes(s []byte, match []int) []byte {
	if len(match) < 2 || match[0] == -1 {
		return nil
	}
	return s[match[0]:match[1]:match[1]]
}

func matchedString(s string, match []int) string {
	if len(match) < 2 || match[0] == -1 {
		return ""
	}
	return s[match[0]:match[1]]
}

func nextPos(bsrc []byte, src string, pos int, matchEnd int) int {
	// Advance past the match; always advance at least one character.
	var width int
	if bsrc != nil {
		_, width = utf8.DecodeRune(bsrc[pos:])
	} else {
		_, width = utf8.DecodeRuneInString(src[pos:])
	}

	if pos+width > matchEnd {
		return pos + width
	} else if pos+1 > matchEnd {
		// This clause is only needed at the end of the input
		// string. In that case, DecodeRuneInString returns width=0.
		return pos + 1
	} else {
		return matchEnd
	}
}

func QuoteForError(s string) string {
	if strconv.CanBackquote(s) {
		return "`" + s + "`"
	}
	return strconv.Quote(s)
}
