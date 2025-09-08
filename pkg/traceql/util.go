package traceql

import (
	"math"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("pkg/traceql")

func MakeCollectTagValueFunc(collect func(tempopb.TagValue) bool) func(v Static) bool {
	return func(v Static) bool {
		tv := tempopb.TagValue{}

		switch v.Type {
		case TypeString:
			tv.Type = "string"
			tv.Value = v.EncodeToString(false) // avoid formatting

		case TypeBoolean:
			tv.Type = "bool"
			tv.Value = v.String()

		case TypeInt:
			tv.Type = "int"
			tv.Value = v.String()

		case TypeFloat:
			tv.Type = "float"
			tv.Value = v.String()

		case TypeDuration:
			tv.Type = duration
			tv.Value = v.String()

		case TypeStatus:
			tv.Type = "keyword"
			tv.Value = v.String()
		}

		return collect(tv)
	}
}

// bucketSet is a simple set of buckets that can be used to track the number of exemplars
type bucketSet interface {
	testTotal() bool
	addAndTest(ts uint64) bool
}

// newExemplarBucketSet creates a new bucket set for the aligned time range
// start and end are in nanoseconds.
// If the range is instant, empty bucket set is returned.
func newExemplarBucketSet(exemplars uint32, start, end, step uint64) bucketSet {
	if isInstant(start, end, step) {
		return &alwaysFullBucketSet{}
	}

	start = alignStart(start, end, step)
	end = alignEnd(start, end, step)

	return newBucketSet(exemplars, start, end)
}

type alwaysFullBucketSet struct{}

func (b *alwaysFullBucketSet) testTotal() bool {
	return true
}

func (b *alwaysFullBucketSet) addAndTest(uint64) bool {
	return true
}

type limitedBucketSet struct {
	sz, maxTotal, maxBucket int
	buckets                 []int
	start, end, bucketWidth uint64
}

// newBucketSet creates a new bucket set for the given time range
// start and end are in nanoseconds
func newBucketSet(exemplars uint32, start, end uint64) *limitedBucketSet {
	if exemplars > maxExemplars || exemplars == 0 {
		exemplars = maxExemplars
	}
	buckets := exemplars / maxExemplarsPerBucket
	if buckets == 0 { // edge case for few exemplars
		buckets = 1
	}

	// convert nanoseconds to milliseconds
	start /= uint64(time.Millisecond.Nanoseconds()) //nolint: gosec // G115
	end /= uint64(time.Millisecond.Nanoseconds())   //nolint: gosec // G115

	interval := end - start
	bucketWidth := interval / uint64(buckets)

	return &limitedBucketSet{
		sz:          int(buckets),
		maxTotal:    int(exemplars),
		maxBucket:   maxExemplarsPerBucket,
		buckets:     make([]int, buckets+1), // +1 for total count
		start:       start,
		end:         end,
		bucketWidth: bucketWidth,
	}
}

func (b *limitedBucketSet) len() int {
	return b.buckets[b.sz]
}

func (b *limitedBucketSet) testTotal() bool {
	return b.len() >= b.maxTotal
}

func (b *limitedBucketSet) inRange(ts uint64) bool {
	return b.start <= ts && ts <= b.end
}

func (b *limitedBucketSet) bucket(ts uint64) int {
	if b.start == b.end {
		return 0
	}

	bucket := int((ts - b.start) / b.bucketWidth) //nolint: gosec // G115

	// Clamp to last bucket to handle edge rounding
	if bucket >= b.sz {
		bucket = b.sz - 1
	}

	return bucket
}

// addAndTest adds a timestamp to the bucket set and returns true if the total exceeds the max total
// the timestamp is in milliseconds
func (b *limitedBucketSet) addAndTest(ts uint64) bool {
	if !b.inRange(ts) || b.testTotal() || b.sz == 0 {
		return true
	}

	i := b.bucket(ts)
	if b.buckets[i] >= b.maxBucket {
		return true
	}

	b.buckets[i]++
	b.buckets[b.sz]++
	return false
}

const (
	leftBranch  = 0
	rightBranch = 1
)

type branchOptimizer struct {
	start            time.Time
	last             []time.Duration
	totals           []time.Duration
	Recording        bool
	samplesRemaining int
}

func newBranchPredictor(numBranches int, numSamples int) branchOptimizer {
	return branchOptimizer{
		totals:           make([]time.Duration, numBranches),
		last:             make([]time.Duration, numBranches),
		samplesRemaining: numSamples,
		Recording:        true,
	}
}

// Start recording. Should be called immediately prior to a branch execution.
func (b *branchOptimizer) Start() {
	b.start = time.Now()
}

// Finish the recording and temporarily save the cost for the given branch number.
func (b *branchOptimizer) Finish(branch int) {
	b.last[branch] = time.Since(b.start)
}

// Penalize the given branch using it's previously recorded cost.  This is called after
// executing all branches and then knowing in retrospect which ones were not needed.
func (b *branchOptimizer) Penalize(branch int) {
	b.totals[branch] += b.last[branch]
}

// Sampled indicates that a full execution was done and see if we have enough samples.
func (b *branchOptimizer) Sampled() (done bool) {
	b.samplesRemaining--
	b.Recording = b.samplesRemaining > 0
	return !b.Recording
}

// OptimalBranch returns the branch with the least penalized cost over time, i.e. the optimal one to start with.
func (b *branchOptimizer) OptimalBranch() int {
	mini := 0
	min := b.totals[0]
	for i := 1; i < len(b.totals); i++ {
		if b.totals[i] < min {
			mini = i
			min = b.totals[i]
		}
	}
	return mini
}

func kahanSumInc(inc, sum, c float64) (newSum, newC float64) {
	t := sum + inc
	switch {
	case math.IsInf(t, 0):
		c = 0

	// Using Neumaier improvement, swap if next term larger than sum.
	case math.Abs(sum) >= math.Abs(inc):
		c += (sum - t) + inc
	default:
		c += (inc - t) + sum
	}
	return t, c
}
