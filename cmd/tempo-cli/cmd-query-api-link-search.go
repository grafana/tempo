package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/dskit/user"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
)

// queryAPILinkSearchCmd executes multi-phase cross-trace link traversal queries.
//
// Link traversal operators:
//   ->>   : Forward traversal (find spans linked TO by current spans)
//   <<-   : Backward traversal (find spans linking FROM current spans)
//   &->>  : Union forward traversal (return both sides of the link)
//   &<<-  : Union backward traversal (return both sides of the link)
//
// Example usage:
//
//   # Generate test data and build CLI
//   make link-dev && make tempo-cli
//
//   # 2-hop backward traversal (gateway ← backend)
//   ./bin/darwin/tempo-cli-arm64 query api link-search localhost:3200 '{span.service.name="gateway"} &<<- {span.service.name="backend"}' now-1h now --verbose
//
//   # 3-hop backward traversal (gateway ← backend ← database)
//   ./bin/darwin/tempo-cli-arm64 query api link-search localhost:3200 '{span.service.name="gateway"} &<<- {span.service.name="backend"} &<<- {span.service.name="database"}' now-1h now --verbose
//
//   # 3-hop forward traversal (database → backend → gateway)
//   ./bin/darwin/tempo-cli-arm64 query api link-search localhost:3200 '{span.service.name="database"} &->> {span.service.name="backend"} &->> {span.service.name="gateway"}' now-1h now --verbose
type queryAPILinkSearchCmd struct {
	HostPort string `arg:"" help:"tempo host and port. e.g. localhost:3200"`
	TraceQL  string `arg:"" help:"traceql query with link traversal operators (->>, <<-, &->>, &<<-)"`
	Start    string `arg:"" help:"start time (ISO8601 or relative: now, now-1h, now-30m, now-1d)"`
	End      string `arg:"" help:"end time (ISO8601 or relative: now, now-1h, now-30m, now-1d)"`

	OrgID              string `help:"optional orgID"`
	Limit              int    `help:"limit number of result traces" default:"20"`
	MaxSpanIDsPerPhase int    `help:"max span IDs passed between phases" default:"1000"`
	Verbose            bool   `help:"verbose logging for multi-phase execution"`
	PathPrefix         string `help:"string to prefix all http paths with"`
	Secure             bool   `help:"use https"`
}

var relativeTimeRegex = regexp.MustCompile(`^(\d+)([smhd])$`)

// parseTimeExpression parses time expressions supporting both ISO8601 and relative formats.
// Supported formats:
//   - ISO8601: 2024-01-01T00:00:00Z
//   - "now": current time
//   - "now-1h": 1 hour ago
//   - "now-30m": 30 minutes ago
//   - "now-1d": 1 day ago
func parseTimeExpression(expr string) (time.Time, error) {
	expr = strings.TrimSpace(expr)

	// Handle "now"
	if expr == "now" {
		return time.Now(), nil
	}

	// Handle "now-<duration>" format
	if durationStr, ok := strings.CutPrefix(expr, "now-"); ok {
		// Parse duration using regex to support h, m, d suffixes
		matches := relativeTimeRegex.FindStringSubmatch(durationStr)
		if matches == nil {
			return time.Time{}, fmt.Errorf("invalid relative time format: %s (expected format: now-1h, now-30m, now-1d)", expr)
		}

		value, err := strconv.Atoi(matches[1])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid duration value: %s", matches[1])
		}

		var duration time.Duration
		switch matches[2] {
		case "s":
			duration = time.Duration(value) * time.Second
		case "m":
			duration = time.Duration(value) * time.Minute
		case "h":
			duration = time.Duration(value) * time.Hour
		case "d":
			duration = time.Duration(value) * 24 * time.Hour
		default:
			return time.Time{}, fmt.Errorf("unsupported time unit: %s (supported: s, m, h, d)", matches[2])
		}

		return time.Now().Add(-duration), nil
	}

	// Try parsing as ISO8601
	t, err := time.Parse(time.RFC3339, expr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format: %s (expected ISO8601 or relative format like 'now-1h')", expr)
	}

	return t, nil
}

func (cmd *queryAPILinkSearchCmd) Run(_ *globalOptions) error {
	logger := log.NewLogfmtLogger(os.Stdout)

	// Parse timestamps
	startDate, err := parseTimeExpression(cmd.Start)
	if err != nil {
		return err
	}
	start := startDate.Unix()

	endDate, err := parseTimeExpression(cmd.End)
	if err != nil {
		return err
	}
	end := endDate.Unix()

	// Validate time range
	if start >= end {
		return fmt.Errorf("start time must be before end time (start: %s, end: %s)", cmd.Start, cmd.End)
	}

	// Parse TraceQL query
	rootExpr, err := traceql.Parse(cmd.TraceQL)
	if err != nil {
		return err
	}

	// Validate link traversal
	if !rootExpr.HasLinkTraversal() {
		return fmt.Errorf("query must contain link traversal operators (->> or <<-)")
	}

	// Extract link chain in execution order (terminal first)
	linkChain := rootExpr.ExtractLinkChain()
	if len(linkChain) == 0 {
		return fmt.Errorf("no link operations found in query")
	}

	if cmd.Verbose {
		level.Info(logger).Log("msg", "Cross-trace link traversal detected",
			"phases", len(linkChain),
			"maxSpanIDsPerPhase", cmd.MaxSpanIDsPerPhase)
	}

	// Execute multi-phase link search
	ctx := context.Background()
	result, err := cmd.executeLinkSearch(ctx, logger, linkChain, start, end)
	if err != nil {
		return fmt.Errorf("link search failed: %w", err)
	}

	// Output JSON result
	return printAsJSON(result)
}

// executeLinkSearch orchestrates multi-phase link traversal
func (cmd *queryAPILinkSearchCmd) executeLinkSearch(
	ctx context.Context,
	logger log.Logger,
	linkChain []*traceql.LinkOperationInfo,
	start, end int64,
) (*tempopb.SearchResponse, error) {
	// Execute terminal phase (phase 0)
	terminal := linkChain[0]
	terminalQuery := buildQueryFromExpression(terminal.Conditions)

	if cmd.Verbose {
		level.Debug(logger).Log("msg", "Executing terminal phase", "phase", 0, "query", terminalQuery)
	}

	terminalResp, err := cmd.executePhase(ctx, terminalQuery, start, end, cmd.MaxSpanIDsPerPhase)
	if err != nil {
		return nil, fmt.Errorf("terminal phase failed: %w", err)
	}

	// Extract span IDs from terminal phase
	terminalTraces := terminalResp.Traces
	spanIDs := extractSpanIDsFromTraces(terminalTraces)

	if cmd.Verbose {
		level.Info(logger).Log("msg", "Terminal phase complete",
			"spanIDs", len(spanIDs),
			"traces", len(terminalTraces))
	}

	// If only one phase, return terminal results
	if len(linkChain) == 1 {
		if cmd.Limit > 0 {
			terminalResp.Traces, _ = applyTraceLimit(terminalResp.Traces, uint32(cmd.Limit))
		}
		return terminalResp, nil
	}

	// If no span IDs found, return empty
	if len(spanIDs) == 0 {
		return &tempopb.SearchResponse{Metrics: terminalResp.Metrics}, nil
	}

	// Execute remaining phases and collect all results
	allTracesByPhase := [][]*tempopb.TraceSearchMetadata{terminalTraces}
	allSpanIDsByPhase := [][]string{spanIDs}
	totalMetrics := terminalResp.Metrics

	for i := 1; i < len(linkChain); i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		phase := linkChain[i]

		// Build query with link:spanID filter
		phaseQuery, ok := buildLinkFilterQuery(phase.Conditions, spanIDs, cmd.MaxSpanIDsPerPhase)
		if !ok {
			if cmd.Verbose {
				level.Warn(logger).Log("msg", "No valid span IDs for phase",
					"phase", i,
					"inputSpanIDs", len(spanIDs))
			}
			return &tempopb.SearchResponse{Metrics: totalMetrics}, nil
		}

		if cmd.Verbose {
			level.Debug(logger).Log("msg", "Executing link phase",
				"phase", i,
				"query", phaseQuery,
				"inputSpanIDs", len(spanIDs))
		}

		// Execute this phase
		phaseResp, err := cmd.executePhase(ctx, phaseQuery, start, end, cmd.MaxSpanIDsPerPhase)
		if err != nil {
			return nil, fmt.Errorf("phase %d failed: %w", i, err)
		}

		// Extract span IDs for next phase
		phaseTraces := phaseResp.Traces
		nextSpanIDs := extractSpanIDsFromTraces(phaseTraces)

		// Merge metrics
		mergeSearchMetrics(totalMetrics, phaseResp.Metrics)

		if cmd.Verbose {
			level.Info(logger).Log("msg", "Link phase complete",
				"phase", i,
				"spanIDs", len(nextSpanIDs),
				"traces", len(phaseTraces))
		}

		// If any phase returns no results, the chain is incomplete
		if len(nextSpanIDs) == 0 {
			if cmd.Verbose {
				level.Info(logger).Log("msg", "Incomplete link chain, returning empty results",
					"phase", i)
			}
			return &tempopb.SearchResponse{Metrics: totalMetrics}, nil
		}

		// Collect this phase's results
		allTracesByPhase = append(allTracesByPhase, phaseTraces)
		allSpanIDsByPhase = append(allSpanIDsByPhase, nextSpanIDs)
		spanIDs = nextSpanIDs
	}

	// All phases completed - filter for complete chains
	if cmd.Verbose {
		level.Info(logger).Log("msg", "All phases complete, filtering for complete chains")
	}

	completeTraces := cmd.filterCompleteChains(allTracesByPhase, allSpanIDsByPhase, linkChain)

	limitedTraces := completeTraces
	truncated := false
	if cmd.Limit > 0 {
		limitedTraces, truncated = applyTraceLimit(completeTraces, uint32(cmd.Limit))
	}

	if cmd.Verbose {
		level.Info(logger).Log("msg", "Complete chains found",
			"traces", len(limitedTraces),
			"truncated", truncated)
	}

	return &tempopb.SearchResponse{
		Traces:  limitedTraces,
		Metrics: totalMetrics,
	}, nil
}

// nolint: goconst // goconst wants us to make http:// a const
// executePhase executes a single phase query via HTTP
func (cmd *queryAPILinkSearchCmd) executePhase(
	ctx context.Context,
	query string,
	start, end int64,
	limit int,
) (*tempopb.SearchResponse, error) {
	// Build search request
	req := &tempopb.SearchRequest{
		Query: query,
		Start: uint32(start),
		End:   uint32(end),
		Limit: uint32(limit),
	}

	// Build HTTP request URL
	scheme := httpScheme(cmd.Secure)
	httpURL := scheme + "://" + path.Join(cmd.HostPort, cmd.PathPrefix, api.PathSearch)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", httpURL, nil)
	if err != nil {
		return nil, err
	}

	httpReq, err = api.BuildSearchRequest(httpReq, req)
	if err != nil {
		return nil, err
	}

	// Set orgID header
	httpReq.Header = http.Header{}
	err = user.InjectOrgIDIntoHTTPRequest(user.InjectOrgID(ctx, cmd.OrgID), httpReq)
	if err != nil {
		return nil, err
	}

	// Execute HTTP request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	// Read response body
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	// Check status
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to query. body: %s status: %s", string(body), httpResp.Status)
	}

	// Parse response
	resp := &tempopb.SearchResponse{}
	if err := jsonpb.Unmarshal(bytes.NewReader(body), resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// filterCompleteChains ensures only traces with complete link chains are returned
// Adapted from modules/frontend/search_sharder.go:653-797
func (cmd *queryAPILinkSearchCmd) filterCompleteChains(
	tracesByPhase [][]*tempopb.TraceSearchMetadata,
	spanIDsByPhase [][]string,
	linkChain []*traceql.LinkOperationInfo,
) []*tempopb.TraceSearchMetadata {
	// Indexed by [phaseIdx][traceID]
	allTraceInfosByPhase := make([]map[string]*traceInfo, len(tracesByPhase))

	for phaseIdx, phaseTraces := range tracesByPhase {
		allTraceInfosByPhase[phaseIdx] = make(map[string]*traceInfo)
		for _, trace := range phaseTraces {
			info := &traceInfo{
				trace:       trace,
				spanIDs:     make(map[string]struct{}),
				linkTargets: make(map[string]struct{}),
			}
			allTraceInfosByPhase[phaseIdx][trace.TraceID] = info

			forEachSpanInTrace(trace, func(span *tempopb.Span) {
				if span.SpanID != "" {
					info.spanIDs[span.SpanID] = struct{}{}
				}
				for _, attr := range span.Attributes {
					if attr.Key == "link:spanID" && attr.Value != nil {
						targetID := attr.Value.GetStringValue()
						if targetID != "" {
							info.linkTargets[targetID] = struct{}{}
						}
					}
				}
			})
		}
	}

	// Convert spanIDsByPhase to sets for quick lookup
	validSpanIDsByPhase := make([]map[string]struct{}, len(spanIDsByPhase))
	for i, ids := range spanIDsByPhase {
		validSpanIDsByPhase[i] = make(map[string]struct{})
		for _, id := range ids {
			validSpanIDsByPhase[i][id] = struct{}{}
		}
	}

	// Build reverse mapping: which traces link to which span ID
	linksToSpanID := make(map[string][]int) // targetSpanID -> []phaseIdx
	for phaseIdx, infos := range allTraceInfosByPhase {
		for _, info := range infos {
			for targetID := range info.linkTargets {
				linksToSpanID[targetID] = append(linksToSpanID[targetID], phaseIdx)
			}
		}
	}

	// Filter traces to only include complete chains
	type phaseKey struct {
		traceID string
		phase   int
	}
	completePhaseTraces := make(map[phaseKey]*tempopb.TraceSearchMetadata)
	completeTraceIDs := make(map[string]struct{})

	for phaseIdx, infos := range allTraceInfosByPhase {
		for traceID, info := range infos {
			if isCompleteChain(info, phaseIdx, validSpanIDsByPhase, linksToSpanID) {
				key := phaseKey{traceID, phaseIdx}
				completePhaseTraces[key] = info.trace
				completeTraceIDs[traceID] = struct{}{}
			}
		}
	}

	// Determine which phases are included based on union operators
	includedPhases := getIncludedPhases(linkChain)

	// Merge spans for complete traces from included phases
	mergedTraces := make(map[string]*tempopb.TraceSearchMetadata)
	for traceID := range completeTraceIDs {
		var mergedTrace *tempopb.TraceSearchMetadata

		for phaseIdx := range len(tracesByPhase) {
			if _, isIncluded := includedPhases[phaseIdx]; !isIncluded {
				continue
			}

			key := phaseKey{traceID, phaseIdx}
			if trace, ok := completePhaseTraces[key]; ok {
				if mergedTrace == nil {
					// Initialize merged trace with metadata from this trace
					mergedTrace = &tempopb.TraceSearchMetadata{
						TraceID:           trace.TraceID,
						RootServiceName:   trace.RootServiceName,
						RootTraceName:     trace.RootTraceName,
						StartTimeUnixNano: trace.StartTimeUnixNano,
						DurationMs:        trace.DurationMs,
					}
				}

				// Merge spans from this phase version of the trace
				mergedTrace.SpanSets = append(mergedTrace.SpanSets, trace.SpanSets...)
				if trace.SpanSet != nil {
					if mergedTrace.SpanSet == nil {
						mergedTrace.SpanSet = &tempopb.SpanSet{}
					}
					mergedTrace.SpanSet.Spans = append(mergedTrace.SpanSet.Spans, trace.SpanSet.Spans...)
				}
			}
		}

		if mergedTrace != nil {
			mergedTraces[traceID] = mergedTrace
		}
	}

	completeTraces := make([]*tempopb.TraceSearchMetadata, 0, len(mergedTraces))
	for _, mergedTrace := range mergedTraces {
		// Deduplicate spans within each trace
		deduplicateSpansInTrace(mergedTrace)
		completeTraces = append(completeTraces, mergedTrace)
	}

	return completeTraces
}

// ========== Helper Functions ==========
// NOTE: These are temporarily copied from modules/frontend/search_sharder.go
// for this experimental feature.

// buildLinkFilterQuery builds a query with link:spanID regex filter
// Copied from modules/frontend/search_sharder.go:1172
func buildLinkFilterQuery(conditions traceql.SpansetExpression, spanIDs []string, maxSpanIDs int) (string, bool) {
	// Limit span IDs
	limitedIDs := make([]string, 0, len(spanIDs))
	for _, id := range spanIDs {
		if !isValidSpanIDString(id) {
			continue
		}
		limitedIDs = append(limitedIDs, id)
		if maxSpanIDs > 0 && len(limitedIDs) >= maxSpanIDs {
			break
		}
	}
	if len(limitedIDs) == 0 {
		return "", false
	}

	// Build regex pattern: "(id1|id2|id3)"
	regexPattern := "(" + strings.Join(limitedIDs, "|") + ")"

	// Get conditions as string and strip outer braces for embedding
	conditionsStr := buildQueryFromExpression(conditions)
	conditionsStr = strings.TrimPrefix(conditionsStr, "{")
	conditionsStr = strings.TrimSuffix(conditionsStr, "}")
	conditionsStr = strings.TrimSpace(conditionsStr)

	// If conditions are empty, just use link filter
	if conditionsStr == "" || conditionsStr == "true" {
		return fmt.Sprintf(`{ link:spanID =~ "%s" }`, regexPattern), true
	}

	// Combine: { link:spanID =~ "pattern" && conditions }
	return fmt.Sprintf(`{ link:spanID =~ "%s" && %s }`, regexPattern, conditionsStr), true
}

// buildQueryFromExpression converts a SpansetExpression to query string
// Copied from modules/frontend/search_sharder.go:1209
func buildQueryFromExpression(expr traceql.SpansetExpression) string {
	if expr == nil {
		return ""
	}
	return strings.TrimSpace(expr.String())
}

// isValidSpanIDString validates that a span ID string is exactly 16 hex characters
// Copied from modules/frontend/search_sharder.go:1283
func isValidSpanIDString(id string) bool {
	if len(id) != 16 {
		return false
	}
	_, err := util.HexStringToSpanID(id)
	return err == nil
}

// forEachSpanInTrace iterates over all spans in a trace
// Copied from modules/frontend/search_sharder.go:1293
func forEachSpanInTrace(trace *tempopb.TraceSearchMetadata, fn func(*tempopb.Span)) {
	if trace == nil {
		return
	}

	for _, spanSet := range trace.SpanSets {
		for _, span := range spanSet.Spans {
			fn(span)
		}
	}

	if trace.SpanSet != nil {
		for _, span := range trace.SpanSet.Spans {
			fn(span)
		}
	}
}

// extractSpanIDsFromTraces extracts all unique valid span IDs from traces
func extractSpanIDsFromTraces(traces []*tempopb.TraceSearchMetadata) []string {
	spanIDSet := make(map[string]struct{})
	for _, trace := range traces {
		forEachSpanInTrace(trace, func(span *tempopb.Span) {
			if span.SpanID != "" && isValidSpanIDString(span.SpanID) {
				spanIDSet[span.SpanID] = struct{}{}
			}
		})
	}

	spanIDs := make([]string, 0, len(spanIDSet))
	for id := range spanIDSet {
		spanIDs = append(spanIDs, id)
	}
	return spanIDs
}

// applyTraceLimit limits the number of traces
// Copied from modules/frontend/search_sharder.go:1311
func applyTraceLimit(traces []*tempopb.TraceSearchMetadata, limit uint32) ([]*tempopb.TraceSearchMetadata, bool) {
	if limit == 0 || uint32(len(traces)) <= limit {
		return traces, false
	}
	return traces[:limit], true
}

// mergeSearchMetrics combines metrics from multiple phases
// Copied from modules/frontend/search_sharder.go:1345
func mergeSearchMetrics(total, phase *tempopb.SearchMetrics) {
	if phase == nil || total == nil {
		return
	}
	total.InspectedTraces += phase.InspectedTraces
	total.InspectedBytes += phase.InspectedBytes
	total.InspectedSpans += phase.InspectedSpans
	total.TotalBlocks += phase.TotalBlocks
	total.TotalJobs += phase.TotalJobs
	total.TotalBlockBytes += phase.TotalBlockBytes
	total.CompletedJobs += phase.CompletedJobs
}

// traceInfo stores pre-extracted metadata for efficient chain validation
// Copied from modules/frontend/search_sharder.go:800
type traceInfo struct {
	trace       *tempopb.TraceSearchMetadata
	spanIDs     map[string]struct{}
	linkTargets map[string]struct{}
}

// isCompleteChain checks if a trace is part of a complete link chain
// Copied from modules/frontend/search_sharder.go:807
func isCompleteChain(
	info *traceInfo,
	tracePhase int,
	validSpanIDsByPhase []map[string]struct{},
	linksToSpanID map[string][]int,
) bool {
	// Check link to previous phase (if not terminal)
	if tracePhase > 0 {
		foundLinkToPrev := false
		for targetID := range info.linkTargets {
			if _, ok := validSpanIDsByPhase[tracePhase-1][targetID]; ok {
				foundLinkToPrev = true
				break
			}
		}

		if !foundLinkToPrev {
			return false
		}
	}

	// Check if something from the next phase links to us (if not last)
	if tracePhase < len(validSpanIDsByPhase)-1 {
		foundLinkFromNext := false
		for mySpanID := range info.spanIDs {
			// Check if any trace in the next phase links to this span ID
			if slices.Contains(linksToSpanID[mySpanID], tracePhase+1) {
				foundLinkFromNext = true
				break
			}
		}

		if !foundLinkFromNext {
			return false
		}
	}

	return true
}

// getIncludedPhases determines which phases should be included in the final output
// Copied from modules/frontend/search_sharder.go:854
func getIncludedPhases(linkChain []*traceql.LinkOperationInfo) map[int]struct{} {
	included := make(map[int]struct{})
	included[0] = struct{}{} // Terminal phase (Phase 0) is always included

	if len(linkChain) < 2 {
		return included
	}

	// Determine if the chain was reversed (LinkTo ->>)
	isReversed := linkChain[0].IsLinkTo

	for i := 1; i < len(linkChain); i++ {
		isUnion := false
		if isReversed {
			isUnion = linkChain[i-1].IsUnion
		} else {
			isUnion = linkChain[i].IsUnion
		}

		// A phase is included if its predecessor was included AND the link is a union
		if _, ok := included[i-1]; ok && isUnion {
			included[i] = struct{}{}
		} else {
			// Once a link is not a union, subsequent phases are not included
			break
		}
	}

	return included
}

// deduplicateSpansInTrace ensures each span ID appears only once in the trace metadata
// Copied from modules/frontend/search_sharder.go:897
func deduplicateSpansInTrace(trace *tempopb.TraceSearchMetadata) {
	if trace == nil {
		return
	}

	seen := make(map[string]struct{})

	if trace.SpanSet != nil {
		newSpans := make([]*tempopb.Span, 0, len(trace.SpanSet.Spans))
		for _, span := range trace.SpanSet.Spans {
			if span.SpanID == "" {
				continue
			}
			if _, ok := seen[span.SpanID]; !ok {
				seen[span.SpanID] = struct{}{}
				newSpans = append(newSpans, span)
			}
		}
		if len(newSpans) > 0 {
			trace.SpanSet.Spans = newSpans
		} else {
			trace.SpanSet = nil
		}
	}

	var newSpanSets []*tempopb.SpanSet
	for _, ss := range trace.SpanSets {
		if ss == nil {
			continue
		}
		newSpans := make([]*tempopb.Span, 0, len(ss.Spans))
		for _, span := range ss.Spans {
			if span.SpanID == "" {
				continue
			}
			if _, ok := seen[span.SpanID]; !ok {
				seen[span.SpanID] = struct{}{}
				newSpans = append(newSpans, span)
			}
		}
		if len(newSpans) > 0 {
			ss.Spans = newSpans
			newSpanSets = append(newSpanSets, ss)
		}
	}
	trace.SpanSets = newSpanSets
}
