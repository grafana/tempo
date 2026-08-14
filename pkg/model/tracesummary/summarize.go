package tracesummary

import (
	"bytes"
	"errors"
	"sort"

	modeltrace "github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
)

const (
	serviceNameAttribute = "service.name"
	childOfRefTypeKey    = "opentracing.ref_type"
	childOfRefTypeValue  = "child_of"

	maxSlowestSpans = 5
	maxErrorSpans   = 5
)

// resolvedSpan pairs a span with its resolved service name (resource-level
// service.name attribute — the OTel-spec source of truth — falling back to
// a span-level attribute of the same name for non-conformant data).
type resolvedSpan struct {
	span    *tracev1.Span
	service string
}

// Summarize computes a condensed Summary over an assembled trace: root
// span, overall duration, critical path, per-service breakdown, top-5
// slowest spans and first-5 error spans.
//
// A nil trace is a caller error and returns an error. A trace with no spans
// is a valid (if unusual) input and returns a zero-value Summary with no
// error.
func Summarize(trace *tempopb.Trace) (*Summary, error) {
	if trace == nil {
		return nil, errors.New("tracesummary: trace is nil")
	}

	spans := flattenSpans(trace)
	if len(spans) == 0 {
		return &Summary{}, nil
	}

	byID := indexSpansByID(spans)
	root := findRoot(spans)
	children := buildChildrenIndex(spans, byID)
	self := selfDurations(spans, children)
	minStart, maxEnd := traceBounds(spans)

	summary := &Summary{
		TraceID:       util.TraceIDToHexString(root.span.GetTraceId()),
		RootService:   root.service,
		RootSpanName:  root.span.GetName(),
		DurationNanos: durationBetween(minStart, maxEnd),
		SpanCount:     len(spans),
		ErrorCount:    countErrors(spans),
		CriticalPath:  criticalPath(root, children, self),
		Services:      serviceBreakdown(spans, self),
		SlowestSpans:  slowestSpans(spans, self),
		ErrorSpans:    errorSpans(spans),
	}

	return summary, nil
}

func flattenSpans(trace *tempopb.Trace) []*resolvedSpan {
	var spans []*resolvedSpan
	for _, rs := range trace.GetResourceSpans() {
		resourceService := attributeString(rs.GetResource().GetAttributes(), serviceNameAttribute)
		for _, ss := range rs.GetScopeSpans() {
			for _, span := range ss.GetSpans() {
				if span == nil {
					continue
				}
				spans = append(spans, &resolvedSpan{
					span:    span,
					service: resolveService(span, resourceService),
				})
			}
		}
	}
	return spans
}

func resolveService(span *tracev1.Span, resourceService string) string {
	if resourceService != "" {
		return resourceService
	}
	return attributeString(span.GetAttributes(), serviceNameAttribute)
}

// indexSpansByID maps a span's own ID to every span sharing that ID. Zipkin
// traces can carry client/server span pairs with identical span IDs, so
// lookups by ID must be disambiguated by kind (see disambiguateParent).
func indexSpansByID(spans []*resolvedSpan) map[uint64][]*resolvedSpan {
	idx := make(map[uint64][]*resolvedSpan, len(spans))
	for _, s := range spans {
		key := util.SpanIDToUint64(s.span.GetSpanId())
		idx[key] = append(idx[key], s)
	}
	return idx
}

// disambiguateParent picks the parent span for child among candidates that
// share a span ID. With two candidates (the zipkin client/server pattern), a
// SERVER-kind child prefers a CLIENT-kind parent and vice versa.
//
// On the real trace-summary endpoint path this is unreachable in practice:
// combiner.NewTraceByIDV2's finalize() runs newDeduper().dedupe() over the
// assembled trace before Summarize ever sees it, and that deduper already
// resolves zipkin dual-ID pairs by assigning the SERVER-kind span a new,
// unique span ID and reparenting it (and its former children) under the
// original shared ID, which the CLIENT-kind span keeps. So by the time
// Summarize runs on real traffic, no two spans in the trace share an ID.
// This function exists purely as defense-in-depth for any future or direct
// caller of Summarize against un-deduped input, so that such input can't
// panic or silently misattribute a parent.
func disambiguateParent(candidates []*resolvedSpan, child *tracev1.Span) *resolvedSpan {
	switch len(candidates) {
	case 0:
		return nil
	case 1:
		return candidates[0]
	case 2:
		kindWant := tracev1.Span_SPAN_KIND_SERVER
		if child.GetKind() == tracev1.Span_SPAN_KIND_SERVER {
			kindWant = tracev1.Span_SPAN_KIND_CLIENT
		}
		if candidates[0].span.GetKind() == kindWant {
			return candidates[0]
		}
		if candidates[1].span.GetKind() == kindWant {
			return candidates[1]
		}
		return nil
	default:
		// More than two spans sharing an ID is invalid data; best-effort
		// rather than panicking or dropping the parent link entirely. Sort
		// deterministically (earliest start, then smallest hex span ID) so
		// the choice doesn't depend on unordered combiner merge order.
		sorted := make([]*resolvedSpan, len(candidates))
		copy(sorted, candidates)
		sort.Slice(sorted, func(i, j int) bool {
			return lessByStartThenID(sorted[i].span, sorted[j].span)
		})
		return sorted[0]
	}
}

func lookupParent(byID map[uint64][]*resolvedSpan, span *tracev1.Span) *resolvedSpan {
	if len(span.GetParentSpanId()) == 0 {
		return nil
	}
	key := util.SpanIDToUint64(span.GetParentSpanId())
	return disambiguateParent(byID[key], span)
}

func hasChildOfLink(span *tracev1.Span) bool {
	for _, link := range span.GetLinks() {
		if !bytes.Equal(link.GetTraceId(), span.GetTraceId()) {
			continue
		}
		for _, attr := range link.GetAttributes() {
			if attr.GetKey() == childOfRefTypeKey && attr.GetValue().GetStringValue() == childOfRefTypeValue {
				return true
			}
		}
	}
	return false
}

// findRoot picks the root span: parentless and without a child_of link to
// the same trace. Ties are broken by earliest start, then smallest hex span
// ID. If no span qualifies, the globally earliest-starting span is used as
// a synthetic root.
func findRoot(spans []*resolvedSpan) *resolvedSpan {
	var candidates []*resolvedSpan
	for _, s := range spans {
		if len(s.span.GetParentSpanId()) == 0 && !hasChildOfLink(s.span) {
			candidates = append(candidates, s)
		}
	}
	if len(candidates) == 0 {
		candidates = spans
	}

	best := candidates[0]
	for _, c := range candidates[1:] {
		if lessByStartThenID(c.span, best.span) {
			best = c
		}
	}
	return best
}

func lessByStartThenID(a, b *tracev1.Span) bool {
	if a.GetStartTimeUnixNano() != b.GetStartTimeUnixNano() {
		return a.GetStartTimeUnixNano() < b.GetStartTimeUnixNano()
	}
	return util.SpanIDToHexString(a.GetSpanId()) < util.SpanIDToHexString(b.GetSpanId())
}

func traceBounds(spans []*resolvedSpan) (minStart, maxEnd uint64) {
	minStart = spans[0].span.GetStartTimeUnixNano()
	maxEnd = spans[0].span.GetEndTimeUnixNano()
	for _, s := range spans[1:] {
		if st := s.span.GetStartTimeUnixNano(); st < minStart {
			minStart = st
		}
		if et := s.span.GetEndTimeUnixNano(); et > maxEnd {
			maxEnd = et
		}
	}
	return minStart, maxEnd
}

func durationBetween(start, end uint64) int64 {
	if end < start {
		return 0
	}
	return int64(end - start)
}

func spanDuration(span *tracev1.Span) int64 {
	return durationBetween(span.GetStartTimeUnixNano(), span.GetEndTimeUnixNano())
}

func parentSpanIDHex(span *tracev1.Span) string {
	if len(span.GetParentSpanId()) == 0 {
		return ""
	}
	return util.SpanIDToHexString(span.GetParentSpanId())
}

// selfDurations computes each span's exclusive duration: its own wall time
// minus the time already accounted for by its direct children.
//
// Child intervals are unioned rather than summed, and clipped to the parent's
// own window. Summing would over-subtract whenever children run concurrently —
// a span fanning out to ten parallel calls would report zero self time — and
// children can start before or outlive their parent in real traces.
func selfDurations(spans []*resolvedSpan, children map[*resolvedSpan][]*resolvedSpan) map[*tracev1.Span]int64 {
	self := make(map[*tracev1.Span]int64, len(spans))
	for _, s := range spans {
		self[s.span] = exclusiveDuration(s, children[s])
	}
	return self
}

func exclusiveDuration(parent *resolvedSpan, kids []*resolvedSpan) int64 {
	total := spanDuration(parent.span)
	if total == 0 || len(kids) == 0 {
		return total
	}

	windowStart := parent.span.GetStartTimeUnixNano()
	windowEnd := parent.span.GetEndTimeUnixNano()

	intervals := make([][2]uint64, 0, len(kids))
	for _, k := range kids {
		start := k.span.GetStartTimeUnixNano()
		end := k.span.GetEndTimeUnixNano()
		if start < windowStart {
			start = windowStart
		}
		if end > windowEnd {
			end = windowEnd
		}
		if end > start {
			intervals = append(intervals, [2]uint64{start, end})
		}
	}
	if len(intervals) == 0 {
		return total
	}
	sort.Slice(intervals, func(i, j int) bool { return intervals[i][0] < intervals[j][0] })

	var covered uint64
	curStart, curEnd := intervals[0][0], intervals[0][1]
	for _, iv := range intervals[1:] {
		if iv[0] > curEnd {
			covered += curEnd - curStart
			curStart, curEnd = iv[0], iv[1]
			continue
		}
		if iv[1] > curEnd {
			curEnd = iv[1]
		}
	}
	covered += curEnd - curStart

	// covered is clipped to the parent's window, so it can't exceed total.
	if int64(covered) >= total {
		return 0
	}
	return total - int64(covered)
}

func isError(span *tracev1.Span) bool {
	return span.GetStatus().GetCode() == tracev1.Status_STATUS_CODE_ERROR
}

// isLaterEnd reports whether a's end time makes it the later-ending span,
// breaking ties by earliest start then smallest hex span ID.
func isLaterEnd(a, b *tracev1.Span) bool {
	if a.GetEndTimeUnixNano() != b.GetEndTimeUnixNano() {
		return a.GetEndTimeUnixNano() > b.GetEndTimeUnixNano()
	}
	return lessByStartThenID(a, b)
}

// buildChildrenIndex maps each span to its direct children, resolving
// zipkin-style duplicate span IDs the same way lookupParent does.
func buildChildrenIndex(spans []*resolvedSpan, byID map[uint64][]*resolvedSpan) map[*resolvedSpan][]*resolvedSpan {
	children := make(map[*resolvedSpan][]*resolvedSpan, len(spans))
	for _, s := range spans {
		parent := lookupParent(byID, s.span)
		if parent == nil {
			continue
		}
		children[parent] = append(children[parent], s)
	}
	return children
}

// criticalPath walks down from the root, at each level descending into the
// direct child whose span ends latest (tie-break: earliest start, then
// smallest hex span ID), until it reaches a span with no children.
func criticalPath(root *resolvedSpan, children map[*resolvedSpan][]*resolvedSpan, self map[*tracev1.Span]int64) []PathSpan {
	var chain []*resolvedSpan
	visited := make(map[*resolvedSpan]struct{}, len(children)+1)

	cur := root
	for cur != nil {
		if _, ok := visited[cur]; ok {
			break
		}
		visited[cur] = struct{}{}
		chain = append(chain, cur)

		kids := children[cur]
		if len(kids) == 0 {
			break
		}

		next := kids[0]
		for _, k := range kids[1:] {
			if isLaterEnd(k.span, next.span) {
				next = k
			}
		}
		cur = next
	}

	out := make([]PathSpan, len(chain))
	for i, s := range chain {
		out[i] = PathSpan{
			SpanID:            util.SpanIDToHexString(s.span.GetSpanId()),
			Service:           s.service,
			Name:              s.span.GetName(),
			Kind:              modeltrace.KindToString(s.span.GetKind()),
			StartTimeUnixNano: s.span.GetStartTimeUnixNano(),
			DurationNanos:     spanDuration(s.span),
			SelfDurationNanos: self[s.span],
			Attributes:        attributesMap(s.span.GetAttributes()),
		}
	}
	return out
}

func serviceBreakdown(spans []*resolvedSpan, self map[*tracev1.Span]int64) []ServiceBreakdown {
	type acc struct {
		spanCount     int
		errorCount    int
		durationNanos int64
		selfNanos     int64
	}

	stats := map[string]*acc{}
	var order []string
	for _, s := range spans {
		a, ok := stats[s.service]
		if !ok {
			a = &acc{}
			stats[s.service] = a
			order = append(order, s.service)
		}
		a.spanCount++
		if isError(s.span) {
			a.errorCount++
		}
		a.durationNanos += spanDuration(s.span)
		a.selfNanos += self[s.span]
	}
	sort.Strings(order)

	out := make([]ServiceBreakdown, 0, len(order))
	for _, svc := range order {
		a := stats[svc]
		out = append(out, ServiceBreakdown{
			Service:           svc,
			SpanCount:         a.spanCount,
			ErrorCount:        a.errorCount,
			DurationNanos:     a.durationNanos,
			SelfDurationNanos: a.selfNanos,
		})
	}
	return out
}

// insertRanked inserts s into top, which is kept sorted by less and capped at
// k entries, dropping the lowest-ranked entry once full. Selecting a bounded
// number of spans this way avoids copying and sorting every span in the trace.
func insertRanked(top []*resolvedSpan, k int, s *resolvedSpan, less func(a, b *tracev1.Span) bool) []*resolvedSpan {
	if k <= 0 {
		return top
	}
	if len(top) == k {
		if !less(s.span, top[k-1].span) {
			return top
		}
	} else {
		top = append(top, nil)
	}

	i := len(top) - 1
	for i > 0 && less(s.span, top[i-1].span) {
		top[i] = top[i-1]
		i--
	}
	top[i] = s
	return top
}

// longerSelfDuration reports whether a spent more time in itself, breaking ties
// by earliest start then smallest hex span ID.
//
// Ranking by self time rather than wall time is what makes this list worth
// returning. Wall time in a nested trace is dominated by ancestry — the slowest
// spans are trivially the outermost ones, which the critical path already
// names. Self time surfaces the spans that actually burned the time.
func longerSelfDuration(self map[*tracev1.Span]int64) func(a, b *tracev1.Span) bool {
	return func(a, b *tracev1.Span) bool {
		sa, sb := self[a], self[b]
		if sa != sb {
			return sa > sb
		}
		return lessByStartThenID(a, b)
	}
}

func slowestSpans(spans []*resolvedSpan, self map[*tracev1.Span]int64) []SpanSummary {
	less := longerSelfDuration(self)
	top := make([]*resolvedSpan, 0, maxSlowestSpans)
	for _, s := range spans {
		top = insertRanked(top, maxSlowestSpans, s, less)
	}

	out := make([]SpanSummary, len(top))
	for i, s := range top {
		out[i] = SpanSummary{
			SpanID:            util.SpanIDToHexString(s.span.GetSpanId()),
			ParentSpanID:      parentSpanIDHex(s.span),
			Service:           s.service,
			Name:              s.span.GetName(),
			Kind:              modeltrace.KindToString(s.span.GetKind()),
			StartTimeUnixNano: s.span.GetStartTimeUnixNano(),
			DurationNanos:     spanDuration(s.span),
			SelfDurationNanos: self[s.span],
			Attributes:        attributesMap(s.span.GetAttributes()),
		}
	}
	return out
}

func errorSpans(spans []*resolvedSpan) []ErrorSpanSummary {
	top := make([]*resolvedSpan, 0, maxErrorSpans)
	for _, s := range spans {
		if !isError(s.span) {
			continue
		}
		top = insertRanked(top, maxErrorSpans, s, lessByStartThenID)
	}

	out := make([]ErrorSpanSummary, len(top))
	for i, s := range top {
		out[i] = ErrorSpanSummary{
			SpanID:            util.SpanIDToHexString(s.span.GetSpanId()),
			ParentSpanID:      parentSpanIDHex(s.span),
			Service:           s.service,
			Name:              s.span.GetName(),
			Kind:              modeltrace.KindToString(s.span.GetKind()),
			StartTimeUnixNano: s.span.GetStartTimeUnixNano(),
			DurationNanos:     spanDuration(s.span),
			StatusMessage:     s.span.GetStatus().GetMessage(),
			Attributes:        attributesMap(s.span.GetAttributes()),
		}
	}
	return out
}

func countErrors(spans []*resolvedSpan) int {
	count := 0
	for _, s := range spans {
		if isError(s.span) {
			count++
		}
	}
	return count
}

func attributeString(attrs []*commonv1.KeyValue, key string) string {
	for _, attr := range attrs {
		if attr.GetKey() == key {
			return attr.GetValue().GetStringValue()
		}
	}
	return ""
}

func attributesMap(attrs []*commonv1.KeyValue) map[string]any {
	out := make(map[string]any, len(attrs))
	for _, attr := range attrs {
		out[attr.GetKey()] = anyValue(attr.GetValue())
	}
	return out
}

func anyValue(value *commonv1.AnyValue) any {
	if value == nil {
		return nil
	}
	switch v := value.GetValue().(type) {
	case *commonv1.AnyValue_StringValue:
		return v.StringValue
	case *commonv1.AnyValue_BoolValue:
		return v.BoolValue
	case *commonv1.AnyValue_IntValue:
		return v.IntValue
	case *commonv1.AnyValue_DoubleValue:
		return v.DoubleValue
	case *commonv1.AnyValue_ArrayValue:
		values := v.ArrayValue.GetValues()
		out := make([]any, 0, len(values))
		for _, item := range values {
			out = append(out, anyValue(item))
		}
		return out
	case *commonv1.AnyValue_KvlistValue:
		return attributesMap(v.KvlistValue.GetValues())
	case *commonv1.AnyValue_BytesValue:
		return v.BytesValue
	default:
		return nil
	}
}
