package secrets

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
	"regexp"
	"regexp/syntax"
	"runtime"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"
)

// contextValidation is native-only. A rejection interval is a promise that no
// later candidate starting before rejectUntil can pass this context predicate.
// The unfiltered reference ignores this optimization; public unique-ID scans
// may skip those candidates without changing acceptance.
type contextValidation struct {
	accepted    bool
	rejectUntil int
}

type catalogRuleSpec struct {
	ID          string
	Regex       string
	Keywords    []string
	Entropy     float64
	SecretGroup int
	// Source and Description document the rule's baseline and supported scope.
	// Custom rules do not need catalog provenance.
	Source      string
	Description string
	Validate    func(string) bool
	// ValidateContext is native-only and receives the original, unsliced value.
	// Match offsets identify the complete regex match, before newline trimming.
	ValidateContext func(value string, matchStart, matchEnd int, secret string) contextValidation
}

type compiledRule struct {
	id    string
	regex *regexp.Regexp
	// Nonzero starts use either anchored or contextAnchored, never both.
	// The latter consumes the actual preceding rune for left-context assertions.
	anchored          *regexp.Regexp
	contextAnchored   *regexp.Regexp
	literalRun        *literalRunMatcher
	plan              ruleEvaluationPlan
	filter            *rejectionFilter
	fallbackFilter    *rejectionFilter // unanchored rejection when keyword positions are unavailable
	run               asciiRun
	asciiRequirement  asciiRequirement
	punctuation       asciiSet // every byte is mandatory; empty disables this rejection gate
	punctuationChoice asciiSet // at least one of these bytes is mandatory
	entropy           float64
	secretGroup       int
	validate          func(string) bool
	validateContext   func(value string, matchStart, matchEnd int, secret string) contextValidation
	// streamKeywords preserves proven guidance when occurrence storage overflows.
	streamKeywords           []string
	literalAlternatives      []literalAlternative
	exactLiteralAlternatives []literalAlternative
}

// compiledRuleSet owns immutable evaluation plans and a matcher. A compiler's
// selected native set is shared; each policy owns only its custom set.
type compiledRuleSet struct {
	rules       []compiledRule
	pairMatcher *catalogPairMatcher
	keywordless ruleSet
	boundary    ruleSet // anchored rules whose keyword must follow a word boundary
	guided      ruleSet // only these rules consume recorded keyword positions
	all         ruleSet
}

type compiledCatalog struct {
	compiledRuleSet
	byID map[string]uint16
}

type evaluationMode uint8

const (
	allRuleMatches evaluationMode = iota
	uniqueRuleMatches
)

// Share immutable expressions, not acceptance plans, within one compilation.
func newCatalogRegexpCompiler() func(string) (*regexp.Regexp, error) {
	var mu sync.Mutex
	regexps := make(map[string]func() (*regexp.Regexp, error))
	return func(expression string) (*regexp.Regexp, error) {
		mu.Lock()
		load, ok := regexps[expression]
		if !ok {
			load = sync.OnceValues(func() (*regexp.Regexp, error) {
				return regexp.Compile(expression)
			})
			regexps[expression] = load
		}
		mu.Unlock()
		return load()
	}
}

// ruleCompiler owns construction-only state for one expression. Catalog workers
// each own one compiler at a time; only immutable regexps, plans and filters escape.
// Acceptance predicates and capture selection remain on each compiledRule.
type ruleCompiler struct {
	expression               string
	parsed                   *syntax.Regexp
	regex                    *regexp.Regexp
	compileRegexp            func(string) (*regexp.Regexp, error)
	anchored                 *regexp.Regexp
	contextAnchored          *regexp.Regexp
	literalRun               *literalRunMatcher
	leftContext              bool
	prefix                   string
	literalPrefix            bool
	run                      asciiRun
	asciiRequirement         asciiRequirement
	punctuation              asciiSet
	punctuationChoice        asciiSet
	filters                  [2]*rejectionFilter
	filtersBuilt             [2]bool
	plans                    []cachedRulePlan
	filterProgram            *syntax.Prog
	literalAlternatives      []literalAlternative
	exactLiteralAlternatives []literalAlternative
	literalsBuilt            bool
}

type cachedRulePlan struct {
	keywords      []string
	literalPrefix bool
	plan          ruleEvaluationPlan
}

func newRuleCompiler(expression string, compileRegexp func(string) (*regexp.Regexp, error)) (*ruleCompiler, error) {
	re, err := compileRegexp(expression)
	if err != nil {
		return nil, err
	}
	parsed, err := syntax.Parse(expression, syntax.Perl)
	if err != nil {
		return nil, err
	}
	prefix, _ := re.LiteralPrefix()
	compiler := &ruleCompiler{
		expression:       expression,
		parsed:           parsed,
		regex:            re,
		compileRegexp:    compileRegexp,
		leftContext:      requiresLeftContext(parsed),
		literalRun:       newLiteralRunMatcher(parsed),
		prefix:           prefix,
		literalPrefix:    prefix != "",
		run:              requiredASCIIRun(parsed),
		asciiRequirement: requiredASCII(parsed),
		punctuation:      requiredPunctuation(parsed),
	}
	compiler.punctuationChoice = requiredPunctuationChoice(parsed, compiler.punctuation)
	if compiler.prefix == "" {
		compiler.prefix = exactASCIIStart(parsed)
	}
	if !isASCIIKeyword(compiler.prefix) {
		compiler.prefix = ""
	}
	return compiler, nil
}

func (c *ruleCompiler) plan(keywords, configured []string) ruleEvaluationPlan {
	literalPrefix := c.literalPrefix && slices.Equal(keywords, configured)
	for _, cached := range c.plans {
		if cached.literalPrefix == literalPrefix && slices.Equal(cached.keywords, keywords) {
			return cached.plan
		}
	}
	plan := analyzeRule(c.parsed, keywords)
	if literalPrefix {
		// The engine already skips literal-prefix occurrences efficiently.
		// Derived keywords retain guidance: the configured ones were not proven.
		plan.strategy = strategyFullScan
	}
	c.plans = append(c.plans, cachedRulePlan{keywords, literalPrefix, plan})
	return plan
}

func (c *ruleCompiler) filter(anchored bool) *rejectionFilter {
	index := 0
	if anchored {
		index = 1
	}
	if !c.filtersBuilt[index] {
		if !c.filtersBuilt[0] && !c.filtersBuilt[1] {
			c.filterProgram, _ = syntax.Compile(relaxedRegexp(c.parsed).Simplify())
		}
		filter, _ := newRejectionFilterFromProgram(c.filterProgram, anchored, rejectionNFACap, rejectionDFACap)
		if filter != nil && !anchored {
			filter.prefix = c.prefix
		}
		c.filters[index] = filter
		c.filtersBuilt[index] = true
	}
	return c.filters[index]
}

func compileNativeCatalog() (*compiledCatalog, error) {
	return compileCatalog(nativeRuleSpecs)
}

func compileCatalog(specs []catalogRuleSpec) (*compiledCatalog, error) {
	if len(specs) == 0 {
		return &compiledCatalog{}, nil
	}
	if len(specs) > catalogRuleWords*64 {
		return nil, fmt.Errorf("secret catalog has %d rules; maximum is %d", len(specs), catalogRuleWords*64)
	}
	catalog := &compiledCatalog{
		compiledRuleSet: compiledRuleSet{rules: make([]compiledRule, len(specs))},
		byID:            make(map[string]uint16, len(specs)),
	}
	ruleKeywords := make([][]string, len(specs))
	expressionGroups := make(map[string]int, len(specs))
	var groups [][]int
	for i, spec := range specs {
		index := uint16(i)
		if _, exists := catalog.byID[spec.ID]; exists {
			return nil, fmt.Errorf("duplicate secret catalog rule %q", spec.ID)
		}
		catalog.byID[spec.ID] = index
		group, exists := expressionGroups[spec.Regex]
		if !exists {
			group = len(groups)
			expressionGroups[spec.Regex] = group
			groups = append(groups, nil)
		}
		groups[group] = append(groups[group], i)
	}

	compileRegexp := newCatalogRegexpCompiler()

	compileErrors := make([]error, len(specs))
	// DFA construction can retain large temporary state sets. Bound both active
	// compilations and goroutine stacks instead of launching one per rule.
	workers := min(len(groups), runtime.GOMAXPROCS(0), 8)
	jobs := make(chan []int)
	var compileGroup sync.WaitGroup
	for range workers {
		compileGroup.Go(func() {
			for indices := range jobs {
				compiler, err := newRuleCompiler(specs[indices[0]].Regex, compileRegexp)
				for _, i := range indices {
					if err != nil {
						compileErrors[i] = err
						continue
					}
					ruleKeywords[i] = effectiveKeywordsForRegexp(compiler.parsed, specs[i].Keywords)
					catalog.rules[i], compileErrors[i] = compiler.compile(specs[i], ruleKeywords[i])
				}
			}
		})
	}
	for _, indices := range groups {
		jobs <- indices
	}
	close(jobs)
	compileGroup.Wait()

	matcherKeywords := make([]matcherKeyword, 0, len(specs)*2)
	for i := range catalog.rules {
		spec := specs[i]
		if compileErrors[i] != nil {
			return nil, fmt.Errorf("compile secret catalog rule %q: %w", spec.ID, compileErrors[i])
		}
		matcherKeywords = catalog.addRuleKeywords(uint16(i), ruleKeywords[i], matcherKeywords)
	}
	pairMatcher, err := newCatalogPairMatcher(matcherKeywords)
	if err != nil {
		return nil, err
	}
	catalog.pairMatcher = pairMatcher
	return catalog, nil
}

func compileRuleSpec(spec catalogRuleSpec, keywords []string) (compiledRule, error) {
	compiler, err := newRuleCompiler(spec.Regex, regexp.Compile)
	if err != nil {
		return compiledRule{}, err
	}
	return compiler.compile(spec, keywords)
}

func (c *ruleCompiler) compile(spec catalogRuleSpec, keywords []string) (compiledRule, error) {
	rule := compiledRule{
		id:              spec.ID,
		entropy:         spec.Entropy,
		secretGroup:     spec.SecretGroup,
		validate:        spec.Validate,
		validateContext: spec.ValidateContext,
	}
	if spec.SecretGroup < 0 || spec.SecretGroup > c.regex.NumSubexp() {
		return compiledRule{}, errors.New("invalid secret capture group")
	}
	rule.regex = c.regex
	rule.plan = c.plan(keywords, spec.Keywords)
	rule.run = c.run
	rule.asciiRequirement = c.asciiRequirement
	rule.punctuation = c.punctuation
	rule.punctuationChoice = c.punctuationChoice
	rule.filter = c.filter(rule.plan.strategy == strategyAnchored)
	if rule.plan.strategy == strategyAnchored {
		rule.fallbackFilter = c.filter(false)
		rule.literalRun = c.literalRun
	}
	if rule.filter == nil && c.prefix == "" {
		if !c.literalsBuilt {
			c.literalAlternatives = requiredLiteralAlternatives(c.parsed)
			c.exactLiteralAlternatives = requiredExactLiteralAlternatives(c.parsed)
			c.literalsBuilt = true
		}
		rule.literalAlternatives = c.literalAlternatives
		rule.exactLiteralAlternatives = c.exactLiteralAlternatives
	}
	if rule.plan.strategy == strategyAnchored {
		var err error
		// A context-consuming program handles every nonzero start. At input start
		// the original regexp sees the complete value, so a second ^(?:regex)
		// program is redundant even with capture-dependent/context acceptance.
		if c.literalRun == nil && !c.leftContext {
			if c.anchored == nil {
				c.anchored, err = c.compileRegexp("^(?:" + c.expression + ")")
				if err != nil {
					return compiledRule{}, err
				}
			}
			rule.anchored = c.anchored
		}
		if c.literalRun == nil && c.leftContext {
			if c.contextAnchored == nil {
				c.contextAnchored, err = c.compileRegexp(`^(?s:.)(?:` + c.expression + ")")
				if err != nil {
					return compiledRule{}, err
				}
			}
			rule.contextAnchored = c.contextAnchored
		}
	}
	if rule.plan.strategy != strategyFullScan && rule.plan.keywordRequired &&
		(rule.plan.maxWidth >= 0 || rule.validateContext != nil) && len(keywords) <= maxStreamingKeywords {
		rule.streamKeywords = make([]string, len(keywords))
		for i, keyword := range keywords {
			rule.streamKeywords[i] = strings.ToLower(keyword)
		}
	}
	return rule, nil
}

func (s *compiledRuleSet) addRuleKeywords(index uint16, keywords []string, matcherKeywords []matcherKeyword) []matcherKeyword {
	rule := &s.rules[index]
	s.all.add(index)
	if len(keywords) == 0 || !rule.plan.keywordRequired {
		s.keywordless.add(index)
	}
	if rule.plan.strategy != strategyFullScan {
		s.guided.add(index)
	}
	if rule.plan.strategy == strategyAnchored && rule.plan.leadingBoundary {
		s.boundary.add(index)
	}
	for _, keyword := range keywords {
		matcherKeywords = append(matcherKeywords, matcherKeyword{value: keyword, rule: index})
	}
	return matcherKeywords
}

func (s *compiledRuleSet) detect(value string, hits *keywordHits, findings []Match) []Match {
	return s.detectExcluding(value, hits, nil, findings)
}

// Exclusions mask native candidates before any regex, entropy or validator work.
// The shared matcher and the unfiltered catalog oracle remain unchanged.
func (s *compiledRuleSet) detectExcluding(value string, hits *keywordHits, disabled *ruleSet, findings []Match) []Match {
	if len(s.rules) == 0 {
		return findings
	}
	hits.reset(&s.guided, &s.boundary)
	var candidates ruleSet
	s.pairMatcher.matchHits(value, &candidates, hits)
	if !hits.invalid {
		hits.sort()
	}
	candidates.merge(s.keywordless)
	if disabled != nil {
		for i := range candidates {
			candidates[i] &^= disabled[i]
		}
	}
	return evaluateRuleSet(s.rules, candidates, value, hits, uniqueRuleMatches, findings)
}

func evaluateRuleSet(rules []compiledRule, candidates ruleSet, value string, hits *keywordHits, mode evaluationMode, findings []Match) []Match {
	var punctuation punctuationPresence
	for wordIndex, word := range candidates {
		for word != 0 {
			bit := bits.TrailingZeros64(word)
			index := wordIndex*64 + bit
			word &= word - 1
			if index >= len(rules) {
				continue
			}
			rule := &rules[index]
			if hits != nil {
				// minWidth is no greater than the rune count of any match, and every rune
				// occupies at least one byte, so a shorter value cannot match this rule.
				if len(value) < rule.plan.minWidth {
					continue
				}
				if !rule.asciiRequirement.possible(value, hits) {
					continue
				}
				if (rule.punctuation[0] != 0 || rule.punctuation[1] != 0) && !punctuation.contains(value, rule.punctuation) {
					continue
				}
				if (rule.punctuationChoice[0] != 0 || rule.punctuationChoice[1] != 0) && !punctuation.containsAny(value, rule.punctuationChoice) {
					continue
				}
				canGuide := !hits.nonASCII || rule.plan.unicodeGuided ||
					(rule.plan.unicodeFoldGuided && !hits.asciiFoldAlias)
				if rule.plan.strategy != strategyFullScan && canGuide && hits.guides(uint16(index)) {
					from, to := hits.rangeFor(uint16(index))
					// Dense candidate starts can cost more than one necessary-run
					// check. Sparse starts retain the bounded-window fast path.
					if to-from >= 8 && rule.run.width > 0 &&
						rule.run.canReject(hits) && !rule.run.exists(value) {
						continue
					}
					findings = rule.detectAtKeywords(value, hits, from, to, findings, mode)
					continue
				}
				// Necessary runs can reject mixed values only when their alphabet
				// proof and the input's Unicode-fold facts preserve byte matching.
				if rule.run.width > 0 && rule.run.canReject(hits) && !rule.run.exists(value) {
					continue
				}
				if !hits.invalid && canGuide && len(rule.streamKeywords) != 0 {
					findings = rule.detectStreamingKeywords(value, hits, findings, mode)
					continue
				}
				// A prefix DFA cannot reject an unanchored search. Rules losing guidance
				// use their separate unanchored DFA, then the original regexp on a hit.
				filter := rule.filter
				if rule.plan.strategy == strategyAnchored {
					filter = rule.fallbackFilter
				}
				if filter == nil && !literalAlternativesPossible(value, rule.literalAlternatives, hits) {
					continue
				}
				// Exact factors complement folded field names on mixed values and
				// remain necessary even with fold aliases or malformed UTF-8.
				if filter == nil && (hits.invalid || hits.nonASCII) &&
					!literalAlternativesPossible(value, rule.exactLiteralAlternatives, hits) {
					continue
				}
				if mode == uniqueRuleMatches && rule.entropy == 0 && rule.validate == nil && rule.validateContext == nil &&
					!hits.invalid && !hits.nonASCII && rule.plan.strategy == strategyAnchored && filter != nil && len(filter.prefix) >= 2 {
					if rule.matchesLiteralStarts(value, filter.prefix) {
						findings = append(findings, Match{RuleID: rule.id})
					}
					continue
				}
				if filter != nil {
					if len(value) >= rejectionPrefixMinBytes && filter.prefix != "" {
						if !filter.mayMatchAfterPrefix(value) {
							continue
						}
					} else if !filter.mayMatch(value) {
						continue
					}
				}
			}
			// Without capture-dependent acceptance, the consumer needs only existence.
			// A nil hit set is the unfiltered, multiplicity-preserving oracle.
			switch {
			case hits == nil || mode != uniqueRuleMatches:
				findings = rule.detect(value, findings)
			case rule.entropy == 0 && rule.validate == nil && rule.validateContext == nil:
				if rule.regex.MatchString(value) {
					findings = append(findings, Match{RuleID: rule.id})
				}
			default:
				findings = rule.detectForVerdict(value, findings)
			}
		}
	}
	return findings
}

// detect is the unfiltered full-finding reference. Optimized evaluation preserves
// its full sequence or, in existence mode, its ordered unique-ID projection.
func (r *compiledRule) detect(value string, findings []Match) []Match {
	for _, indices := range r.regex.FindAllStringSubmatchIndex(value, -1) {
		findings, _ = r.appendFinding(value, indices, findings)
	}
	return findings
}

// The public consumer needs one accepted occurrence per rule, not every
// occurrence. Avoid FindAll's remainder scan when the first candidate is valid.
// If entropy or structural validation rejects it, retain the complete reference
// scan so a later accepted candidate cannot be hidden.
func (r *compiledRule) detectForVerdict(value string, findings []Match) []Match {
	indices := r.regex.FindStringSubmatchIndex(value)
	if indices == nil {
		return findings
	}
	before := len(findings)
	var rejectUntil int
	findings, rejectUntil = r.appendFinding(value, indices, findings)
	if len(findings) != before || rejectUntil == len(value) {
		return findings
	}
	for _, indices := range r.regex.FindAllStringSubmatchIndex(value, -1) {
		if indices[0] < rejectUntil {
			continue
		}
		findings, rejectUntil = r.appendFinding(value, indices, findings)
		if len(findings) != before || rejectUntil == len(value) {
			return findings
		}
	}
	return findings
}

// match returns original-value offsets for the complete match and sole capture.
// Reading the unsliced value preserves real EOF, word boundaries and the width
// of a consumed Unicode delimiter, including RuneError for malformed UTF-8.
func (m *literalRunMatcher) match(value string, start int) (int, int, bool) {
	if m.leadingBoundary && !asciiWordBoundary(value, start) || !strings.HasPrefix(value[start:], m.prefix) {
		return 0, 0, false
	}
	bodyStart := start + len(m.prefix)
	limit := len(value)
	if m.max >= 0 {
		limit = min(limit, bodyStart+m.max)
	}
	end := bodyStart
	for end < limit && m.body.has(value[end]) {
		end++
	}
	if end-bodyStart < m.min {
		return 0, 0, false
	}
	switch m.trailing {
	case syntax.OpEmptyMatch:
		return end, end, true
	case syntax.OpWordBoundary:
		return end, end, asciiWordBoundary(value, end)
	case syntax.OpEndText:
		return end, end, end == len(value)
	case syntax.OpAlternate:
		if end == len(value) {
			return end, end, true
		}
		r, width := utf8.DecodeRuneInString(value[end:])
		return end + width, end, m.delimiter.MatchRune(r)
	}
	return 0, 0, false
}

// Complete literal starts can be streamed without retaining unbounded keyword
// positions. This Boolean path is used only for ASCII values and rules without
// capture-dependent acceptance; overlapping starts are intentionally considered.
func (r *compiledRule) matchesLiteralStarts(value, prefix string) bool {
	for next := 0; next < len(value); {
		offset := strings.Index(value[next:], prefix)
		if offset < 0 {
			return false
		}
		start := next + offset
		next = start + 1
		if r.plan.leadingBoundary && !asciiWordBoundary(value, start) {
			continue
		}
		end := len(value)
		if r.plan.maxWidth >= 0 {
			end = min(end, start+r.plan.maxWidth+1)
		}
		if end-start < r.plan.minWidth || r.filter != nil && !r.filter.mayMatchPrefix(value[start:end]) {
			continue
		}
		if r.literalRun != nil {
			if _, _, matched := r.literalRun.match(value, start); matched {
				return true
			}
			continue
		}
		if start == 0 || r.contextAnchored == nil {
			if r.anchored == nil {
				return r.regex.MatchString(value)
			}
			if r.anchored.MatchString(value[start:end]) {
				return true
			}
		} else if r.contextAnchored.MatchString(value[start-1 : end]) {
			return true
		}
	}
	return false
}

// detectAtKeywords uses a complete range of verified keyword byte positions.
// The caller checks the plan's input-safety proof. In allRuleMatches mode the
// output equals detect; an empty complete range means the rule cannot match.
func (r *compiledRule) detectAtKeywords(value string, hits *keywordHits, from, to int, findings []Match, mode evaluationMode) []Match {
	if r.plan.strategy == strategyAnchored {
		return r.detectAnchored(value, hits, from, to, findings, mode)
	}
	return r.detectWindows(value, hits, from, to, findings, mode)
}

const maxStreamingKeywords = 16

// streamingKeywords merges monotonically searched positions without retaining
// a value-sized occurrence list. Its storage is independent of the value length.
type streamingKeywords struct {
	keywords  []string
	positions [maxStreamingKeywords]int
	byEnd     bool
}

func (s *streamingKeywords) next(value string, scan int) (int, int) {
	selected := -1
	selectedOrder := 0
	for i, keyword := range s.keywords {
		if s.positions[i] >= 0 && s.positions[i] < scan {
			s.positions[i] = indexFoldedASCII(value[scan:], keyword)
			if s.positions[i] >= 0 {
				s.positions[i] += scan
			}
		}
		order := s.positions[i]
		if s.byEnd {
			order += len(keyword)
		}
		if s.positions[i] >= 0 && (selected < 0 || order < selectedOrder) {
			selected = i
			selectedOrder = order
		}
	}
	if selected < 0 {
		return -1, -1
	}
	position := s.positions[selected]
	end := position + len(s.keywords[selected])
	next := position + 1
	s.positions[selected] = indexFoldedASCII(value[next:], s.keywords[selected])
	if s.positions[selected] >= 0 {
		s.positions[selected] += next
	}
	return position, end
}

func (r *compiledRule) detectStreamingKeywords(value string, hits *keywordHits, findings []Match, mode evaluationMode) []Match {
	stream := streamingKeywords{keywords: r.streamKeywords, byEnd: r.plan.strategy == strategyWindow}
	for i, keyword := range stream.keywords {
		stream.positions[i] = indexFoldedASCII(value, keyword)
	}
	if r.plan.strategy == strategyWindow {
		return r.detectStreamingWindows(value, hits, findings, mode, &stream)
	}
	return r.detectAnchoredCandidates(value, hits, 0, 0, findings, mode, &stream)
}

// detectAnchored reproduces FindAll in allRuleMatches mode: the leftmost match starting at or after the scan position
// begins with the earliest keyword occurrence (after its optional prefix) at which the
// anchored regex matches, and scanning resumes at that match's end.
func (r *compiledRule) detectAnchored(value string, hits *keywordHits, from, to int, findings []Match, mode evaluationMode) []Match {
	return r.detectAnchoredCandidates(value, hits, from, to, findings, mode, nil)
}

func (r *compiledRule) detectAnchoredCandidates(value string, hits *keywordHits, from, to int, findings []Match, mode evaluationMode, stream *streamingKeywords) []Match {
	scan := 0
	contextRejectUntil := 0
	width := r.plan.maxWidth
	if hits.nonASCII && width >= 0 {
		width *= utf8.UTFMax
	}
	for {
		var position int
		if stream == nil {
			if from == to {
				break
			}
			position = int(hits.buffer[from].start)
			from++
		} else {
			position, _ = stream.next(value, scan)
			if position < 0 {
				break
			}
		}
		if position < scan {
			continue
		}
		if r.plan.leadingBoundary && !asciiWordBoundary(value, position) {
			continue
		}
		start := position
		for start > scan && position-start < r.plan.prefixMax && r.plan.prefixClass.has(value[start-1]) {
			start--
		}
		end := len(value)
		if width >= 0 {
			end = min(end, start+width+1)
			if hits.nonASCII {
				end = runeWindowEnd(value, end)
			}
		}
		// minWidth is no greater than the byte width of any match, so the anchored regex
		// cannot match inside a shorter candidate interval.
		if end-start < r.plan.minWidth {
			continue
		}
		// The prefix filter accepts a superset of every regex match beginning at start, so
		// rejection proves the anchored regexp cannot match this candidate interval.
		if r.filter != nil && !r.filter.mayMatchPrefix(value[start:end]) {
			continue
		}
		var indices []int
		if r.literalRun != nil {
			matchEnd, captureEnd, matched := r.literalRun.match(value, start)
			if !matched {
				continue
			}
			if mode == uniqueRuleMatches && r.entropy == 0 && r.validate == nil && r.validateContext == nil {
				return append(findings, Match{RuleID: r.id})
			}
			captures := [4]int{start, matchEnd, start, captureEnd}
			indices = captures[:]
		} else {
			contextStart := start
			anchored := r.anchored
			if start > 0 && r.contextAnchored != nil {
				contextStart--
				if hits.nonASCII && value[contextStart] >= utf8.RuneSelf {
					_, contextWidth := utf8.DecodeLastRuneInString(value[:start])
					contextStart = start - contextWidth
				}
				anchored = r.contextAnchored
			}
			// At offset zero, the original regexp sees the real left context.
			// A bounded streamed candidate also retains a rune beyond its maximum
			// match width. Ignore later starts, whose matches could be truncated
			// at that endpoint; their keyword occurrences are considered in turn.
			if start == 0 && stream != nil && width >= 0 {
				indices = r.regex.FindStringSubmatchIndex(value[:end])
				if indices != nil && indices[0] != 0 {
					continue
				}
			} else if mode == uniqueRuleMatches && r.entropy == 0 && r.validate == nil && r.validateContext == nil {
				if anchored == nil {
					if r.regex.MatchString(value) {
						return append(findings, Match{RuleID: r.id})
					}
					continue
				}
				if anchored.MatchString(value[contextStart:end]) {
					return append(findings, Match{RuleID: r.id})
				}
				continue
			} else if start == 0 {
				if r.anchored != nil {
					indices = r.anchored.FindStringSubmatchIndex(value)
				} else {
					indices = r.regex.FindStringSubmatchIndex(value)
				}
			} else if indices = anchored.FindStringSubmatchIndex(value[contextStart:end]); indices != nil {
				for k := range indices {
					if indices[k] >= 0 {
						indices[k] += contextStart
					}
				}
				indices[0] = start
			}
		}
		if indices == nil {
			continue
		}
		// Keep consuming the reference's regex matches even inside a rejected
		// context range; jumping the regex cursor could expose overlapping hits.
		if mode == uniqueRuleMatches && indices[0] < contextRejectUntil {
			scan = indices[1]
			continue
		}
		before := len(findings)
		findings, contextRejectUntil = r.appendFinding(value, indices, findings)
		if mode == uniqueRuleMatches && len(findings) != before {
			return findings
		}
		if mode == uniqueRuleMatches && contextRejectUntil == len(value) {
			return findings
		}
		scan = indices[1]
	}
	return findings
}

// detectWindows runs FindAll over the merged windows around keyword occurrences. Windows are
// disjoint, every match lies strictly inside one of them, and the scan enters each window
// with no pending match, so the concatenated results equal a whole-value FindAll in
// allRuleMatches mode. A window missing a required factor is skipped. The unique-ID
// consumer stops at its first accepted match, after any required capture validation.
func (r *compiledRule) detectWindows(value string, hits *keywordHits, from, to int, findings []Match, mode evaluationMode) []Match {
	hits.sortRangeByEnd(from, to)
	width := r.plan.maxWidth
	if hits.nonASCII {
		width *= utf8.UTFMax
	}
	for i := from; i < to; {
		low := max(0, int(hits.buffer[i].end)-width-1)
		high := min(len(value), int(hits.buffer[i].start)+width+1)
		if hits.nonASCII {
			low = runeWindowStart(value, low)
			high = runeWindowEnd(value, high)
		}
		j := i + 1
		for ; j < to; j++ {
			nextLow := max(0, int(hits.buffer[j].end)-width-1)
			if hits.nonASCII {
				nextLow = runeWindowStart(value, nextLow)
			}
			if nextLow > high {
				break
			}
			nextHigh := min(len(value), int(hits.buffer[j].start)+width+1)
			if hits.nonASCII {
				nextHigh = runeWindowEnd(value, nextHigh)
			}
			high = max(high, nextHigh)
		}
		before := len(findings)
		findings = r.detectWindow(value, hits, low, high, findings, mode)
		if mode == uniqueRuleMatches && len(findings) != before {
			return findings
		}
		i = j
	}
	return findings
}

// Window starts are monotone in keyword ends, not keyword starts: different
// keyword lengths can reverse their order. Merge exactly as for recorded hits,
// retaining only one pending occurrence and the fixed-size keyword stream.
func (r *compiledRule) detectStreamingWindows(value string, hits *keywordHits, findings []Match, mode evaluationMode, stream *streamingKeywords) []Match {
	width := r.plan.maxWidth
	if hits.nonASCII {
		width *= utf8.UTFMax
	}
	start, end := stream.next(value, 0)
	for start >= 0 {
		low := max(0, end-width-1)
		high := min(len(value), start+width+1)
		if hits.nonASCII {
			low = runeWindowStart(value, low)
			high = runeWindowEnd(value, high)
		}
		start, end = stream.next(value, 0)
		for start >= 0 {
			nextLow := max(0, end-width-1)
			if hits.nonASCII {
				nextLow = runeWindowStart(value, nextLow)
			}
			if nextLow > high {
				break
			}
			nextHigh := min(len(value), start+width+1)
			if hits.nonASCII {
				nextHigh = runeWindowEnd(value, nextHigh)
			}
			high = max(high, nextHigh)
			start, end = stream.next(value, 0)
		}
		before := len(findings)
		findings = r.detectWindow(value, hits, low, high, findings, mode)
		if mode == uniqueRuleMatches && len(findings) != before {
			return findings
		}
	}
	return findings
}

func (r *compiledRule) detectWindow(value string, hits *keywordHits, low, high int, findings []Match, mode evaluationMode) []Match {
	// minWidth is no greater than the byte width of any match, so a shorter window
	// cannot contain a match.
	if high-low < r.plan.minWidth || low >= high {
		return findings
	}
	window := value[low:high]
	// The complete regex match lies inside this window. Check its
	// mandatory run here, without scanning an unrelated long prefix.
	if r.run.width > 0 && r.run.canReject(hits) && !r.run.exists(window) {
		return findings
	}
	// Both required factors and the unanchored filter are necessary conditions for
	// a regex match in this window, so either negative result soundly skips it.
	if !hits.asciiFoldAlias && !containsFoldedFactors(window, r.plan.factors) || r.filter != nil && !r.filter.mayMatch(window) {
		return findings
	}
	if mode == uniqueRuleMatches && r.entropy == 0 && r.validate == nil && r.validateContext == nil {
		if r.regex.MatchString(window) {
			return append(findings, Match{RuleID: r.id})
		}
		return findings
	}
	if mode == uniqueRuleMatches && r.validateContext == nil {
		return r.detectForVerdict(window, findings)
	}
	for _, indices := range r.regex.FindAllStringSubmatchIndex(window, -1) {
		for k := range indices {
			if indices[k] >= 0 {
				indices[k] += low
			}
		}
		before := len(findings)
		findings, _ = r.appendFinding(value, indices, findings)
		if mode == uniqueRuleMatches && len(findings) != before {
			return findings
		}
	}
	return findings
}

// Byte width bounds may land inside a valid multibyte encoding. Expand outwards
// to retain its real rune instead of manufacturing RuneError at a sliced edge.
// Invalid UTF-8 bytes already occupy one rune each in Go regexp and stay intact.
func runeWindowStart(value string, position int) int {
	if position == len(value) || utf8.RuneStart(value[position]) {
		return position
	}
	for start := position - 1; start >= 0 && position-start < utf8.UTFMax; start-- {
		if utf8.RuneStart(value[start]) {
			_, width := utf8.DecodeRuneInString(value[start:])
			if start+width > position {
				return start
			}
			break
		}
	}
	return position
}

func runeWindowEnd(value string, position int) int {
	start := runeWindowStart(value, position)
	if start < position {
		_, width := utf8.DecodeRuneInString(value[start:])
		return start + width
	}
	return position
}

func (r *compiledRule) appendFinding(value string, indices []int, findings []Match) ([]Match, int) {
	rawMatch := value[indices[0]:indices[1]]
	fullMatch := strings.Trim(rawMatch, "\n")
	secret := extractSecretFromIndices(value, fullMatch, rawMatch, indices, r.regex, r.secretGroup)
	if r.validate != nil && !r.validate(secret) {
		return findings, 0
	}
	if r.entropy != 0 && shannonEntropy(secret) <= r.entropy {
		return findings, 0
	}
	if r.validateContext != nil {
		context := r.validateContext(value, indices[0], indices[1], secret)
		if !context.accepted {
			if context.rejectUntil >= indices[1] && context.rejectUntil <= len(value) {
				return findings, context.rejectUntil
			}
			return findings, 0
		}
	}
	return append(findings, Match{RuleID: r.id}), 0
}

// Case-sensitive branches retain their exact case. Folded ASCII checks cannot
// reject when Unicode aliases or unavailable input facts could hide a match.
func literalAlternativesPossible(value string, alternatives []literalAlternative, hits *keywordHits) bool {
	if len(alternatives) == 0 {
		return true
	}
	for _, literal := range alternatives {
		if literal.fold {
			if hits == nil || hits.invalid || hits.asciiFoldAlias || indexFoldedASCII(value, literal.text) >= 0 {
				return true
			}
		} else if strings.Contains(value, literal.text) {
			return true
		}
	}
	return false
}

// containsFoldedFactors reports whether the ASCII window contains every folded factor, comparing
// case-insensitively. Folding is a superset of any case-sensitive literal, so a window that
// fails this check cannot match the regex.
func containsFoldedFactors(window string, factors []string) bool {
	for _, factor := range factors {
		if indexFoldedASCII(window, factor) < 0 {
			return false
		}
	}
	return true
}

// factor must already be ASCII-lowercase; only window is folded during scanning.
func indexFoldedASCII(window, factor string) int {
	first := factor[0]
	for i := 0; i+len(factor) <= len(window); i++ {
		if foldASCII(window[i]) != first {
			continue
		}
		j := 1
		for j < len(factor) && foldASCII(window[i+j]) == factor[j] {
			j++
		}
		if j == len(factor) {
			return i
		}
	}
	return -1
}

func asciiWordBoundary(value string, position int) bool {
	before := position > 0 && isASCIIWordByte(value[position-1])
	after := position < len(value) && isASCIIWordByte(value[position])
	return before != after
}

func isASCIIWordByte(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z' || value == '_'
}

func extractSecretFromIndices(value, fullMatch, rawMatch string, indices []int, re *regexp.Regexp, secretGroup int) string {
	if len(fullMatch) != len(rawMatch) {
		return extractSecret(re, fullMatch, secretGroup)
	}
	if len(indices) < 4 {
		return fullMatch
	}
	if secretGroup > 0 {
		offset := secretGroup * 2
		if offset+1 < len(indices) && indices[offset] >= 0 {
			return value[indices[offset]:indices[offset+1]]
		}
		return ""
	}
	for offset := 2; offset+1 < len(indices); offset += 2 {
		if indices[offset] >= 0 && indices[offset] != indices[offset+1] {
			return value[indices[offset]:indices[offset+1]]
		}
	}
	return fullMatch
}

func extractSecret(re *regexp.Regexp, fullMatch string, secretGroup int) string {
	groups := re.FindStringSubmatch(fullMatch)
	if len(groups) < 2 {
		return fullMatch
	}
	if secretGroup > 0 {
		if secretGroup < len(groups) {
			return groups[secretGroup]
		}
		return ""
	}
	for _, group := range groups[1:] {
		if group != "" {
			return group
		}
	}
	return fullMatch
}

const entropyCountTermTableSize = 256

var entropyCountTerms = func() [entropyCountTermTableSize]float64 {
	var terms [entropyCountTermTableSize]float64
	for count := 2; count < len(terms); count++ {
		terms[count] = float64(count) * math.Log2(float64(count))
	}
	return terms
}()

func shannonEntropy(value string) float64 {
	if value == "" {
		return 0
	}
	var asciiCounts [128]int
	ascii := true
	for i := range len(value) {
		if value[i] >= 0x80 {
			ascii = false
			break
		}
		asciiCounts[value[i]]++
	}
	inverseLength := 1 / float64(len(value))
	logLength := math.Log2(float64(len(value)))
	if ascii {
		entropy := logLength
		for _, count := range asciiCounts {
			if count != 0 {
				entropy -= entropyCountTerm(count) * inverseLength
			}
		}
		return entropy
	}
	counts := make(map[rune]int)
	runeCount := 0
	for _, char := range value {
		counts[char]++
		runeCount++
	}
	entropy := float64(runeCount) * inverseLength * logLength
	for _, count := range counts {
		entropy -= entropyCountTerm(count) * inverseLength
	}
	return entropy
}

func entropyCountTerm(count int) float64 {
	if count < len(entropyCountTerms) {
		return entropyCountTerms[count]
	}
	return float64(count) * math.Log2(float64(count))
}
