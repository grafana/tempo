package usage

import (
	"math"
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
	tenantLabel  = "tenant"
	trackerLabel = "tracker"
)

var emptyHash = hash(nil, nil)

type tenantLabelsFunc func(string) []string

type bucket struct {
	// Configuration
	descr  *prometheus.Desc // Configuration can change over time so it is captured with the bucket.
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
	series      map[uint64]*bucket
	constLabels prometheus.Labels

	// Buffers for Observe
	dimensions []string // Originally configured dimensions
	labels     []string // Sanitized dimensions => final labels
	values     []string // Used to capture values and guaranteed to be a matching length.
}

// GetBuffersForDimensions takes advantage of the fact that the configuration for a tracker
// changes slowly.  Reuses buffers from the previous call when the dimensions are the same.
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

func (t *tenantUsage) getSeries(labels, values []string, maxCardinality int) *bucket {
	h := hash(labels, values)

	b := t.series[h]
	if b == nil {
		// Before creating a new series, check for cardinality limit.
		if len(t.series) >= maxCardinality {
			// Overflow
			// It goes into the unlabeled bucket
			// TODO - Do we want to do something else?
			clear(values)
			h = emptyHash
			b = t.series[h]
		}
	}

	if b == nil {
		// First encounter with this series. Initialize it.
		l, v := nonEmpties(labels, values)
		b = &bucket{
			// Metric description - constant for this pass now that the dimensions are known
			descr:  prometheus.NewDesc("tempo_usage_tracker_bytes_received_total", "bytes total received with these attributes", l, t.constLabels),
			labels: v,
		}
		t.series[h] = b
	}
	return b
}

type Tracker struct {
	mtx     sync.Mutex
	name    string
	tenants map[string]*tenantUsage
	fn      tenantLabelsFunc
	reg     *prometheus.Registry
	cfg     Config
}

func NewTracker(cfg Config, name string, fn tenantLabelsFunc) (*Tracker, error) {
	u := &Tracker{
		cfg:     cfg,
		name:    name,
		tenants: make(map[string]*tenantUsage),
		fn:      fn,
		reg:     prometheus.NewRegistry(),
	}

	err := u.reg.Register(u)
	if err != nil {
		return nil, err
	}

	go u.PurgeRoutine()

	return u, nil
}

// getTenant must be called under lock.
func (u *Tracker) getTenant(tenant string) *tenantUsage {
	data := u.tenants[tenant]
	if data == nil {
		data = &tenantUsage{
			series: make(map[uint64]*bucket),
			constLabels: prometheus.Labels{
				tenantLabel:  tenant,
				trackerLabel: u.name,
			},
		}
		u.tenants[tenant] = data
	}
	return data
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

	var (
		now            = time.Now().Unix()
		data           = u.getTenant(tenant)
		labels, values = data.GetBuffersForDimensions(dimensions)
	)

	for _, batch := range batches {
		unaccountedForBatchData, totalSpanCount := nonSpanDataLength(batch)

		if totalSpanCount == 0 {
			// Mainly to prevent a panic below, but is this even possible?
			continue
		}

		// This is 1/Nth of the unaccounted for batch data that gets added to each span.
		// Adding this incrementally as we go through the spans is the fastest method, but
		// loses some precision. The other (original) implementation is to record span counts
		// per series into a map and reconcile at the end. That method has more accurate data because
		// it performs the floating point math once on the total, instead of accumulating 1/N + 1/N ... errors.
		batchPortion := int(math.RoundToEven(float64(unaccountedForBatchData) / float64(totalSpanCount)))

		// To account for the accumulated error we dump the remaining delta onto the first span, which can be negative.
		// The result ensures the total recorded bytes matches the input.
		firstSpanPortion := unaccountedForBatchData - batchPortion*totalSpanCount

		// Reset value buffer to be empties
		clear(values)

		if batch.Resource != nil {
			for _, a := range batch.Resource.Attributes {
				v := a.Value.GetStringValue()
				if v == "" {
					continue
				}
				for i, d := range dimensions {
					if d == a.Key {
						values[i] = v
					}
				}
			}
		}

		for i, ss := range batch.ScopeSpans {
			for j, s := range ss.Spans {
				sz := s.Size()
				sz += protoLengthMath(sz)
				sz += batchPortion // Incrementally add 1/Nth worth of the unaccounted for batch data
				if i == 0 && j == 0 {
					sz += firstSpanPortion
				}

				for _, a := range s.Attributes {
					v := a.Value.GetStringValue()
					if v == "" {
						continue
					}
					for i, d := range dimensions {
						if d == a.Key {
							values[i] = v
						}
					}
				}

				// Update after every span because each span
				// can be a different series.
				// TODO - See if we can determine if the buffers
				// haven't changed and avoid hashing again.
				b := data.getSeries(labels, values, int(u.cfg.MaxCardinality))
				b.Inc(uint64(sz), now)
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
// that aren't attributable to specific spans.  It's complicated but much faster
// to do this because it ensures we only measure each part of the proto once.
// The first (and simplier) approach was to call batch.Size() and then subtract
// each encountered span.  But this measures spans twice, which is already the slowest
// part by far. Hopefully isn't too brittle.  It must be updated for new fields above the
// span level.  Also returns the count of spans while we're here so we don't have to loop again.
func nonSpanDataLength(batch *v1.ResourceSpans) (int, int) {
	total := 0
	spans := 0

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

		spans += len(ss.Spans)
	}

	return total, spans
}

// Bookkeeping data to encode a length in proto.
// Copied from sovTrace in .pb.go
func protoLengthMath(x int) (n int) {
	return 1 + (bits.Len64(uint64(x)|1)+6)/7
}
