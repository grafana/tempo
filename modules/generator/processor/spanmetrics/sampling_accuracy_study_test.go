package spanmetrics

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"sort"
	"testing"

	"github.com/prometheus/prometheus/promql"
)

// TestSamplingAccuracyStudy derives the sample counts documented alongside
// max_spans_per_series_per_second: how many of a series' spans have to be
// sampled inside a query's lookback window before that query's answer is
// within 1%, 3% or 5% of the unsampled answer 99% of the time.
//
// It is a numerical study rather than an assertion about the implementation,
// so it is skipped unless TEMPO_SAMPLING_STUDY is set:
//
//	TEMPO_SAMPLING_STUDY=1 go test ./modules/generator/processor/spanmetrics \
//		-run TestSamplingAccuracyStudy -v -timeout 30m
//
// Re-run it when the default histogram buckets change, or to re-derive the
// numbers for a different latency shape, and update the table in
// docs/sources/tempo/metrics-from-traces/span-metrics/.
func TestSamplingAccuracyStudy(t *testing.T) {
	if os.Getenv("TEMPO_SAMPLING_STUDY") == "" {
		t.Skip("set TEMPO_SAMPLING_STUDY=1 to run the sampling accuracy study")
	}

	const trials = 20_000
	studyCounts(t)
	studyQuantiles(t, trials)
	studySchemes(t)
}

// zP99 is the two-sided 99% normal quantile: 99% of draws land within this many
// standard deviations of the mean.
const zP99 = 2.5758293

// studyCounts covers every quantity whose estimate is a scaled-up count of
// sampled spans: _count, each histogram bucket, and (with the coefficient of
// variation folded in) _sum and size_total.
func studyCounts(t *testing.T) {
	t.Log("== Scaled-up count accuracy ==")
	t.Log("m = sampled spans of that series (or in that bucket) inside the query window")
	t.Logf("%10s  %14s  %14s", "m", "p99 |relerr|", "z/sqrt(m)")
	for _, m := range []float64{10, 100, 1_000, 10_000, 66_347, 100_000} {
		t.Logf("%10.0f  %13.2f%%  %13.2f%%", m, 100*countRelErrAtP99(m), 100*zP99/math.Sqrt(m))
	}

	t.Log("")
	t.Logf("%-46s %10s %10s %10s", "quantity", "1%", "3%", "5%")
	for _, row := range []struct {
		name string
		need func(eps float64) float64
	}{
		// Stratified sampling keeps exactly one span per block, so a count can
		// only be off by the one block still in flight: |relerr| <= 1/m. This is
		// the whole reason the processor stratifies rather than keeping each span
		// with independent probability -- see studySchemes.
		{"count/rate, stratified (what Tempo does)", func(e float64) float64 { return math.Ceil(1 / e) }},
		{"count/rate, independent per-span sampling", func(e float64) float64 { return math.Ceil(zP99 * zP99 / (e * e)) }},
		{"sum of a quantity with CV=1.0 (size_total)", func(e float64) float64 { return sumSamplesNeeded(1.0, e) }},
		{"sum with CV=2.06 (sigma=1.29 lognormal latency)", func(e float64) float64 { return sumSamplesNeeded(2.06, e) }},
		{"sum with CV=0.50 (sigma=0.47 lognormal latency)", func(e float64) float64 { return sumSamplesNeeded(0.5, e) }},
	} {
		t.Logf("%-46s %10.0f %10.0f %10.0f", row.name, row.need(0.01), row.need(0.03), row.need(0.05))
	}
	t.Log("")
}

// sumSamplesNeeded inverts the standard error of a sample mean, sd = CV/sqrt(m).
func sumSamplesNeeded(cv, eps float64) float64 {
	return math.Ceil(zP99 * zP99 * cv * cv / (eps * eps))
}

// countRelErrAtP99 returns the relative error on a scaled-up count that holds
// 99% of the time, when m spans are expected to land in it. The sampled count
// is modelled as Poisson(m), the small-share limit of sampling m of a much
// larger stream.
func countRelErrAtP99(m float64) float64 {
	lo, hi := poissonInterval(m, 0.99)
	return math.Max(math.Abs(lo/m-1), math.Abs(hi/m-1))
}

// poissonInterval returns the central interval of Poisson(lambda) holding at
// least the requested coverage.
func poissonInterval(lambda, coverage float64) (lo, hi float64) {
	tail := (1 - coverage) / 2
	logLambda := math.Log(lambda)

	// log pmf computed recursively: logpmf(k) = logpmf(k-1) + log(lambda) - log(k)
	logPmf := -lambda
	cdf := math.Exp(logPmf)
	lo, hi = -1, -1
	if cdf > tail {
		lo = 0
	}
	if cdf >= 1-tail {
		return 0, 0
	}
	for k := 1.0; ; k++ {
		logPmf += logLambda - math.Log(k)
		cdf += math.Exp(logPmf)
		if lo < 0 && cdf > tail {
			lo = k
		}
		if cdf >= 1-tail || k > lambda+50*math.Sqrt(lambda)+100 {
			return lo, k
		}
	}
}

// studyQuantiles measures the error sampling adds to histogram_quantile. The
// reference is histogram_quantile over the unsampled population with the same
// buckets, so bucket quantization -- which sampling does not change -- is
// excluded and only the sampling error is left.
//
// A classic histogram's quantile is a function of its bucket counts alone, so
// drawing the bucket counts directly from the latency distribution is exact:
// there is no need to draw individual latencies.
func studyQuantiles(t *testing.T, trials int) {
	bounds := defaultHistogramBuckets()
	phis := []float64{0.50, 0.90, 0.99}
	targets := []float64{0.01, 0.03, 0.05}

	// Logarithmic grid, 8 points per decade.
	var grid []float64
	for e := 1.5; e <= 6.5001; e += 0.125 {
		grid = append(grid, math.Round(math.Pow(10, e)))
	}

	t.Log("== histogram_quantile error introduced by sampling ==")
	t.Log("Smallest m whose error, and every larger m's error, stays under target.")
	t.Logf("%-30s %6s %12s %12s %12s", "latency shape", "phi", "m for 1%", "m for 3%", "m for 5%")

	for _, dist := range []lognormalLatency{
		newLognormalLatency("web   (p50 50ms, p99 1s)", 0.05, 1.0),
		newLognormalLatency("fast  (p50 5ms, p99 100ms)", 0.005, 0.1),
		newLognormalLatency("slow  (p50 500ms, p99 8s)", 0.5, 8.0),
		newLognormalLatency("tight (p50 50ms, p99 150ms)", 0.05, 0.15),
	} {
		for _, phi := range phis {
			errs := make([]float64, len(grid))
			for i, m := range grid {
				errs[i] = quantileErrAtP99(dist, bounds, phi, m, trials, rand.New(rand.NewPCG(uint64(m), uint64(phi*1000))))
			}
			row := fmt.Sprintf("%-30s %6s", dist.name, fmt.Sprintf("p%g", phi*100))
			for _, target := range targets {
				need := math.Inf(1)
				for i := len(grid) - 1; i >= 0 && errs[i] <= target; i-- {
					need = grid[i]
				}
				if math.IsInf(need, 1) {
					row += fmt.Sprintf(" %12s", ">3e6")
				} else {
					row += fmt.Sprintf(" %12.0f", need)
				}
			}
			t.Log(row)
		}
	}
	t.Log("")
}

func quantileErrAtP99(dist lognormalLatency, bounds []float64, phi, m float64, trials int, r *rand.Rand) float64 {
	probs := dist.bucketProbs(bounds)
	reference := classicQuantile(phi, bounds, probs)
	counts := make([]float64, len(probs))
	errs := make([]float64, 0, trials)
	for i := 0; i < trials; i++ {
		for j, p := range probs {
			counts[j] = poissonDraw(r, m*p)
		}
		errs = append(errs, classicQuantile(phi, bounds, counts)/reference-1)
	}
	return p99Abs(errs)
}

// studySchemes compares three unbiased schemes that all keep the same number of
// spans, so it isolates the variance each contributes at equal CPU cost. It is
// the justification for stratifying: the count error collapses and nothing else
// gets worse.
func studySchemes(t *testing.T) {
	bounds := defaultHistogramBuckets()
	dist := newLognormalLatency("web (p50 50ms, p99 1s)", 0.05, 1.0)

	t.Log("== Scheme comparison at equal CPU cost ==")
	t.Log("One series,", dist.name, ", keep rate 1/20 in every scheme.")
	t.Logf("%8s  %-12s  %9s  %9s  %9s  %9s", "m", "scheme", "count", "sum", "p50", "p99")

	const (
		blockSize = 20
		trials    = 2_000
	)
	for _, m := range []int{100, 1_000, 10_000} {
		for _, row := range compareSchemes(bounds, dist, m*blockSize, blockSize, trials, uint64(m)*104729) {
			t.Logf("%8d  %-12s  %8.2f%%  %8.2f%%  %8.2f%%  %8.2f%%",
				m, row.scheme, 100*row.countErr, 100*row.sumErr, 100*row.p50Err, 100*row.p99Err)
		}
	}
	t.Log("")
	t.Log("Stratified pins the count; bucket shape, and so every quantile, is")
	t.Log("multinomial in all three, which is why no scheme wins on quantiles.")
}

type schemeErrors struct {
	scheme                           string
	countErr, sumErr, p50Err, p99Err float64
}

func compareSchemes(bounds []float64, dist lognormalLatency, spans, blockSize, trials int, seed uint64) []schemeErrors {
	probs := dist.bucketProbs(bounds)
	cumulative := make([]float64, len(probs))
	acc := 0.0
	for i, p := range probs {
		acc += p
		cumulative[i] = acc
	}
	// Representative value per bucket for the _sum estimate.
	mid := make([]float64, len(probs))
	for i := range probs {
		switch {
		case i == len(bounds):
			mid[i] = bounds[len(bounds)-1] * 1.5
		case i == 0:
			mid[i] = bounds[0] / 2
		default:
			mid[i] = math.Sqrt(bounds[i-1] * bounds[i])
		}
	}

	nb := len(probs)
	truth := make([]float64, nb)
	stream := make([]int, spans)
	estimates := map[string][]float64{}
	record := func(scheme string, counts []float64, trueTotal, trueSum, trueP50, trueP99 float64) {
		estimates[scheme+":count"] = append(estimates[scheme+":count"], sumOf(counts)/trueTotal-1)
		estimates[scheme+":sum"] = append(estimates[scheme+":sum"], dotOf(counts, mid)/trueSum-1)
		estimates[scheme+":p50"] = append(estimates[scheme+":p50"], classicQuantile(0.5, bounds, counts)/trueP50-1)
		estimates[scheme+":p99"] = append(estimates[scheme+":p99"], classicQuantile(0.99, bounds, counts)/trueP99-1)
	}

	r := rand.New(rand.NewPCG(seed, seed+1))
	kept := int(spans / blockSize)
	scale := float64(blockSize)
	stratified := make([]float64, nb)
	independent := make([]float64, nb)
	reservoir := make([]float64, nb)

	for trial := 0; trial < trials; trial++ {
		for i := range truth {
			truth[i], stratified[i], independent[i], reservoir[i] = 0, 0, 0, 0
		}
		for i := range stream {
			b := sort.SearchFloat64s(cumulative, r.Float64())
			if b >= nb {
				b = nb - 1
			}
			stream[i] = b
			truth[b]++
		}

		// Stratified: one uniformly chosen span per consecutive block.
		for start := 0; start < spans; start += blockSize {
			end := min(start+blockSize, spans)
			stratified[stream[start+r.IntN(end-start)]] += scale
		}
		// Independent: keep each span with probability 1/blockSize.
		for i := range stream {
			if r.Float64() < 1/scale {
				independent[stream[i]] += scale
			}
		}
		// Global reservoir over the whole tenant stream. This series is a small
		// share of it, so how many of its spans land in the reservoir is itself
		// random and averages kept rather than being exactly kept.
		landed := min(int(poissonDraw(r, float64(kept))), spans)
		for i := 0; i < landed; i++ {
			j := i + r.IntN(spans-i)
			stream[i], stream[j] = stream[j], stream[i]
			reservoir[stream[i]] += scale
		}

		trueTotal, trueSum := sumOf(truth), dotOf(truth, mid)
		trueP50 := classicQuantile(0.5, bounds, truth)
		trueP99 := classicQuantile(0.99, bounds, truth)
		record("stratified", stratified, trueTotal, trueSum, trueP50, trueP99)
		record("independent", independent, trueTotal, trueSum, trueP50, trueP99)
		record("reservoir", reservoir, trueTotal, trueSum, trueP50, trueP99)
	}

	rows := make([]schemeErrors, 0, 3)
	for _, scheme := range []string{"stratified", "independent", "reservoir"} {
		rows = append(rows, schemeErrors{
			scheme:   scheme,
			countErr: p99Abs(estimates[scheme+":count"]),
			sumErr:   p99Abs(estimates[scheme+":sum"]),
			p50Err:   p99Abs(estimates[scheme+":p50"]),
			p99Err:   p99Abs(estimates[scheme+":p99"]),
		})
	}
	return rows
}

// --- helpers ---

// defaultHistogramBuckets returns the buckets a default-configured processor uses.
func defaultHistogramBuckets() []float64 {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	return cfg.HistogramBuckets
}

type lognormalLatency struct {
	name      string
	mu, sigma float64
}

// newLognormalLatency builds a lognormal from a median and a p99, in seconds.
func newLognormalLatency(name string, median, p99 float64) lognormalLatency {
	mu := math.Log(median)
	return lognormalLatency{name: name, mu: mu, sigma: (math.Log(p99) - mu) / 2.3263478740408408}
}

func (l lognormalLatency) cdf(x float64) float64 {
	if x <= 0 {
		return 0
	}
	return 0.5 * math.Erfc(-((math.Log(x)-l.mu)/l.sigma)/math.Sqrt2)
}

// bucketProbs returns the probability of landing in each bucket, plus a final
// +Inf bucket, for the given upper bounds.
func (l lognormalLatency) bucketProbs(bounds []float64) []float64 {
	probs := make([]float64, len(bounds)+1)
	prev := 0.0
	for i, b := range bounds {
		c := l.cdf(b)
		probs[i] = c - prev
		prev = c
	}
	probs[len(bounds)] = 1 - prev
	return probs
}

// classicQuantile runs Prometheus' own histogram_quantile over per-bucket
// counts, so the study measures what a real query would return.
func classicQuantile(phi float64, bounds, counts []float64) float64 {
	buckets := make(promql.Buckets, 0, len(counts))
	cumulative := 0.0
	for i, c := range counts {
		cumulative += c
		upper := math.Inf(1)
		if i < len(bounds) {
			upper = bounds[i]
		}
		buckets = append(buckets, promql.Bucket{UpperBound: upper, Count: cumulative})
	}
	q, _, _, _, _, _ := promql.BucketQuantile(phi, buckets)
	return q
}

// p99Abs is the 99th percentile of the absolute values. A NaN -- a trial whose
// sample had no observations at all, so no quantile exists -- counts as
// unbounded so it cannot silently improve the percentile.
func p99Abs(values []float64) float64 {
	abs := make([]float64, len(values))
	for i, v := range values {
		if math.IsNaN(v) {
			abs[i] = math.Inf(1)
			continue
		}
		abs[i] = math.Abs(v)
	}
	sort.Float64s(abs)
	return abs[max(int(math.Ceil(0.99*float64(len(abs))))-1, 0)]
}

// poissonDraw samples Poisson(lambda): Knuth's multiplication method for small
// lambda, Hoermann's PTRS transformed rejection for large.
func poissonDraw(r *rand.Rand, lambda float64) float64 {
	if lambda <= 0 {
		return 0
	}
	if lambda < 10 {
		limit := math.Exp(-lambda)
		k, p := 0.0, 1.0
		for {
			k++
			p *= r.Float64()
			if p <= limit {
				return k - 1
			}
		}
	}

	b := 0.931 + 2.53*math.Sqrt(lambda)
	a := -0.059 + 0.02483*b
	invAlpha := 1.1239 + 1.1328/(b-3.4)
	vr := 0.9277 - 3.6224/(b-2)
	logLambda := math.Log(lambda)

	for {
		u := r.Float64() - 0.5
		v := r.Float64()
		us := 0.5 - math.Abs(u)
		k := math.Floor((2*a/us+b)*u + lambda + 0.43)
		if us >= 0.07 && v <= vr {
			return k
		}
		if k < 0 || (us < 0.013 && v > us) {
			continue
		}
		lgamma, _ := math.Lgamma(k + 1)
		if math.Log(v*invAlpha/(a/(us*us)+b)) <= -lambda+k*logLambda-lgamma {
			return k
		}
	}
}

func sumOf(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}

func dotOf(values, weights []float64) float64 {
	total := 0.0
	for i, v := range values {
		total += v * weights[i]
	}
	return total
}
