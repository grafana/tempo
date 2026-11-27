package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2" // nosemgrep: math-random-used -- a fixed seed keeps the report reproducible; nothing here is secret
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"time"

	"github.com/parquet-go/parquet-go"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/tracefingerprint"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/encoding/vparquet5"
)

const (
	// similaritySampleSize is how many traces the accuracy check compares
	// pairwise against their exact Jaccard similarity.
	similaritySampleSize = 200
	// allocationSampleEvery is how often a fingerprint call is wrapped in
	// runtime.ReadMemStats, which stops the world. The first call is always
	// sampled.
	allocationSampleEvery = 100
	// distributionSampleSize bounds the values a distribution keeps for its
	// percentiles, so the report's memory does not grow with the block.
	distributionSampleSize = 100_000

	phaseRewrite  = "rewrite"
	phaseReadback = "readback"
)

var similarityThresholds = []float64{0.5, 0.8, 0.9}

// fingerprintReport is the JSON report written by --fingerprint-report. Its
// measurements come from reading the rewritten block back, so they describe
// the fingerprints of traces as storage returns them.
type fingerprintReport struct {
	BlockID string `json:"blockID"`
	// FingerprintVersion is the version stored in the block, or empty if the
	// block stores no fingerprints. The measurements below are computed
	// either way, so rewrites with and without fingerprints can be compared.
	FingerprintVersion string `json:"fingerprintVersion"`
	// FailedPhase is "rewrite" or "readback" if that phase failed, and Error
	// says why. The fields of later phases are then zero.
	FailedPhase string `json:"failedPhase,omitempty"`
	Error       string `json:"error,omitempty"`

	// InputRows is the row count of the input data file, and InputRowsRead
	// the rows the rewrite read and passed to the writer. InputRowsFailed is
	// 1 if reading an input row failed; Error also says when the input ended
	// early. InputRowsUnread counts the rows that were never read because the
	// rewrite stopped first.
	InputRows       int64 `json:"inputRows"`
	InputRowsRead   int64 `json:"inputRowsRead"`
	InputRowsFailed int64 `json:"inputRowsFailed"`
	InputRowsUnread int64 `json:"inputRowsUnread"`

	// RewriteSeconds times CreateBlock. RewritePeakRSSBytes is the peak
	// resident memory of the process when it returned, and RewriteAllocs and
	// RewriteAllocBytes the process-wide heap allocations made during it.
	// PeakRSSBytes is the peak at the end, readback included. The peaks are
	// 0 where the platform does not report them.
	RewriteSeconds      float64 `json:"rewriteSeconds"`
	RewritePeakRSSBytes uint64  `json:"rewritePeakRSSBytes"`
	RewriteAllocs       uint64  `json:"rewriteAllocs"`
	RewriteAllocBytes   uint64  `json:"rewriteAllocBytes"`
	PeakRSSBytes        uint64  `json:"peakRSSBytes"`

	// BlockBytes is the size of every file in the block directory, and
	// DataFileBytes the size of its data file.
	BlockBytes    int64 `json:"blockBytes"`
	DataFileBytes int64 `json:"dataFileBytes"`
	// SimilarityColumnBytes is the compressed size of the fingerprint and
	// shingle count columns, and SimilarityColumnShare their share of the
	// data file.
	SimilarityColumnBytes             int64   `json:"similarityColumnBytes"`
	SimilarityColumnUncompressedBytes int64   `json:"similarityColumnUncompressedBytes"`
	SimilarityColumnShare             float64 `json:"similarityColumnShareOfDataFile"`

	// Rows is the number of rows in the output data file and Traces the
	// number read back.
	Rows   int64 `json:"rows"`
	Traces int   `json:"traces"`
	// Computed, Empty and Unavailable count the fingerprints recomputed from
	// the traces read back.
	Computed    int `json:"computed"`
	Empty       int `json:"empty"`
	Unavailable int `json:"unavailable"`
	// StoredFingerprints counts rows that store a fingerprint.
	// StoredMismatches counts rows whose stored columns differ from the
	// recomputed fingerprint, or are not zero where there is none to store.
	// Any mismatch fails the readback.
	StoredFingerprints int `json:"storedFingerprints"`
	StoredMismatches   int `json:"storedMismatches"`

	// Spans counts the spans of every trace, and Shingles the distinct
	// shingles of every trace except unavailable ones.
	Spans    distribution `json:"spans"`
	Shingles distribution `json:"shingles"`

	Computation computation `json:"computation"`
	Accuracy    accuracy    `json:"accuracy"`
}

// distribution summarizes Count values. Min, Max and Mean are exact. The
// percentiles come from a uniform sample of up to distributionSampleSize
// values, so they are exact only for smaller counts.
type distribution struct {
	Count uint64  `json:"count"`
	Min   uint64  `json:"min"`
	P50   uint64  `json:"p50"`
	P90   uint64  `json:"p90"`
	P99   uint64  `json:"p99"`
	Max   uint64  `json:"max"`
	Mean  float64 `json:"mean"`
}

// computation times the steps of fingerprinting, one trace at a time and
// without reading or decoding rows.
type computation struct {
	// Conversion turns a stored row back into a trace. The writer pays it
	// only when service.name is a dedicated resource column.
	Conversion phase `json:"conversion"`
	// Fingerprint is the whole Fingerprint call: shingle extraction, then
	// the MinHash sketch.
	Fingerprint phase `json:"fingerprint"`
	// Extraction is a separate ShingleHashes call on the same trace right
	// after Fingerprint, including copying the hashes out. It is timed on
	// its own, so it is not a share of Fingerprint.
	Extraction phase `json:"shingleExtraction"`
	// AllocsPerTrace and BytesPerTrace average the process-wide heap
	// allocations during AllocationSamples Fingerprint calls. Nothing was
	// measured when AllocationSamples is 0.
	AllocationSamples int     `json:"allocationSamples"`
	AllocsPerTrace    float64 `json:"allocsPerTrace"`
	BytesPerTrace     float64 `json:"bytesPerTrace"`
}

type phase struct {
	Seconds         float64      `json:"seconds"`
	TracesPerSecond float64      `json:"tracesPerSecond"`
	SpansPerSecond  float64      `json:"spansPerSecond"`
	Nanos           distribution `json:"nanosPerTrace"`
}

// accuracy compares Similarity estimates with exact Jaccard similarity for
// every pair in a random sample of traces with fingerprints.
type accuracy struct {
	SampledTraces int     `json:"sampledTraces"`
	Pairs         int     `json:"pairs"`
	MeanAbsError  float64 `json:"meanAbsError"`
	P99AbsError   float64 `json:"p99AbsError"`
	MaxAbsError   float64 `json:"maxAbsError"`
	// ThresholdFlips counts pairs on different sides of a threshold by
	// estimate and by exact similarity.
	ThresholdFlips map[string]int `json:"thresholdFlips"`
	// FalseExactMatches counts pairs estimated as 1 that are not identical.
	FalseExactMatches int  `json:"falseExactMatches"`
	Ties              ties `json:"ties"`
}

// ties looks at each sampled trace as a query and counts the pairs of other
// sampled traces that get the same estimate, so a ranking by estimate cannot
// order them. Comparisons is the number of candidate pairs looked at.
type ties struct {
	Comparisons int `json:"comparisons"`
	Tied        int `json:"tied"`
	// TiedDifferentExact counts tied pairs whose exact similarities differ,
	// and TiedDifferentExactAboveZero those of them estimated above 0.
	TiedDifferentExact          int `json:"tiedDifferentExact"`
	TiedDifferentExactAboveZero int `json:"tiedDifferentExactAboveZero"`
}

type sampledTrace struct {
	result   tracefingerprint.Result
	shingles []uint64
}

// rewriteStats describes the rewrite that wrote the measured block.
type rewriteStats struct {
	duration       time.Duration
	inputRows      int64
	inputRowsRead  int64
	inputRowFailed bool
	peakRSSBytes   uint64
	allocs         uint64
	allocBytes     uint64
}

// measureMemory records the peak memory of the process and its heap
// allocations since before.
func (s *rewriteStats) measureMemory(before *runtime.MemStats) {
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	s.peakRSSBytes = peakRSSBytes()
	s.allocs = after.Mallocs - before.Mallocs
	s.allocBytes = after.TotalAlloc - before.TotalAlloc
}

func newFingerprintReport(blockID string, s rewriteStats) *fingerprintReport {
	var failed int64
	if s.inputRowFailed {
		failed = 1
	}
	return &fingerprintReport{
		BlockID:             blockID,
		InputRows:           s.inputRows,
		InputRowsRead:       s.inputRowsRead,
		InputRowsFailed:     failed,
		InputRowsUnread:     max(0, s.inputRows-s.inputRowsRead-failed),
		RewriteSeconds:      s.duration.Seconds(),
		RewritePeakRSSBytes: s.peakRSSBytes,
		RewriteAllocs:       s.allocs,
		RewriteAllocBytes:   s.allocBytes,
		PeakRSSBytes:        s.peakRSSBytes,
	}
}

func (r *fingerprintReport) fail(phase string, err error) {
	r.FailedPhase = phase
	r.Error = err.Error()
}

// measure reads the vParquet5 block in blockDir back and fingerprints every
// trace. It fails if reading fails or any stored fingerprint differs from the
// trace read back. Either way the report keeps what was measured and is
// marked as failed in the readback phase.
func (r *fingerprintReport) measure(blockDir string, meta *backend.BlockMeta) error {
	err := r.readBlock(blockDir, meta)
	if err == nil && r.StoredMismatches > 0 {
		err = fmt.Errorf("%d rows store fingerprints that differ from the traces read back", r.StoredMismatches)
	}
	r.PeakRSSBytes = peakRSSBytes()
	if err != nil {
		r.fail(phaseReadback, err)
	}
	return err
}

func (r *fingerprintReport) readBlock(blockDir string, meta *backend.BlockMeta) error {
	blockBytes, err := dirBytes(blockDir)
	if err != nil {
		return err
	}
	in, err := os.Open(filepath.Join(blockDir, vparquet5.DataFileName))
	if err != nil {
		return err
	}
	defer in.Close()
	stat, err := in.Stat()
	if err != nil {
		return err
	}
	schema, _, readerOptions := vparquet5.SchemaWithDynamicChanges(vparquet5.TraceSchemaConfig{DedicatedColumns: meta.DedicatedColumns, Options: meta.Options})
	pf, err := parquet.OpenFile(in, stat.Size(), parquet.FileSchema(schema))
	if err != nil {
		return err
	}
	if err := checkColumns(pf, schema, false); err != nil {
		return err
	}

	version := meta.Options[vparquet5.TraceSignatureVersionOption]
	r.FingerprintVersion = version
	r.BlockBytes = blockBytes
	r.DataFileBytes = stat.Size()
	r.Rows = pf.NumRows()
	r.SimilarityColumnBytes, r.SimilarityColumnUncompressedBytes = similarityColumnBytes(pf)
	if stat.Size() > 0 {
		r.SimilarityColumnShare = float64(r.SimilarityColumnBytes) / float64(stat.Size())
	}

	reader := parquet.NewGenericReader[*vparquet5.Trace](pf, readerOptions...)
	defer reader.Close()
	return r.readBack(reader, meta, version == vparquet5.TraceSignatureVersionV1)
}

// readBack fingerprints every row of reader and fills in the measurements.
// signed says whether the block stores fingerprints.
func (r *fingerprintReport) readBack(reader *parquet.GenericReader[*vparquet5.Trace], meta *backend.BlockMeta, signed bool) error {
	var (
		f             tracefingerprint.Fingerprinter
		rng           = rand.New(rand.NewPCG(1, 2)) //nolint:gosec // G404: a fixed seed keeps reports reproducible
		spans         = newDistributionBuilder(3)
		shingleCounts = newDistributionBuilder(4)
		conversion    = newDistributionBuilder(5)
		fingerprint   = newDistributionBuilder(6)
		extraction    = newDistributionBuilder(7)
		shingles      []uint64
		totalSpans    uint64
		allocs        uint64
		allocBytes    uint64
		sample        []sampledTrace
		before        runtime.MemStats
		after         runtime.MemStats
		readErr       error
		// One at a time, so a block of large traces holds one decoded trace.
		rows = make([]*vparquet5.Trace, 1)
		c    = &r.Computation
	)
	for {
		for i := range rows {
			rows[i] = &vparquet5.Trace{}
		}
		n, err := reader.Read(rows)
		for _, row := range rows[:n] {
			start := time.Now()
			tr := vparquet5.ParquetTraceToTempopbTrace(meta, row)
			conversion.addDuration(time.Since(start))
			spanCount := uint64(countSpans(tr))
			totalSpans += spanCount
			spans.add(spanCount)
			r.Traces++

			// Keep the timer's own bookkeeping outside the sampled window.
			sampleAllocs := (r.Traces-1)%allocationSampleEvery == 0
			if sampleAllocs {
				runtime.ReadMemStats(&before)
			}
			start = time.Now()
			res := f.Fingerprint(tr)
			elapsed := time.Since(start)
			if sampleAllocs {
				runtime.ReadMemStats(&after)
				allocs += after.Mallocs - before.Mallocs
				allocBytes += after.TotalAlloc - before.TotalAlloc
				c.AllocationSamples++
			}
			fingerprint.addDuration(elapsed)

			start = time.Now()
			shingles, _ = f.ShingleHashes(shingles[:0], tr)
			extraction.addDuration(time.Since(start))

			if row.Similarity.ShingleCount > 0 {
				r.StoredFingerprints++
			}
			if !storedMatches(row.Similarity, res, signed) {
				r.StoredMismatches++
			}

			switch res.Status {
			case tracefingerprint.StatusComputed:
				r.Computed++
				shingleCounts.add(uint64(res.ShingleCount))
				sample = sampleTrace(rng, sample, r.Computed, shingles, res)
			case tracefingerprint.StatusEmpty:
				r.Empty++
				shingleCounts.add(0)
			default:
				r.Unavailable++
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				readErr = err
			}
			break
		}
	}
	if readErr == nil && int64(r.Traces) != r.Rows {
		readErr = fmt.Errorf("read %d of %d rows", r.Traces, r.Rows)
	}

	r.Spans = spans.distribution()
	r.Shingles = shingleCounts.distribution()
	c.Conversion = conversion.phase(r.Traces, totalSpans)
	c.Fingerprint = fingerprint.phase(r.Traces, totalSpans)
	c.Extraction = extraction.phase(r.Traces, totalSpans)
	if c.AllocationSamples > 0 {
		c.AllocsPerTrace = float64(allocs) / float64(c.AllocationSamples)
		c.BytesPerTrace = float64(allocBytes) / float64(c.AllocationSamples)
	}
	r.Accuracy = measureAccuracy(sample)
	return readErr
}

// storedMatches reports whether the stored columns s hold res, in a block
// that stores fingerprints (signed), or zeros otherwise. It compares the
// fields directly so that it does not share the writer's encoding.
func storedMatches(s vparquet5.TraceSimilarity, res tracefingerprint.Result, signed bool) bool {
	if signed && res.Status == tracefingerprint.StatusComputed {
		return s.Signature == res.Fingerprint && s.ShingleCount == res.ShingleCount
	}
	return s == (vparquet5.TraceSimilarity{})
}

// distributionBuilder collects values for a distribution in bounded memory.
type distributionBuilder struct {
	rng    *rand.Rand
	count  uint64
	sum    float64
	min    uint64
	max    uint64
	sample []uint64
}

func newDistributionBuilder(seed uint64) *distributionBuilder {
	return &distributionBuilder{rng: rand.New(rand.NewPCG(seed, seed))} //nolint:gosec // G404: a fixed seed keeps reports reproducible
}

func (d *distributionBuilder) add(v uint64) {
	if d.count == 0 || v < d.min {
		d.min = v
	}
	d.max = max(d.max, v)
	d.count++
	d.sum += float64(v)
	if len(d.sample) < distributionSampleSize {
		d.sample = append(d.sample, v)
	} else if i := d.rng.Uint64N(d.count); i < distributionSampleSize {
		d.sample[i] = v
	}
}

func (d *distributionBuilder) addDuration(elapsed time.Duration) {
	d.add(uint64(elapsed.Nanoseconds()))
}

func (d *distributionBuilder) distribution() distribution {
	if d.count == 0 {
		return distribution{}
	}
	s := d.sample
	slices.Sort(s)
	return distribution{
		Count: d.count,
		Min:   d.min,
		P50:   s[percentileIndex(len(s), 0.5)],
		P90:   s[percentileIndex(len(s), 0.9)],
		P99:   s[percentileIndex(len(s), 0.99)],
		Max:   d.max,
		Mean:  d.sum / float64(d.count),
	}
}

// phase returns the timing of a step whose per-trace durations in
// nanoseconds were added to d.
func (d *distributionBuilder) phase(traces int, spans uint64) phase {
	ph := phase{Seconds: d.sum / 1e9, Nanos: d.distribution()}
	if d.sum > 0 {
		ph.TracesPerSecond = float64(traces) / ph.Seconds
		ph.SpansPerSecond = float64(spans) / ph.Seconds
	}
	return ph
}

// sampleTrace keeps a uniform random sample of similaritySampleSize traces
// with fingerprints (reservoir sampling). seen counts the traces offered so
// far, including this one. shingles is copied.
func sampleTrace(rng *rand.Rand, sample []sampledTrace, seen int, shingles []uint64, res tracefingerprint.Result) []sampledTrace {
	i := len(sample)
	if i >= similaritySampleSize {
		i = rng.IntN(seen)
		if i >= similaritySampleSize {
			return sample
		}
	} else {
		sample = append(sample, sampledTrace{})
	}
	sample[i] = sampledTrace{result: res, shingles: append(sample[i].shingles[:0], shingles...)}
	return sample
}

func measureAccuracy(sample []sampledTrace) accuracy {
	n := len(sample)
	acc := accuracy{SampledTraces: n, ThresholdFlips: make(map[string]int, len(similarityThresholds))}
	flips := make([]int, len(similarityThresholds))
	// estimates and exacts hold the score of every pair, indexed i*n+j.
	estimates := make([]float64, n*n)
	exacts := make([]float64, n*n)
	var errs []float64
	for i := range sample {
		for j := i + 1; j < n; j++ {
			estimate, err := tracefingerprint.Similarity(&sample[i].result, &sample[j].result)
			if err != nil {
				estimate = math.NaN()
			}
			exact := jaccard(sample[i].shingles, sample[j].shingles)
			estimates[i*n+j], estimates[j*n+i] = estimate, estimate
			exacts[i*n+j], exacts[j*n+i] = exact, exact
			if err != nil {
				continue
			}
			errs = append(errs, math.Abs(estimate-exact))
			for k, t := range similarityThresholds {
				if (estimate >= t) != (exact >= t) {
					flips[k]++
				}
			}
			if estimate == 1 && exact < 1 {
				acc.FalseExactMatches++
			}
		}
	}
	for k, t := range similarityThresholds {
		acc.ThresholdFlips[strconv.FormatFloat(t, 'f', -1, 64)] = flips[k]
	}
	acc.Ties = countTies(n, estimates, exacts)

	acc.Pairs = len(errs)
	if len(errs) == 0 {
		return acc
	}
	slices.Sort(errs)
	sum := 0.0
	for _, e := range errs {
		sum += e
	}
	acc.MeanAbsError = sum / float64(len(errs))
	acc.P99AbsError = errs[percentileIndex(len(errs), 0.99)]
	acc.MaxAbsError = errs[len(errs)-1]
	return acc
}

// countTies counts ties over the n×n pair scores built by measureAccuracy.
// Pairs without an estimate are NaN and skipped.
func countTies(n int, estimates, exacts []float64) ties {
	var t ties
	for q := range n {
		estimate, exact := estimates[q*n:(q+1)*n], exacts[q*n:(q+1)*n]
		for a := range n {
			if a == q || math.IsNaN(estimate[a]) {
				continue
			}
			for b := a + 1; b < n; b++ {
				if b == q || math.IsNaN(estimate[b]) {
					continue
				}
				t.Comparisons++
				if estimate[a] != estimate[b] {
					continue
				}
				t.Tied++
				if exact[a] != exact[b] {
					t.TiedDifferentExact++
					if estimate[a] > 0 {
						t.TiedDifferentExactAboveZero++
					}
				}
			}
		}
	}
	return t
}

// jaccard returns the Jaccard similarity of two sorted sets without duplicates.
func jaccard(a, b []uint64) float64 {
	i, j, shared := 0, 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			shared++
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	union := len(a) + len(b) - shared
	if union == 0 {
		return 1
	}
	return float64(shared) / float64(union)
}

// percentileIndex returns the index of the q quantile in a sorted slice of
// length n > 0, using the nearest-rank method.
func percentileIndex(n int, q float64) int {
	return min(n-1, max(0, int(math.Ceil(q*float64(n)))-1))
}

func similarityColumnBytes(pf *parquet.File) (compressed, uncompressed int64) {
	for _, rg := range pf.Metadata().RowGroups {
		for _, col := range rg.Columns {
			if isSimilarityColumn(col.MetaData.PathInSchema) {
				compressed += col.MetaData.TotalCompressedSize
				uncompressed += col.MetaData.TotalUncompressedSize
			}
		}
	}
	return compressed, uncompressed
}

// dirBytes returns the total size of the regular files in dir.
func dirBytes(dir string) (int64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			return 0, err
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
	}
	return total, nil
}

func countSpans(tr *tempopb.Trace) int {
	n := 0
	for _, rs := range tr.GetResourceSpans() {
		for _, ss := range rs.GetScopeSpans() {
			n += len(ss.GetSpans())
		}
	}
	return n
}

// write writes the report to f, the file reserved for it, closes f and
// prints a summary.
func (r *fingerprintReport) write(f *os.File) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	c := r.Computation
	fmt.Printf("Fingerprint report written to %s\n", f.Name())
	if r.FailedPhase != "" {
		fmt.Printf("  %s failed: %s\n", r.FailedPhase, r.Error)
	}
	fmt.Printf("  input rows=%d read=%d failed=%d unread=%d; rewrite %.1fs, peak RSS %d bytes\n",
		r.InputRows, r.InputRowsRead, r.InputRowsFailed, r.InputRowsUnread, r.RewriteSeconds, r.RewritePeakRSSBytes)
	if r.FailedPhase == phaseRewrite {
		return nil
	}
	fmt.Printf("  traces=%d of %d rows, computed=%d empty=%d unavailable=%d stored=%d storedMismatches=%d\n",
		r.Traces, r.Rows, r.Computed, r.Empty, r.Unavailable, r.StoredFingerprints, r.StoredMismatches)
	fmt.Printf("  block=%d bytes, fingerprint columns=%d bytes (%.2f%% of data file)\n",
		r.BlockBytes, r.SimilarityColumnBytes, 100*r.SimilarityColumnShare)
	fmt.Printf("  fingerprint=%.3fs (%.0f traces/s), shingle extraction alone=%.3fs, row conversion=%.3fs\n",
		c.Fingerprint.Seconds, c.Fingerprint.TracesPerSecond, c.Extraction.Seconds, c.Conversion.Seconds)
	fmt.Printf("  accuracy over %d pairs: mean abs error %.4f, p99 %.4f, max %.4f; ties with different exact scores %d of %d\n",
		r.Accuracy.Pairs, r.Accuracy.MeanAbsError, r.Accuracy.P99AbsError, r.Accuracy.MaxAbsError, r.Accuracy.Ties.TiedDifferentExact, r.Accuracy.Ties.Comparisons)
	return nil
}
