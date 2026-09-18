package secrets

import (
	"cmp"
	"slices"
	"sync"
)

// maxKeywordHits bounds the keyword occurrences recorded per value. Overflowed
// rules use proven streaming guidance where available, otherwise whole-value evaluation.
const maxKeywordHits = 256

// keywordHit is one occurrence of an ASCII rule keyword at verified byte offsets.
type keywordHit struct {
	start uint32
	end   uint32
	rule  uint16
}

type keywordHitBuffer [maxKeywordHits]keywordHit

var keywordHitBufferPool = sync.Pool{New: func() any { return new(keywordHitBuffer) }}

// keywordHits records where rule-set keywords occur so bounded and anchored
// regex strategies can skip the parts of a value that provably cannot match.
//
// The buffer is attached lazily on the first occurrence, so values without keywords never pay
// for it. Owners either point buffer at storage they embed (BatchDetector) or call release
// after each detection to return the pooled buffer.
type keywordHits struct {
	buffer *keywordHitBuffer
	count  int
	// invalid marks matcher paths that do not provide positions.
	invalid bool
	// nonASCII requires a proven ASCII start and UTF-8 byte bounds before using
	// the byte scan's keyword positions for anchored evaluation.
	nonASCII bool
	// asciiFoldAlias means the original value contains a non-ASCII rune that
	// can participate in an ASCII regexp simple-fold class.
	asciiFoldAlias bool
	// ASCII presence is computed lazily, once per non-ASCII value, only when a
	// candidate proves that every match must consume a strict-ASCII rune.
	asciiChecked bool
	asciiPresent bool
	// enabled and boundary point at immutable rule-set-owned sets so reset stays cheap. Occurrences of
	// boundary rules (whose matches start with \b before the keyword) glued to a word character
	// are dropped because detectAnchored would skip them anyway.
	// Rules outside enabled still select candidates, but do not consume buffer space.
	enabled  *ruleSet
	boundary *ruleSet
	// overflowed holds the rules with occurrences that did not fit in the buffer.
	// Recorded occurrences of every other rule are complete.
	overflowed ruleSet
}

func (h *keywordHits) reset(enabled, boundary *ruleSet) {
	h.count = 0
	h.invalid = false
	h.nonASCII = false
	h.asciiFoldAlias = false
	h.asciiChecked = false
	h.asciiPresent = false
	h.enabled = enabled
	h.boundary = boundary
	h.overflowed = ruleSet{}
}

// release returns a pooled buffer. It must not be called on hits whose buffer is embedded.
func (h *keywordHits) release() {
	if h.buffer != nil {
		keywordHitBufferPool.Put(h.buffer)
		h.buffer = nil
	}
}

func (h *keywordHits) invalidate() {
	if h != nil {
		h.invalid = true
	}
}

// add records that a keyword of rule occupies value[start:end].
func (h *keywordHits) add(value string, rule uint16, start, end int) {
	if !h.enabled.has(rule) {
		return
	}
	if h.boundary.has(rule) && !asciiWordBoundary(value, start) {
		return
	}
	if h.count == maxKeywordHits {
		h.overflowed.add(rule)
		return
	}
	if h.buffer == nil {
		h.buffer = keywordHitBufferPool.Get().(*keywordHitBuffer)
	}
	h.buffer[h.count] = keywordHit{start: uint32(start), end: uint32(end), rule: rule}
	h.count++
}

// guides reports whether recorded ASCII-keyword occurrences are complete for a
// rule. Non-ASCII input additionally needs the plan's literal-start proof.
// Rules outside enabled or with overflowed hits cannot use the recorded offsets.
func (h *keywordHits) guides(rule uint16) bool {
	return h != nil && !h.invalid && h.enabled.has(rule) && !h.overflowed.has(rule)
}

// sort orders the hits by rule and then by keyword start so each rule owns a contiguous range.
func (h *keywordHits) sort() {
	if h.count < 2 {
		return
	}
	if h.count < 32 {
		slices.SortFunc(h.buffer[:h.count], compareKeywordHits)
		return
	}
	// A dense run for one keyword is commonly already in final order.
	// Avoid regrouping and copying it only to discover that again per rule.
	if slices.IsSortedFunc(h.buffer[:h.count], compareKeywordHits) {
		return
	}
	// Group bounded rule IDs in linear time. Occurrences for a single keyword
	// are already ordered by the byte scan; only genuinely disordered groups
	// need a comparison sort (different keywords may have different anchors).
	var offsets [catalogRuleWords*64 + 1]uint16
	for _, hit := range h.buffer[:h.count] {
		offsets[hit.rule+1]++
	}
	for i := 1; i < len(offsets); i++ {
		offsets[i] += offsets[i-1]
	}
	next := offsets
	var grouped keywordHitBuffer
	for _, hit := range h.buffer[:h.count] {
		grouped[next[hit.rule]] = hit
		next[hit.rule]++
	}
	copy(h.buffer[:h.count], grouped[:h.count])
	for rule := range catalogRuleWords * 64 {
		group := h.buffer[offsets[rule]:offsets[rule+1]]
		if !slices.IsSortedFunc(group, compareKeywordHits) {
			slices.SortFunc(group, compareKeywordHits)
		}
	}
}

func compareKeywordHits(left, right keywordHit) int {
	return cmp.Or(
		cmp.Compare(left.rule, right.rule),
		cmp.Compare(left.start, right.start),
		cmp.Compare(left.end, right.end),
	)
}

// rangeFor returns the sorted hit range [from, to) recorded for rule.
func (h *keywordHits) rangeFor(rule uint16) (int, int) {
	from := 0
	for from < h.count && h.buffer[from].rule < rule {
		from++
	}
	to := from
	for to < h.count && h.buffer[to].rule == rule {
		to++
	}
	return from, to
}

// sortRangeByEnd reorders hits[from:to] by keyword end, the order in which their windows
// start, so overlapping windows can be merged in a single sweep.
func (h *keywordHits) sortRangeByEnd(from, to int) {
	slices.SortFunc(h.buffer[from:to], func(left, right keywordHit) int {
		return cmp.Compare(left.end, right.end)
	})
}
