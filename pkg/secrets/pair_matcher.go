package secrets

import (
	"fmt"
	"math/bits"
	"slices"
	"strings"
)

var asciiFoldTable = func() [256]byte {
	var table [256]byte
	for i := range table {
		table[i] = byte(i)
		if table[i] >= 'A' && table[i] <= 'Z' {
			table[i] += 'a' - 'A'
		}
	}
	return table
}()

func foldASCII(value byte) byte {
	return asciiFoldTable[value]
}

type catalogPairChoice struct {
	pair   uint16
	offset uint16
}

type catalogPairCertificate struct {
	value       string
	output      uint16
	offset      uint16
	outputCount uint16
}

type catalogCollisionGuard struct {
	mask   uint32
	offset int32
}

const catalogPairCollisionGroup = uint16(1 << 15)

// Custom rules derive at most one keyword each. Keep their pair inventory
// compact, without adding a sparse-lookup branch to the native per-byte scan.
const maxCompactCatalogPairs = 16

type catalogPairEntry struct {
	pair  uint16
	entry uint16
}

type catalogPairMatcher struct {
	entryByPair            *[1 << 14]uint16
	compactPairs           []catalogPairEntry
	singleCertificates     []catalogPairCertificate
	collisionGroups        [][]catalogPairCertificate
	collisionGuards        []catalogCollisionGuard
	singleCertificateCount uint8
	duplicateOutputs       []uint16
	// Single-byte keyword outputs are unused by the normal byte-pair path.
	singleOutputs  *[128][]uint16
	hasSingles     bool // Retain the hot-path discriminator rather than testing the rare table pointer.
	unicodeKeyword []unicodeKeyword
}

func newCatalogPairMatcher(keywords []matcherKeyword) (*catalogPairMatcher, error) {
	type source struct {
		value   string
		outputs []uint16
		choices []catalogPairChoice
	}
	matcher := &catalogPairMatcher{}
	frequency := make(map[uint16]int)
	var quotientFrequency [32]int
	sourceByValue := make(map[string]int, len(keywords))
	sources := make([]source, 0, len(keywords))
	for _, item := range keywords {
		value := foldKeyword(item.value)
		if value == "" {
			continue
		}
		if !isASCIIKeyword(value) {
			matcher.unicodeKeyword = append(matcher.unicodeKeyword, unicodeKeyword{value: value, rule: item.rule})
			continue
		}
		if len(value) == 1 {
			quotientFrequency[value[0]&31]++
			if matcher.singleOutputs == nil {
				matcher.singleOutputs = new([128][]uint16)
			}
			matcher.hasSingles = true
			matcher.singleOutputs[value[0]] = appendUniqueOutput(matcher.singleOutputs[value[0]], item.rule)
			continue
		}
		if sourceIndex, exists := sourceByValue[value]; exists {
			sources[sourceIndex].outputs = appendUniqueOutput(sources[sourceIndex].outputs, item.rule)
			continue
		}
		for i := range len(value) {
			quotientFrequency[value[i]&31]++
		}
		choices := catalogPairChoices(value)
		for _, choice := range choices {
			frequency[choice.pair]++
		}
		sourceByValue[value] = len(sources)
		sources = append(sources, source{value: value, outputs: []uint16{item.rule}, choices: choices})
	}
	if len(sources) >= 1<<15 {
		return nil, fmt.Errorf("secret pair matcher exceeds %d certificates", 1<<15-1)
	}

	byPair := make(map[uint16][]catalogPairCertificate)
	for _, source := range sources {
		slices.SortFunc(source.choices, func(left, right catalogPairChoice) int {
			if frequency[left.pair] != frequency[right.pair] {
				return frequency[left.pair] - frequency[right.pair]
			}
			if left.pair != right.pair {
				return int(left.pair) - int(right.pair)
			}
			return int(left.offset) - int(right.offset)
		})
		anchor := source.choices[0]
		byPair[anchor.pair] = append(byPair[anchor.pair], matcher.newPairCertificate(source.value, source.outputs, anchor.offset))
	}

	pairs := make([]uint16, 0, len(byPair))
	for pair := range byPair {
		pairs = append(pairs, pair)
	}
	slices.SortFunc(pairs, func(left, right uint16) int {
		leftCertificates := byPair[left]
		rightCertificates := byPair[right]
		leftDirect := isCatalogDirectPair(leftCertificates)
		rightDirect := isCatalogDirectPair(rightCertificates)
		switch {
		case leftDirect && rightDirect:
			if len(leftCertificates[0].value) != len(rightCertificates[0].value) {
				return len(leftCertificates[0].value) - len(rightCertificates[0].value)
			}
		case leftDirect:
			return -1
		case rightDirect:
			return 1
		}
		return int(left) - int(right)
	})
	directCount := 0
	for _, pair := range pairs {
		if directCount == 127 || !isCatalogDirectPair(byPair[pair]) {
			break
		}
		directCount++
	}
	matcher.singleCertificates = make([]catalogPairCertificate, directCount)
	if len(pairs) <= maxCompactCatalogPairs {
		matcher.compactPairs = make([]catalogPairEntry, 0, len(pairs))
	} else {
		matcher.entryByPair = new([1 << 14]uint16)
	}
	for _, pair := range pairs {
		certificates := byPair[pair]
		var entry uint16
		if isCatalogDirectPair(certificates) && matcher.singleCertificateCount < 127 {
			index := matcher.singleCertificateCount
			matcher.singleCertificates[index] = certificates[0]
			matcher.singleCertificateCount++
			entry = uint16(matcher.singleCertificateCount)
		} else {
			if len(matcher.collisionGroups) == int(catalogPairCollisionGroup) {
				return nil, fmt.Errorf("secret pair matcher exceeds %d collision groups", catalogPairCollisionGroup)
			}
			entry = catalogPairCollisionGroup | uint16(len(matcher.collisionGroups))
			matcher.collisionGroups = append(matcher.collisionGroups, certificates)
			matcher.collisionGuards = append(matcher.collisionGuards, newCatalogCollisionGuard(certificates, quotientFrequency))
		}
		matcher.setPairEntry(byte(pair>>8), byte(pair), entry)
	}
	slices.SortFunc(matcher.compactPairs, func(left, right catalogPairEntry) int {
		return int(left.pair) - int(right.pair)
	})
	return matcher, nil
}

func isCatalogDirectPair(certificates []catalogPairCertificate) bool {
	return len(certificates) == 1 && certificates[0].outputCount == 0
}

func (m *catalogPairMatcher) newPairCertificate(value string, outputs []uint16, offset uint16) catalogPairCertificate {
	certificate := catalogPairCertificate{value: value, output: outputs[0], offset: offset}
	if len(outputs) == 1 {
		return certificate
	}
	certificate.output = uint16(len(m.duplicateOutputs))
	certificate.outputCount = uint16(len(outputs))
	m.duplicateOutputs = append(m.duplicateOutputs, outputs...)
	return certificate
}

// emitCertificate adds the rules behind a verified keyword occurrence starting at start and,
// when hits is non-nil, records the occurrence for keyword-guided evaluation.
func (m *catalogPairMatcher) emitCertificate(value string, certificate catalogPairCertificate, start int, candidates *ruleSet, hits *keywordHits) {
	end := start + len(certificate.value)
	if certificate.outputCount == 0 {
		candidates.add(certificate.output)
		if hits != nil {
			hits.add(value, certificate.output, start, end)
		}
		return
	}
	first := int(certificate.output)
	for _, output := range m.duplicateOutputs[first : first+int(certificate.outputCount)] {
		candidates.add(output)
		if hits != nil {
			hits.add(value, output, start, end)
		}
	}
}

func (m *catalogPairMatcher) setPairEntry(first, second byte, entry uint16) {
	if m.entryByPair == nil {
		m.compactPairs = append(m.compactPairs, catalogPairEntry{pair: uint16(first)<<7 | uint16(second), entry: entry})
		return
	}
	m.entryByPair[uint16(first)<<7|uint16(second)] = entry
	firstUpper := first >= 'a' && first <= 'z'
	secondUpper := second >= 'a' && second <= 'z'
	if firstUpper {
		m.entryByPair[uint16(first-('a'-'A'))<<7|uint16(second)] = entry
	}
	if secondUpper {
		m.entryByPair[uint16(first)<<7|uint16(second-('a'-'A'))] = entry
	}
	if firstUpper && secondUpper {
		m.entryByPair[uint16(first-('a'-'A'))<<7|uint16(second-('a'-'A'))] = entry
	}
}

func (m *catalogPairMatcher) compactPairEntry(first, second byte) uint16 {
	pair := uint16(foldASCII(first&0x7f))<<7 | uint16(foldASCII(second&0x7f))
	low, high := 0, len(m.compactPairs)
	for low < high {
		middle := int(uint(low+high) >> 1)
		candidate := m.compactPairs[middle]
		switch {
		case candidate.pair < pair:
			low = middle + 1
		case candidate.pair > pair:
			high = middle
		default:
			return candidate.entry
		}
	}
	return 0
}

func catalogPairChoices(value string) []catalogPairChoice {
	choices := make([]catalogPairChoice, 0, len(value)-1)
	for i := 1; i < len(value); i++ {
		pair := uint16(value[i-1])<<8 | uint16(value[i])
		seen := false
		for _, choice := range choices {
			if choice.pair == pair {
				seen = true
				break
			}
		}
		if !seen {
			choices = append(choices, catalogPairChoice{pair: pair, offset: uint16(i - 1)})
		}
	}
	return choices
}

func newCatalogCollisionGuard(certificates []catalogPairCertificate, quotientFrequency [32]int) catalogCollisionGuard {
	low := -int(certificates[0].offset)
	high := len(certificates[0].value) - int(certificates[0].offset) - 1
	for _, certificate := range certificates[1:] {
		low = max(low, -int(certificate.offset))
		high = min(high, len(certificate.value)-int(certificate.offset)-1)
	}

	bestSupport := 33
	bestWeight := int(^uint(0) >> 1)
	bestDistance := -1
	var best catalogCollisionGuard
	for offset := low; offset <= high; offset++ {
		if offset == 0 || offset == 1 {
			continue
		}
		var mask uint32
		for _, certificate := range certificates {
			value := certificate.value[int(certificate.offset)+offset]
			mask |= uint32(1) << (value & 31)
		}
		support := bits.OnesCount32(mask)
		weight := 0
		for quotient, frequency := range quotientFrequency {
			if mask&(uint32(1)<<quotient) != 0 {
				weight += frequency
			}
		}
		distance := max(-offset, offset-1)
		if support < bestSupport ||
			support == bestSupport && weight < bestWeight ||
			support == bestSupport && weight == bestWeight && distance > bestDistance {
			bestSupport = support
			bestWeight = weight
			bestDistance = distance
			best = catalogCollisionGuard{mask: mask, offset: int32(offset)}
		}
	}
	if best.mask == 0 {
		value := certificates[0].value[certificates[0].offset]
		best = catalogCollisionGuard{mask: uint32(1) << (value & 31)}
	}
	return best
}

func (g catalogCollisionGuard) matches(value string, anchorPosition int) bool {
	position := anchorPosition + int(g.offset)
	return uint(position) < uint(len(value)) && g.mask&(uint32(1)<<(value[position]&31)) != 0
}

func (m *catalogPairMatcher) match(value string, candidates *ruleSet) {
	m.matchHits(value, candidates, nil)
}

// matchHits selects candidate rules and records verified ASCII-keyword byte
// positions, including single-byte keywords. Unicode-keyword inventories retain
// their whole-value fallback; non-ASCII values keep proven ASCII positions.
func (m *catalogPairMatcher) matchHits(value string, candidates *ruleSet, hits *keywordHits) {
	if value == "" {
		return
	}
	if len(m.unicodeKeyword) != 0 {
		hits.invalidate()
		if hits != nil {
			hits.nonASCII = !isASCIIKeyword(value)
		}
		m.matchUnicode(value, candidates)
		return
	}
	first := value[0]
	nonASCII := first
	if m.entryByPair == nil {
		nonASCII = m.matchCompactPairs(value, candidates, hits)
	} else {
		// The 7-bit index avoids a per-byte branch; aliases cannot pass certificate verification, and the accumulated high bit triggers Unicode folding after the scan.
		for i := 1; i < len(value); i++ {
			second := value[i]
			nonASCII |= second
			entry := m.entryByPair[int(first&0x7f)<<7|int(second&0x7f)]
			if entry != 0 {
				anchorPosition := i - 1
				if entry&catalogPairCollisionGroup == 0 {
					certificate := m.singleCertificates[entry-1]
					keyword := certificate.value
					start := anchorPosition - int(certificate.offset)
					if start >= 0 && start+len(keyword) <= len(value) && matchCatalogKeyword(value, start, keyword) {
						m.emitCertificate(value, certificate, start, candidates, hits)
					}
				} else {
					group := entry &^ catalogPairCollisionGroup
					if m.collisionGuards[group].matches(value, anchorPosition) {
						for _, certificate := range m.collisionGroups[group] {
							keyword := certificate.value
							start := anchorPosition - int(certificate.offset)
							if start >= 0 && start+len(keyword) <= len(value) && matchCatalogKeyword(value, start, keyword) {
								m.emitCertificate(value, certificate, start, candidates, hits)
							}
						}
					}
				}
			}
			first = second
		}
	}
	if m.hasSingles {
		// Record the more selective pair keywords first so frequent single-byte
		// keywords cannot consume their hit budget. Overflow remains per rule.
		for i := range len(value) {
			b := foldASCII(value[i])
			if b >= 0x80 {
				continue
			}
			for _, output := range m.singleOutputs[b] {
				candidates.add(output)
				if hits != nil {
					hits.add(value, output, i, i+1)
				}
			}
		}
	}
	if nonASCII >= 0x80 {
		if hits != nil {
			hits.nonASCII = true
		}
		// The byte scan already verified every ASCII keyword occurrence. Only
		// Unicode members of ASCII fold classes can add an unseen occurrence;
		// unrelated Unicode needs neither normalization nor a second pair scan.
		if containsUnicodeASCIIFold(value) {
			if hits != nil {
				hits.asciiFoldAlias = true
			}
			m.matchUnicode(value, candidates)
		}
	}
}

// matchCompactPairs uses the same certificates and guards as the dense scan.
// Inputs may contain non-ASCII bytes; certificate verification rejects aliases
// introduced by the 7-bit lookup, before the caller handles Unicode folding.
func (m *catalogPairMatcher) matchCompactPairs(value string, candidates *ruleSet, hits *keywordHits) byte {
	first := value[0]
	nonASCII := first
	for i := 1; i < len(value); i++ {
		second := value[i]
		nonASCII |= second
		entry := m.compactPairEntry(first, second)
		first = second
		if entry == 0 {
			continue
		}
		anchorPosition := i - 1
		var certificates []catalogPairCertificate
		if entry&catalogPairCollisionGroup == 0 {
			certificates = m.singleCertificates[entry-1 : entry]
		} else {
			group := entry &^ catalogPairCollisionGroup
			if !m.collisionGuards[group].matches(value, anchorPosition) {
				continue
			}
			certificates = m.collisionGroups[group]
		}
		for _, certificate := range certificates {
			start := anchorPosition - int(certificate.offset)
			if start >= 0 && start+len(certificate.value) <= len(value) && matchCatalogKeyword(value, start, certificate.value) {
				m.emitCertificate(value, certificate, start, candidates, hits)
			}
		}
	}
	return nonASCII
}

func (m *catalogPairMatcher) matchASCIIEntry(value string, anchorPosition int, entry uint16, candidates *ruleSet) {
	if entry&catalogPairCollisionGroup == 0 {
		certificate := m.singleCertificates[entry-1]
		keyword := certificate.value
		start := anchorPosition - int(certificate.offset)
		if start >= 0 && start+len(keyword) <= len(value) && value[start:start+len(keyword)] == keyword {
			m.emitCertificate(value, certificate, start, candidates, nil)
		}
		return
	}
	group := entry &^ catalogPairCollisionGroup
	if m.collisionGuards[group].matches(value, anchorPosition) {
		for _, certificate := range m.collisionGroups[group] {
			keyword := certificate.value
			start := anchorPosition - int(certificate.offset)
			if start >= 0 && start+len(keyword) <= len(value) && value[start:start+len(keyword)] == keyword {
				m.emitCertificate(value, certificate, start, candidates, nil)
			}
		}
	}
}

func (m *catalogPairMatcher) matchUnicode(value string, candidates *ruleSet) {
	if len(m.unicodeKeyword) == 0 {
		value = foldASCIIKeywordInput(value)
	} else {
		value = foldKeyword(value)
	}
	for i := 0; i < len(value); {
		for i < len(value) && value[i] >= 0x80 {
			i++
		}
		start := i
		for i < len(value) && value[i] < 0x80 {
			i++
		}
		if start < i {
			m.matchASCII(value[start:i], candidates)
		}
	}
	for _, keyword := range m.unicodeKeyword {
		if strings.Contains(value, keyword.value) {
			candidates.add(keyword.rule)
		}
	}
}

func (m *catalogPairMatcher) matchASCII(value string, candidates *ruleSet) {
	if value == "" {
		return
	}
	if m.entryByPair == nil {
		m.matchCompactPairs(value, candidates, nil)
		if m.hasSingles {
			for i := range len(value) {
				for _, output := range m.singleOutputs[value[i]] {
					candidates.add(output)
				}
			}
		}
		return
	}
	if m.hasSingles {
		m.matchASCIIWithSingles(value, candidates)
		return
	}
	first := value[0]
	for i := 1; i < len(value); i++ {
		second := value[i]
		entry := m.entryByPair[int(first)<<7|int(second)]
		if entry != 0 {
			m.matchASCIIEntry(value, i-1, entry, candidates)
		}
		first = second
	}
}

func (m *catalogPairMatcher) matchASCIIWithSingles(value string, candidates *ruleSet) {
	first := value[0]
	for _, output := range m.singleOutputs[first] {
		candidates.add(output)
	}
	for i := 1; i < len(value); i++ {
		second := value[i]
		for _, output := range m.singleOutputs[second] {
			candidates.add(output)
		}
		entry := m.entryByPair[int(first)<<7|int(second)]
		if entry != 0 {
			m.matchASCIIEntry(value, i-1, entry, candidates)
		}
		first = second
	}
}

func matchCatalogKeyword(value string, start int, keyword string) bool {
	for i := range len(keyword) {
		if foldASCII(value[start+i]) != keyword[i] {
			return false
		}
	}
	return true
}
