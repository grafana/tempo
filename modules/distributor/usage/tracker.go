package usage

import (
	"math/bits"
	"net/http"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/prometheus/util/strutil"

	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

const (
	tenantLabel = "tenant"
)

type tenantLabelsFunc func(string) []string

type bucket struct {
	// Configuration
	descr  *prometheus.Desc
	labels []string

	// Runtime data
	bytes uint64
	unix  int64
}

func (b *bucket) Inc(bytes uint64, unix int64) {
	b.bytes += bytes
	b.unix = unix
}

type tenantUsage struct {
	series map[uint64]*bucket

	// Buffers for Observe
	dimensions []string // Originally configured dimensions
	labels     []string // Sanitized dimensions => final labels
	values     []string
}

func (t *tenantUsage) GetBuffersForDimensions(dimensions []string) ([]string, []string) {
	reset := false

	if len(dimensions) == len(t.dimensions) {
		// Any change to dimensions?
		for i := range dimensions {
			if dimensions[i] != t.dimensions[i] {
				reset = true
				break
			}
		}
	} else {
		reset = true
	}

	if reset {
		t.dimensions = dimensions
		t.labels = make([]string, len(dimensions))
		t.values = make([]string, len(dimensions))
		for i, d := range dimensions {
			t.labels[i] = strutil.SanitizeFullLabelName(d)
		}
	}

	return t.labels, t.values
}

type Tracker struct {
	mtx     sync.Mutex
	name    string
	tenants map[string]*tenantUsage
	fn      tenantLabelsFunc
	reg     *prometheus.Registry
	cfg     Config

	// Buffers
	spanCounts map[uint64]uint64
}

func NewTracker(cfg Config, name string, fn tenantLabelsFunc) (*Tracker, error) {
	u := &Tracker{
		cfg:        cfg,
		name:       name,
		tenants:    make(map[string]*tenantUsage),
		fn:         fn,
		reg:        prometheus.NewRegistry(),
		spanCounts: make(map[uint64]uint64),
	}

	err := u.reg.Register(u)
	if err != nil {
		return nil, err
	}

	go u.PurgeRoutine()

	return u, nil
}

func (u *Tracker) Observe(tenant string, batches []*v1.ResourceSpans) {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	dimensions := u.fn(tenant)
	if len(dimensions) == 0 {
		// Not configured
		// TODO - Should we put it all in the unattributed bucket instead?
		return
	}

	now := time.Now().Unix()
	spanCounts := u.spanCounts

	data := u.tenants[tenant]
	if data == nil {
		data = &tenantUsage{
			series: make(map[uint64]*bucket),
		}
		u.tenants[tenant] = data
	}

	labels, buffer := data.GetBuffersForDimensions(dimensions)

	for _, batch := range batches {
		unaccountedForBatchData := nonSpanDataLength(batch)
		totalSpanCount := 0

		// Reset spancounts for batch proportioning at the end
		clear(spanCounts)

		// Reset value buffer to be empties
		clear(buffer)

		if batch.Resource != nil {
			for _, a := range batch.Resource.Attributes {
				v := a.Value.GetStringValue()
				if v == "" {
					continue
				}
				for i, d := range dimensions {
					if d == a.Key {
						buffer[i] = v
					}
				}
			}
		}

		for _, ss := range batch.ScopeSpans {
			for _, s := range ss.Spans {
				sz := s.Size()
				sz += protoLengthMath(sz)

				for _, a := range s.Attributes {
					v := a.Value.GetStringValue()
					if v == "" {
						continue
					}
					for i, d := range dimensions {
						if d == a.Key {
							buffer[i] = v
						}
					}
				}

				//  Update trackers after every span
				h := hash(labels, buffer)
				b := data.series[h]
				if b == nil {
					// Before creating a new series, check for cardinality limit.
					if len(data.series) >= int(u.cfg.MaxCardinality) {
						// Overflow
						// It goes into the unlabeled bucket
						// TODO - Do we want to do something else?
						clear(buffer)
						h = hash(labels, buffer)
						b = data.series[h]
					}
				}

				if b == nil {
					// First encounter with this series. Initialize it.
					l, v := nonEmpties(labels, buffer)
					b = &bucket{
						// m:
						// Metric description - constant for this pass now that the dimensions are known
						descr: prometheus.NewDesc("tempo_usage_tracker_bytes_received_total", "bytes total received with these attributes", l, prometheus.Labels{
							tenantLabel: tenant,
						}),
						labels: v,
					}
					data.series[h] = b
				}
				b.Inc(uint64(sz), now)
				spanCounts[h]++
				totalSpanCount++
			}
		}

		// This is all non-span data that must be split proportionally
		if unaccountedForBatchData > 0 {
			// Each series that came out of this batch
			// gets a proportion of the batch-level data
			// based on the proportion of spans.
			for h, count := range spanCounts {
				bytes := uint64(float64(count) / float64(totalSpanCount) * float64(unaccountedForBatchData))
				data.series[h].Inc(bytes, now)
			}
		}
	}
}

func (u *Tracker) PurgeRoutine() {
	purge := time.NewTicker(u.cfg.PurgePeriod)
	for range purge.C {
		u.purge()
	}
}

func (u *Tracker) purge() {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	stale := time.Now().Add(-u.cfg.StaleDuration).Unix()

	for t, data := range u.tenants {
		for h, s := range data.series {
			if s.unix <= stale {
				delete(data.series, h)
			}
		}

		if len(data.series) == 0 {
			// Remove empty tenant
			delete(u.tenants, t)
		}
	}
}

func (u *Tracker) Handler() http.Handler {
	return promhttp.HandlerFor(u.reg, promhttp.HandlerOpts{})
}

func (u *Tracker) Describe(chan<- *prometheus.Desc) {
}

func (u *Tracker) Collect(ch chan<- prometheus.Metric) {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	for _, t := range u.tenants {
		for _, b := range t.series {
			ch <- prometheus.MustNewConstMetric(b.descr, prometheus.CounterValue, float64(b.bytes), b.labels...)
		}
	}
}

func hash(labels, values []string) uint64 {
	h := xxhash.New()

	for i := range values {
		// Only include labels where we got a value
		if values[i] != "" {
			_, _ = h.WriteString(labels[i])
			_, _ = h.Write([]byte{255})
			_, _ = h.WriteString(values[i])
			_, _ = h.Write([]byte{255})
		}
	}

	return h.Sum64()
}

func nonEmpties(labels, values []string) ([]string, []string) {
	var outLabels []string
	var outValues []string

	for i := range values {
		// Only include labels where we got a value
		if values[i] != "" {
			outLabels = append(outLabels, labels[i])
			outValues = append(outValues, values[i])
		}
	}

	return outLabels, outValues
}

// nonSpanDataLength returns the number of proto bytes in the batch
// that aren't attributable to specific spans.  Please don't hate me,
// it's complicated but much faster to do this manually.  The easiest way is to
// call batch.Size() and then subtract each encountered span.  But this measures
// spans twice, which is significant.  Therefore this eliminates a ton of work.
// Hopefully isn't too brittle.  It must be updated for new fields above the
// span level.
func nonSpanDataLength(batch *v1.ResourceSpans) int {
	total := 0

	if batch.Resource != nil {
		sz := batch.Resource.Size()
		total += sz + protoLengthMath(sz)
	}

	l := len(batch.SchemaUrl)
	if l > 0 {
		total += l + protoLengthMath(l)
	}

	for _, ss := range batch.ScopeSpans {
		// This is the data to store the prescence of this ss
		total += protoLengthMath(1)

		l = len(ss.SchemaUrl)
		if l > 0 {
			total += l + protoLengthMath(l)
		}

		if ss.Scope != nil {
			sz := ss.Scope.Size()
			total += sz + protoLengthMath(sz)
		}
	}

	return total
}

// Bookkeeping data to encode a length in proto.
// Copied from sovTrace in .pb.go
func protoLengthMath(x int) (n int) {
	return 1 + (bits.Len64(uint64(x)|1)+6)/7
}
