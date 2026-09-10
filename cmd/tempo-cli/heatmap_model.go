package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"

	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const debounceDelay = 200 * time.Millisecond

// Layout constants shared between recomputeGrid (which sizes the grids to fit) and
// handleMouse (which has to know exactly where they land on screen to hit-test a click or
// hover). Keeping these in one place means the two can never drift out of sync.
const (
	cellWidth       = 2                   // characters per rendered grid cell
	gapChars        = 3                   // characters between the span and io columns
	linesBeforeGrid = 5                   // title(1) + query box(3) + column header(1)
	reservedLines   = linesBeforeGrid + 1 // + footer(1)
)

// rowNumbered is implemented by vparquet5 spans returned from Fetch/FetchSpans. It exposes
// where a span physically sits in the block file, which is how this tool maps a match onto
// the span-location grid. Blocks in other encodings don't implement it, so those spans are
// simply not counted.
type rowNumbered interface {
	RowNumber() parquetquery.RowNumber
}

// heatGrid is one density map: a fixed number of cells, each covering an equal-sized range of
// some 1-D domain (span row numbers on the left, file byte offsets on the right), scaled so
// the whole domain fits in the current terminal regardless of how large it is.
type heatGrid struct {
	cols, rows   int
	unitsPerCell int64
	counts       []int64
	maxCount     int64
}

func newHeatGrid(cols, rows int, totalUnits int64) heatGrid {
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	cells := int64(cols) * int64(rows)

	unitsPerCell := totalUnits / cells
	if totalUnits%cells != 0 {
		unitsPerCell++
	}
	if unitsPerCell < 1 {
		unitsPerCell = 1
	}

	return heatGrid{cols: cols, rows: rows, unitsPerCell: unitsPerCell, counts: make([]int64, cells)}
}

func (g *heatGrid) reset() {
	g.counts = make([]int64, len(g.counts))
	g.maxCount = 0
}

func (g *heatGrid) apply(counts []int64) {
	g.counts = counts
	g.maxCount = 0
	for _, c := range counts {
		if c > g.maxCount {
			g.maxCount = c
		}
	}
}

// heatmapModel is a bubbletea model showing two side-by-side density maps for the current
// TraceQL query against a vParquet5 block: on the left, where matching spans physically sit
// in the block (by row number); on the right, where in the block file the query actually read
// bytes from. Both are single, non-scrolling screens no matter how large the block is - cell
// size is recomputed to fit whatever terminal size is current.
type heatmapModel struct {
	reader     backend.Reader
	meta       *backend.BlockMeta
	searchOpts common.SearchOptions

	program *tea.Program

	input  []rune
	cursor int

	width, height int
	ready         bool

	spanGrid heatGrid
	ioGrid   heatGrid
	ioStats  ioStats

	// rowGroupEnds/rowGroupByteEnds are the cumulative row count and cumulative file byte
	// offset through the end of each row group, from rowGroupBoundaries. boundaryCells and
	// ioBoundaryCells mark, per spanGrid/ioGrid cell, whether a row group boundary falls
	// somewhere inside that cell's range; both are rebuilt whenever the grid is resized.
	rowGroupEnds     []int64
	rowGroupByteEnds []int64
	boundaryCells    []bool
	ioBoundaryCells  []bool

	matched     int64
	scanning    bool
	metricsMode bool
	errText     string

	// hover describes whichever grid cell the mouse last moved over or clicked, so it can be
	// drawn as a tooltip right next to that cell rather than in a status line that might be
	// on the opposite side of the screen. hoverActive is false when the mouse is outside
	// both grids.
	hoverActive     bool
	hoverOnSpanGrid bool
	hoverRow        int
	hoverCol        int
	hoverText       string

	inputGen int // bumped on every keystroke, invalidates pending debounce timers
	fetchGen int // bumped on every fetch that actually starts, tags in-flight results
	cancel   context.CancelFunc
}

// defaultQuery pre-fills the query box on startup so the heatmap has something to show
// immediately rather than starting blank.
const defaultQuery = `{resource.service.name="tempo-querier"}`

func newHeatmapModel(reader backend.Reader, meta *backend.BlockMeta, opts common.SearchOptions, rowGroupEnds, rowGroupByteEnds []int64) *heatmapModel {
	input := []rune(defaultQuery)
	return &heatmapModel{
		reader:           reader,
		meta:             meta,
		searchOpts:       opts,
		rowGroupEnds:     rowGroupEnds,
		rowGroupByteEnds: rowGroupByteEnds,
		input:            input,
		cursor:           len(input),
	}
}

func (m *heatmapModel) Init() tea.Cmd {
	return nil
}

type debounceMsg struct{ gen int }

type fetchProgressMsg struct {
	gen      int
	counts   []int64
	ioCounts []int64
	ioStats  ioStats
	matched  int64
}

type fetchDoneMsg struct {
	gen      int
	counts   []int64
	ioCounts []int64
	ioStats  ioStats
	matched  int64
	err      error
}

func (m *heatmapModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.recomputeGrid()
		return m, m.triggerFetch()

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		m.handleMouse(msg)
		return m, nil

	case debounceMsg:
		if msg.gen != m.inputGen {
			return m, nil // stale, a newer keystroke has already superseded this timer
		}
		return m, m.triggerFetch()

	case fetchProgressMsg:
		if msg.gen != m.fetchGen {
			return m, nil // a slower, now-abandoned fetch; ignore it
		}
		m.applyCounts(msg.counts, msg.ioCounts, msg.ioStats, msg.matched)
		return m, nil

	case fetchDoneMsg:
		if msg.gen != m.fetchGen {
			return m, nil
		}
		m.scanning = false
		if msg.err != nil {
			// A canceled fetch has no valid data (its counts/ioCounts are zero-valued, not
			// "actually empty"), so applyCounts must never run for it - only the error
			// display is conditional, not whether we touch the grids at all.
			if !errors.Is(msg.err, context.Canceled) {
				m.errText = msg.err.Error()
			}
			return m, nil
		}
		m.applyCounts(msg.counts, msg.ioCounts, msg.ioStats, msg.matched)
		return m, nil
	}

	return m, nil
}

func (m *heatmapModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyBackspace:
		if m.cursor > 0 {
			m.input = append(m.input[:m.cursor-1], m.input[m.cursor:]...)
			m.cursor--
		}
	case tea.KeyDelete:
		if m.cursor < len(m.input) {
			m.input = append(m.input[:m.cursor], m.input[m.cursor+1:]...)
		}
	case tea.KeyLeft:
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case tea.KeyRight:
		if m.cursor < len(m.input) {
			m.cursor++
		}
		return m, nil
	case tea.KeyHome:
		m.cursor = 0
		return m, nil
	case tea.KeyEnd:
		m.cursor = len(m.input)
		return m, nil
	case tea.KeyCtrlU:
		m.input = nil
		m.cursor = 0
	case tea.KeyRunes, tea.KeySpace:
		runes := msg.Runes
		if msg.Type == tea.KeySpace {
			runes = []rune{' '}
		}
		m.input = append(m.input[:m.cursor], append(append([]rune{}, runes...), m.input[m.cursor:]...)...)
		m.cursor += len(runes)
	default:
		return m, nil
	}

	m.inputGen++
	gen := m.inputGen
	return m, tea.Tick(debounceDelay, func(time.Time) tea.Msg {
		return debounceMsg{gen: gen}
	})
}

// handleMouse updates the hover state to describe whichever grid cell the mouse is currently
// over - on any mouse event, motion or click alike, so hovering and clicking both work the
// same way. It mirrors the exact layout renderHeaders/renderGrids use (linesBeforeGrid,
// cellWidth, gapChars) to translate the event's terminal coordinates back into a cell index.
// The actual tooltip is drawn later, by renderGrids, right next to the cell it describes -
// see overlayTooltip - rather than in a status line that could be anywhere on screen.
func (m *heatmapModel) handleMouse(ev tea.MouseMsg) {
	row := ev.Y - linesBeforeGrid
	leftWidth := m.spanGrid.cols * cellWidth
	rightStart := leftWidth + gapChars

	m.hoverActive = false

	switch {
	case row < 0:
		// above the grid entirely

	case ev.X < leftWidth:
		col := ev.X / cellWidth
		if idx, ok := cellIndex(&m.spanGrid, row, col); ok && m.spanGrid.counts[idx] > 0 {
			m.hoverActive = true
			m.hoverOnSpanGrid = true
			m.hoverRow, m.hoverCol = row, col
			m.hoverText = humanize.Comma(m.spanGrid.counts[idx]) + " matches"
		}

	case ev.X >= rightStart:
		col := (ev.X - rightStart) / cellWidth
		if idx, ok := cellIndex(&m.ioGrid, row, col); ok && m.ioGrid.counts[idx] > 0 {
			m.hoverActive = true
			m.hoverOnSpanGrid = false
			m.hoverRow, m.hoverCol = row, col
			m.hoverText = humanize.Bytes(uint64(m.ioGrid.counts[idx])) + " read"
		}
	}
}

// cellIndex converts a (row, col) position within g into a counts index, or ok=false if it
// falls outside the grid (including the ragged last row, which can be shorter than g.cols).
func cellIndex(g *heatGrid, row, col int) (idx int, ok bool) {
	if row < 0 || row >= g.rows || col < 0 || col >= g.cols {
		return 0, false
	}
	idx = row*g.cols + col
	if idx < 0 || idx >= len(g.counts) {
		return 0, false
	}
	return idx, true
}

// recomputeGrid sizes both density maps to the current terminal, reserving a few lines for
// the header, query box, status line and footer, and splitting the width in two with a small
// gap between them. Each cell is rendered 2 characters wide so cells read as roughly square
// despite terminal characters being taller than they are wide.
func (m *heatmapModel) recomputeGrid() {
	rows := m.height - reservedLines
	if rows < 1 {
		rows = 1
	}

	usableChars := m.width - gapChars
	if usableChars < cellWidth*2 {
		usableChars = cellWidth * 2
	}
	leftChars := usableChars / 2
	rightChars := usableChars - leftChars

	m.spanGrid = newHeatGrid(leftChars/cellWidth, rows, m.meta.TotalObjects)
	m.ioGrid = newHeatGrid(rightChars/cellWidth, rows, int64(m.meta.Size_))
	m.boundaryCells = boundaryCellsFor(m.rowGroupEnds, m.meta.TotalObjects, m.spanGrid.unitsPerCell, int64(len(m.spanGrid.counts)))
	m.ioBoundaryCells = boundaryCellsFor(m.rowGroupByteEnds, int64(m.meta.Size_), m.ioGrid.unitsPerCell, int64(len(m.ioGrid.counts)))
	m.matched = 0
}

// boundaryCellsFor marks which cells a row group boundary falls into, given a grid's domain
// (ends, totalUnits, unitsPerCell, cells) - the same shape whether that domain is trace row
// numbers (spanGrid) or file byte offsets (ioGrid). The boundary at the very end of the
// domain isn't marked since there's no split to show there.
func boundaryCellsFor(ends []int64, totalUnits, unitsPerCell, cells int64) []bool {
	marked := make([]bool, cells)
	for _, end := range ends {
		if end <= 0 || end >= totalUnits {
			continue
		}
		if idx := end / unitsPerCell; idx >= 0 && idx < cells {
			marked[idx] = true
		}
	}
	return marked
}

// isMetricsQuery reports whether expr is a metrics query (uses a batch/series processor like
// rate(), count_over_time(), by(...), ...). traceql.Compile silently drops anything past the
// plain spanset filter for these - it just returns the filter's own Pipeline - so without this
// check a query like "{ } | rate()" would quietly run as if the user had typed "{ }": a
// full-block scan with no indication the aggregation was ignored. Metrics queries get their
// own execution path via runHeatmapMetricsFetch instead.
func isMetricsQuery(expr *traceql.RootExpr) bool {
	return len(expr.SeriesProcessor) > 0 || len(expr.BatchSpanProcessor) > 0
}

// needsFullTraceQuery reports whether expr is a structural or spanset-aggregate query (e.g.
// {a}>>{b}, | count()) that operates across multiple spans in a trace at once rather than
// filtering spans individually. Those don't fit this tool's per-span visualization model, so
// they're rejected rather than silently showing a confusing "0 matches".
func needsFullTraceQuery(expr *traceql.RootExpr) bool {
	pipeline, ok := expr.SinglePipeline()
	return ok && traceql.NeedsFullTrace(pipeline)
}

func (m *heatmapModel) applyCounts(spanCounts, ioCounts []int64, stats ioStats, matched int64) {
	m.matched = matched
	m.spanGrid.apply(spanCounts)
	m.ioGrid.apply(ioCounts)
	m.ioStats = stats
}

// triggerFetch parses the current query and, if valid, cancels any in-flight fetch and starts
// a new one. An unparsable query (which is the normal state of things while typing) leaves
// whatever was last successfully rendered on screen untouched, aside from the error message.
func (m *heatmapModel) triggerFetch() tea.Cmd {
	query := strings.TrimSpace(string(m.input))

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}

	// Bump fetchGen unconditionally, before we even know whether the new query is valid.
	// This invalidates every message still in flight from whatever fetch we just canceled -
	// including its eventual fetchDoneMsg, which carries no real data (its counts/ioCounts
	// are zero-valued) - so a canceled fetch can never be mistaken for the still-current one
	// and clobber what's on screen with that empty payload. It's also what lets every early
	// return below leave scanning=false rather than stuck showing "scanning..." forever for a
	// fetch that's already been abandoned.
	m.fetchGen++
	m.scanning = false

	if query == "" {
		m.errText = ""
		m.metricsMode = false
		m.matched = 0
		m.spanGrid.reset()
		m.ioGrid.reset()
		m.ioStats = ioStats{}
		return nil
	}

	expr, _, eval, fetchReq, err := traceql.Compile(query)
	if err != nil {
		m.errText = err.Error()
		return nil
	}
	metrics := isMetricsQuery(expr)
	if !metrics && needsFullTraceQuery(expr) {
		m.errText = "structural/aggregate queries (e.g. {a}>>{b}, | count()) aren't supported here: this tool visualizes individual span matches"
		return nil
	}

	m.errText = ""
	m.metricsMode = metrics
	m.scanning = true
	gen := m.fetchGen

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	reader := m.reader
	meta := m.meta
	opts := m.searchOpts
	prog := m.program
	spanCells, spanUnitsPerCell := int64(len(m.spanGrid.counts)), m.spanGrid.unitsPerCell
	ioCells, ioUnitsPerCell := int64(len(m.ioGrid.counts)), m.ioGrid.unitsPerCell

	// bubbletea already runs the returned Cmd in its own goroutine, so both fetch paths can
	// stream progress via prog.Send for as long as the scan takes before returning nil here.
	if metrics {
		return func() tea.Msg {
			runHeatmapMetricsFetch(ctx, prog, gen, reader, meta, query, opts, spanCells, spanUnitsPerCell, ioCells, ioUnitsPerCell)
			return nil
		}
	}

	fetchReq.SecondPass = func(s *traceql.Spanset) ([]*traceql.Spanset, error) {
		if s == nil || len(s.Spans) == 0 {
			return nil, nil
		}
		return eval([]*traceql.Spanset{s})
	}

	return func() tea.Msg {
		runHeatmapFetch(ctx, prog, gen, reader, meta, *fetchReq, opts, spanCells, spanUnitsPerCell, ioCells, ioUnitsPerCell)
		return nil
	}
}

// runHeatmapFetch streams matches from the block, bucketing each one by its row number, while
// an ioTrackingReader records the byte range of every read the fetch issues against the block
// file. It periodically reports partial progress back into the bubbletea program so both maps
// fill in live rather than freezing until an entire multi-million-span scan completes.
//
// The block is opened fresh here, on the tracked reader, rather than reusing one opened at
// startup: that's what lets each fetch's own file reads be attributed to it alone.
func runHeatmapFetch(
	ctx context.Context, prog *tea.Program, gen int,
	reader backend.Reader, meta *backend.BlockMeta, req traceql.FetchSpansRequest, opts common.SearchOptions,
	spanCells, spanUnitsPerCell, ioCells, ioUnitsPerCell int64,
) {
	io := newIOTracker(ioCells, ioUnitsPerCell)
	tracked := &ioTrackingReader{Reader: reader, onRead: io.record}

	block, err := encoding.OpenBlock(meta, tracked)
	if err != nil {
		prog.Send(fetchDoneMsg{gen: gen, err: err})
		return
	}

	resp, err := block.FetchSpans(ctx, req, opts)
	if err != nil {
		prog.Send(fetchDoneMsg{gen: gen, err: err})
		return
	}
	defer resp.Results.Close()

	rec := newSpanRecorder(prog, gen, io, spanCells, spanUnitsPerCell)

	for {
		sp, err := resp.Results.Next(ctx)
		if err != nil {
			rec.done(err)
			return
		}
		if sp == nil {
			break
		}
		rec.record(sp)
	}

	rec.done(nil)
}

// runHeatmapMetricsFetch handles metrics queries (rate(), count_over_time(), by(...), ...),
// which traceql.Compile can't express as a plain span filter + eval. Instead it drives the
// real metrics engine (the same machinery the query-range API uses) against this block, and
// installs a traceql.SpanWatcher that gets called with every span the query actually matches
// - exactly the same per-span row-number bucketing runHeatmapFetch does, just fed by the
// watcher hook rather than iterating the fetch results directly. The metrics engine's own
// aggregation result (rate/count/etc.) is discarded; only which spans it touched matters here.
func runHeatmapMetricsFetch(
	ctx context.Context, prog *tea.Program, gen int,
	reader backend.Reader, meta *backend.BlockMeta, query string, opts common.SearchOptions,
	spanCells, spanUnitsPerCell, ioCells, ioUnitsPerCell int64,
) {
	io := newIOTracker(ioCells, ioUnitsPerCell)
	tracked := &ioTrackingReader{Reader: reader, onRead: io.record}

	block, err := encoding.OpenBlock(meta, tracked)
	if err != nil {
		prog.Send(fetchDoneMsg{gen: gen, err: err})
		return
	}

	rec := newSpanRecorder(prog, gen, io, spanCells, spanUnitsPerCell)
	watcher := &rowWatcher{onSpan: rec.record}

	// The metrics engine requires a nonzero [start,end) window and drops any span outside
	// it, so pad generously around the block's own recorded range rather than leave it
	// unbounded like the plain search path does - every real span must fall well inside it.
	const pad = time.Hour
	start := uint64(meta.StartTime.Add(-pad).UnixNano())
	end := uint64(meta.EndTime.Add(pad).UnixNano())

	evaluator, err := traceql.NewEngine().CompileMetricsQueryRange(&tempopb.QueryRangeRequest{
		Query: query,
		Start: start,
		End:   end,
		Step:  end - start,
	}, traceql.WithWatchers(watcher))
	if err != nil {
		prog.Send(fetchDoneMsg{gen: gen, err: err})
		return
	}

	fetcher := spansetFetcherAdapter{block: block, opts: opts}
	fetcherStart := uint64(meta.StartTime.UnixNano())
	fetcherEnd := uint64(meta.EndTime.UnixNano())

	if err := evaluator.Do(ctx, fetcher, fetcherStart, fetcherEnd, heatmapMaxSeries); err != nil {
		rec.done(err)
		return
	}

	rec.done(nil)
}

// heatmapMaxSeries caps the number of distinct series a metrics query's grouping (e.g.
// by(resource.service.name)) can create, as a safety limit against unbounded cardinality.
// It's generous since only span locations matter here, not the actual aggregated result.
const heatmapMaxSeries = 1000

// spanRecorder buckets matched spans by row number into spanCounts and periodically streams
// progress - alongside a live snapshot of the io tracker's byte counts - back to the
// bubbletea program. Shared by the plain search and metrics fetch paths so both fill the
// grids in exactly the same way.
type spanRecorder struct {
	prog             *tea.Program
	gen              int
	io               *ioTracker
	spanCounts       []int64
	spanUnitsPerCell int64
	spanCells        int64
	matched          int64
	lastSent         time.Time
}

func newSpanRecorder(prog *tea.Program, gen int, io *ioTracker, spanCells, spanUnitsPerCell int64) *spanRecorder {
	return &spanRecorder{
		prog:             prog,
		gen:              gen,
		io:               io,
		spanCounts:       make([]int64, spanCells),
		spanUnitsPerCell: spanUnitsPerCell,
		spanCells:        spanCells,
		lastSent:         time.Now(),
	}
}

func (r *spanRecorder) record(sp traceql.Span) {
	r.matched++
	if rn, ok := sp.(rowNumbered); ok {
		if cell := int64(rn.RowNumber()[0]) / r.spanUnitsPerCell; cell >= 0 && cell < r.spanCells {
			r.spanCounts[cell]++
		}
	}

	if r.matched%2000 == 0 && time.Since(r.lastSent) > 80*time.Millisecond {
		r.prog.Send(fetchProgressMsg{gen: r.gen, counts: append([]int64(nil), r.spanCounts...), ioCounts: r.io.snapshot(), ioStats: r.io.statsSnapshot(), matched: r.matched})
		r.lastSent = time.Now()
	}
}

func (r *spanRecorder) done(err error) {
	if err != nil {
		r.prog.Send(fetchDoneMsg{gen: r.gen, err: err})
		return
	}
	r.prog.Send(fetchDoneMsg{gen: r.gen, counts: r.spanCounts, ioCounts: r.io.snapshot(), ioStats: r.io.statsSnapshot(), matched: r.matched})
}

// rowWatcher is a traceql.SpanWatcher that forwards every span the metrics engine matches to
// onSpan. It never goes inactive: unlike the built-in watchers (which stop once they've seen
// what they're looking for), this one wants to see every single matched span for as long as
// the query runs.
type rowWatcher struct {
	onSpan func(traceql.Span)
}

func (w *rowWatcher) Conditions() []traceql.Condition { return nil }

func (w *rowWatcher) WatchSpan(s traceql.Span) bool {
	w.onSpan(s)
	return true
}

func (w *rowWatcher) Active() bool { return true }

func (w *rowWatcher) Stats() map[string]int64 { return nil }

// spansetFetcherAdapter adapts a common.BackendBlock (whose Fetch/FetchSpans take an extra
// common.SearchOptions argument) to traceql.SpansetFetcher, the interface the metrics engine
// evaluator expects.
type spansetFetcherAdapter struct {
	block common.BackendBlock
	opts  common.SearchOptions
}

func (a spansetFetcherAdapter) Fetch(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
	return a.block.Fetch(ctx, req, a.opts)
}

func (a spansetFetcherAdapter) FetchSpans(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansOnlyResponse, error) {
	return a.block.FetchSpans(ctx, req, a.opts)
}

// ioStats summarizes the individual ReadRange calls a fetch has made against the block file:
// how many there were, and the smallest/largest/average size. This is separate from
// ioTracker's per-cell histogram, which only tracks where bytes landed, not how they were
// grouped into reads.
type ioStats struct {
	Reads   int64
	MinSize int64
	MaxSize int64
	SumSize int64
}

func (s ioStats) avgSize() int64 {
	if s.Reads == 0 {
		return 0
	}
	return s.SumSize / s.Reads
}

// ioTracker accumulates, per file-byte-offset cell, how many bytes of that range a fetch has
// read, plus overall read-count/size stats. A read spanning multiple cells contributes to each
// cell proportional to its overlap, so summing all cells always equals the total bytes read.
// Reads can arrive concurrently (the block opens the file in async read mode), so access is
// serialized with a mutex.
type ioTracker struct {
	mu           sync.Mutex
	counts       []int64
	unitsPerCell int64
	stats        ioStats
}

func newIOTracker(cells, unitsPerCell int64) *ioTracker {
	return &ioTracker{counts: make([]int64, cells), unitsPerCell: unitsPerCell}
}

func (t *ioTracker) record(offset uint64, length int) {
	if length <= 0 || len(t.counts) == 0 {
		return
	}

	start := int64(offset)
	size := int64(length)
	end := start + size // exclusive
	startCell := start / t.unitsPerCell
	endCell := (end - 1) / t.unitsPerCell

	t.mu.Lock()
	defer t.mu.Unlock()

	t.stats.Reads++
	t.stats.SumSize += size
	if t.stats.Reads == 1 || size < t.stats.MinSize {
		t.stats.MinSize = size
	}
	if size > t.stats.MaxSize {
		t.stats.MaxSize = size
	}

	for cell := startCell; cell <= endCell; cell++ {
		if cell < 0 || cell >= int64(len(t.counts)) {
			continue
		}
		cellStart := cell * t.unitsPerCell
		cellEnd := cellStart + t.unitsPerCell

		overlapStart, overlapEnd := start, end
		if cellStart > overlapStart {
			overlapStart = cellStart
		}
		if cellEnd < overlapEnd {
			overlapEnd = cellEnd
		}
		if overlapEnd > overlapStart {
			t.counts[cell] += overlapEnd - overlapStart
		}
	}
}

func (t *ioTracker) snapshot() []int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]int64(nil), t.counts...)
}

func (t *ioTracker) statsSnapshot() ioStats {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.stats
}

// ioTrackingReader wraps a backend.Reader to record the byte range of every ReadRange call
// against the block's data file, which is how the heatmap knows where in the file a query
// actually read from. Embedding backend.Reader means every other method (listing blocks,
// tenants, reading other files, ...) just delegates straight through unchanged.
type ioTrackingReader struct {
	backend.Reader
	onRead func(offset uint64, length int)
}

func (r *ioTrackingReader) ReadRange(ctx context.Context, name string, blockID uuid.UUID, tenantID string, offset uint64, buffer []byte, cacheInfo *backend.CacheInfo) error {
	err := r.Reader.ReadRange(ctx, name, blockID, tenantID, offset, buffer, cacheInfo)
	if err == nil {
		r.onRead(offset, len(buffer))
	}
	return err
}

var heatLevels = []lipgloss.Style{
	lipgloss.NewStyle().Foreground(lipgloss.Color("237")), // empty
	lipgloss.NewStyle().Foreground(lipgloss.Color("24")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("31")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("37")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("108")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("178")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("202")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	headerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("246")).Bold(true)
	boxStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	footerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	boundaryStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
)

func (m *heatmapModel) View() string {
	if !m.ready {
		return "loading...\n"
	}

	var b strings.Builder

	title := fmt.Sprintf(
		"tempo-cli view heatmap  block=%s  tenant=%s  traces=%d  row groups=%d",
		m.meta.BlockID.String(), m.meta.TenantID, m.meta.TotalObjects, len(m.rowGroupEnds),
	)

	metricsNote := ""
	if m.metricsMode {
		metricsNote = "  [metrics query: showing matched span locations, aggregated result discarded]"
	}

	var suffix string
	switch {
	case m.errText != "":
		suffix = errStyle.Render("  error: " + m.errText)
	case m.scanning:
		suffix = statusStyle.Render("  scanning..." + metricsNote)
	default:
		suffix = statusStyle.Render(metricsNote)
	}
	fmt.Fprintf(&b, "%s%s\n", titleStyle.Render(title), suffix)

	before, after := string(m.input[:m.cursor]), string(m.input[m.cursor:])
	fmt.Fprintf(&b, "%s\n", boxStyle.Render("Query: "+before+"█"+after))

	b.WriteString(m.renderHeaders())
	b.WriteString(m.renderGrids())

	b.WriteString(footerStyle.Render("esc/ctrl+c: quit    e.g. {resource.service.name=\"tempo-gateway\"}    "))
	b.WriteString(boundaryStyle.Render("▕▏"))
	b.WriteString(footerStyle.Render(" = row group boundary"))

	return b.String()
}

// gridGap separates the span-location and I/O columns, both in the header and in the grids
// themselves, so the two stay aligned with each other.
var gridGap = strings.Repeat(" ", gapChars)

// renderHeaders packs each column's label and all its stats onto a single line, so the
// column header costs exactly one line no matter how much there is to report - leaving the
// rest of the screen for the heatmap itself.
func (m *heatmapModel) renderHeaders() string {
	leftWidth := m.spanGrid.cols * cellWidth
	rightWidth := m.ioGrid.cols * cellWidth

	left := fmt.Sprintf("SPAN LOCATIONS  %s matches  |  cell=%d rows", humanize.Comma(m.matched), m.spanGrid.unitsPerCell)
	right := fmt.Sprintf(
		"FILE I/O READS  %s read  |  cell=%s  |  %s",
		humanize.Bytes(uint64(m.ioStats.SumSize)), humanize.Bytes(uint64(m.ioGrid.unitsPerCell)), ioStatsSummary(m.ioStats),
	)

	return fmt.Sprintf(
		"%s%s%s\n",
		headerStyle.Render(fmt.Sprintf("%-*s", leftWidth, truncate(left, leftWidth))),
		gridGap,
		headerStyle.Render(fmt.Sprintf("%-*s", rightWidth, truncate(right, rightWidth))),
	)
}

// ioStatsSummary formats the read count and min/max/avg read size for the I/O column header.
func ioStatsSummary(s ioStats) string {
	if s.Reads == 0 {
		return "no reads yet"
	}
	return fmt.Sprintf(
		"%s reads  min %s  max %s  avg %s",
		humanize.Comma(s.Reads), humanize.Bytes(uint64(s.MinSize)), humanize.Bytes(uint64(s.MaxSize)), humanize.Bytes(uint64(s.avgSize())),
	)
}

// truncate keeps a header string from overflowing its column and bleeding into the next one
// on a narrow terminal, marking the cut with an ellipsis. Strings within width are unchanged
// (and %-*s below still right-pads them as normal).
func truncate(s string, width int) string {
	r := []rune(s)
	if width <= 0 || len(r) <= width {
		return s
	}
	if width == 1 {
		return string(r[:1])
	}
	return string(r[:width-1]) + "…"
}

// renderGrids draws both density maps. Each row is built as a slice of already-rendered,
// fixed-width (cellWidth-wide) cell strings rather than one big joined string, specifically so
// that when the mouse is hovering a cell on this row, overlayTooltip can splice a label
// directly into that slice - right next to the cell it describes - before the row is joined.
func (m *heatmapModel) renderGrids() string {
	var b strings.Builder
	rows := m.spanGrid.rows
	if m.ioGrid.rows > rows {
		rows = m.ioGrid.rows
	}

	for row := 0; row < rows; row++ {
		leftCells := gridRowCells(&m.spanGrid, row, m.boundaryCells)
		rightCells := gridRowCells(&m.ioGrid, row, m.ioBoundaryCells)

		if m.hoverActive && m.hoverRow == row {
			if m.hoverOnSpanGrid {
				overlayTooltip(leftCells, m.hoverCol, m.hoverText)
			} else {
				overlayTooltip(rightCells, m.hoverCol, m.hoverText)
			}
		}

		b.WriteString(strings.Join(leftCells, ""))
		b.WriteString(gridGap)
		b.WriteString(strings.Join(rightCells, ""))
		b.WriteString("\n")
	}
	return b.String()
}

// gridRowCells renders one row of g as a slice with one entry per cell.
func gridRowCells(g *heatGrid, row int, boundary []bool) []string {
	cells := make([]string, g.cols)
	for col := 0; col < g.cols; col++ {
		idx := row*g.cols + col
		switch {
		case idx >= len(g.counts):
			cells[col] = "  "
		case boundary != nil && idx < len(boundary) && boundary[idx]:
			cells[col] = boundaryStyle.Render("▕▏")
		default:
			cells[col] = heatLevels[levelFor(g.counts[idx], g.maxCount)].Render("██")
		}
	}
	return cells
}

// tooltipStyle marks both the hovered cell and its label distinctly from every heat color, so
// the tooltip reads clearly as an overlay rather than blending into the map.
var tooltipStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("220")).Bold(true)

// overlayTooltip splices text into cells right next to index col - to its right if there's
// room, otherwise to its left - so the reader never has to look away from the cursor to read
// it. The hovered cell itself is also highlighted, so it's unambiguous which cell the tooltip
// belongs to even once its neighbors are covered by the label.
func overlayTooltip(cells []string, col int, text string) {
	if col < 0 || col >= len(cells) {
		return
	}

	label := []rune(" " + text + " ")
	need := (len(label) + cellWidth - 1) / cellWidth // cells needed, rounding up

	start := col + 1
	if start+need > len(cells) {
		start = col - need
	}
	if start < 0 {
		start = 0
	}
	if start+need > len(cells) {
		need = len(cells) - start
	}

	for i := 0; i < need; i++ {
		lo, hi := i*cellWidth, i*cellWidth+cellWidth
		var chunk string
		if lo < len(label) {
			if hi > len(label) {
				hi = len(label)
			}
			chunk = string(label[lo:hi])
		}
		for len([]rune(chunk)) < cellWidth {
			chunk += " "
		}
		cells[start+i] = tooltipStyle.Render(chunk)
	}

	cells[col] = tooltipStyle.Render("██") // set last: guaranteed visible even if the label range overlapped it
}

// levelFor buckets a cell's count into a color level on a log scale, since density across a
// block is typically extremely uneven (most cells empty, a few very hot).
func levelFor(count, maxCount int64) int {
	if count <= 0 {
		return 0
	}
	if maxCount <= 1 {
		return len(heatLevels) - 1
	}

	logCount := math.Log(float64(count) + 1)
	logMax := math.Log(float64(maxCount) + 1)
	frac := logCount / logMax

	level := 1 + int(frac*float64(len(heatLevels)-2))
	if level >= len(heatLevels) {
		level = len(heatLevels) - 1
	}
	if level < 1 {
		level = 1
	}
	return level
}
