package usage

import (
	"maps"
	"math"
	"math/bits"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/prometheus/util/strutil"

	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

const (
	tenantLabel   = "tenant"
	trackerLabel  = "tracker"
	missingLabel  = "__missing__"
	overflowLabel = "__overflow__"
)

type (
	tenantLabelsFunc func(string) map[string]string
	tenantMaxFunc    func(string) uint64
)

type bucket struct {
	// Configuration
	descr  *prometheus.Desc // Configuration can change over time so it is captured with the bucket.
	labels []string

	// Runtime data
	bytes       uint64
	lastUpdated int64
}

func (b *bucket) Inc(bytes uint64, unix int64) {
	b.bytes += bytes
	b.lastUpdated = unix
}

type mapping struct {
	from string
	to   string
}

type tenantUsage struct {
	series      map[uint64]*bucket
	constLabels prometheus.Labels

	// Buffers for Observe
	dimensions map[string]string // Originally configured dimensions
	mapping    []mapping         // Mapping from attribute => final sanitized label. Typically few values and slice is faster than map
	sortedKeys []string          // So we can always iterate the buffer in order, this can be precomputed up front
	buffer1    map[string]string
	buffer2    map[string]string
	overflow   uint64
}

// GetBuffersForDimensions takes advantage of the fact that the configuration for a tracker
// changes slowly.  Reuses buffers from the previous call when the dimensions are the same.
func (t *tenantUsage) GetBuffersForDimensions(dimensions map[string]string) ([]mapping, map[string]string, map[string]string) {
	if !maps.Equal(dimensions, t.dimensions) {
		// The configuration changed.

		// Save new config and reset all buffers
		t.dimensions = dimensions
		distinctKeys := make(map[string]struct{}, len(dimensions))
		t.mapping = t.mapping[:0]
		t.sortedKeys = t.sortedKeys[:0]

		for k, v := range dimensions {
			// Get the final sanitized output label for this
			// dimension.  Dimensions are key-value pairs with
			// optional value.  If value is empty string, then
			// we use the just the key.  Regardless the output
			// is always the sanitized version.
			// Example:
			//    service.name="" 			=> "service_name"
			//    service.name="foo.bar"	=> "foo_bar"
			var sanitized string
			if v == "" {
				// The dimension is using default mapping
				v = k
			}
			sanitized = strutil.SanitizeFullLabelName(v)
			t.mapping = append(t.mapping, mapping{from: k, to: sanitized})
			distinctKeys[sanitized] = struct{}{}
		}

		// This is the sorted listed of final distinct output labels
		for k := range distinctKeys {
			t.sortedKeys = append(t.sortedKeys, k)
		}
		slices.Sort(t.sortedKeys)

		// Prepopulate the buffer and precompute the overflow bucket
		t.buffer1 = make(map[string]string, len(t.sortedKeys))
		t.buffer2 = make(map[string]string, len(t.sortedKeys))
		for _, k := range t.sortedKeys {
			t.buffer1[k] = overflowLabel
			t.buffer2[k] = missingLabel
		}
		t.overflow = hash(t.sortedKeys, t.buffer1)
	}
	return t.mapping, t.buffer1, t.buffer2
}

// func (t *tenantUsage) getSeries(labels, values []string, maxCardinality uint64) *bucket {
func (t *tenantUsage) getSeries(buffer map[string]string, maxCardinality uint64) *bucket {
	h := hash(t.sortedKeys, buffer)

	b := t.series[h]
	if b == nil {
		// Before creating a new series, check for cardinality limit.
		if uint64(len(t.series)) >= maxCardinality {
			// Overflow
			// This tenant is at the maximum number of series.  In this case all data
			// goes into the final overflow bucket. It has the same dimensions as the
			// current configuration, except every label is overridden to the special overflow value.
			for k := range buffer {
				buffer[k] = overflowLabel
			}
			h = t.overflow
			b = t.series[h]
		}
	}

	if b == nil {
		// First encounter with this series. Initialize it.
		v := flatten(t.sortedKeys, buffer)
		b = &bucket{
			// Metric description - constant for this pass now that the dimensions are known
			descr:  prometheus.NewDesc("tempo_usage_tracker_bytes_received_total", "bytes total received with these attributes", t.sortedKeys, t.constLabels),
			labels: v,
		}
		t.series[h] = b
	}
	return b
}

type Tracker struct {
	mtx      sync.Mutex
	name     string
	tenants  map[string]*tenantUsage
	labelsFn tenantLabelsFunc
	maxFn    tenantMaxFunc
	reg      *prometheus.Registry
	cfg      Config
}

func NewTracker(cfg Config, name string, labelsFn tenantLabelsFunc, maxFn tenantMaxFunc) (*Tracker, error) {
	u := &Tracker{
		cfg:      cfg,
		name:     name,
		tenants:  make(map[string]*tenantUsage),
		labelsFn: labelsFn,
		maxFn:    maxFn,
		reg:      prometheus.NewRegistry(),
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

	dimensions := u.labelsFn(tenant)
	if len(dimensions) == 0 {
		// Not configured
		// TODO - Should we put it all in the unattributed bucket instead?
		return
	}

	max := u.maxFn(tenant)
	if max == 0 {
		max = u.cfg.MaxCardinality
	}

	var (
		now                       = time.Now().Unix()
		data                      = u.getTenant(tenant)
		mapping, buffer1, buffer2 = data.GetBuffersForDimensions(dimensions)
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

		// Reset value buffer for every batch.
		for k := range buffer1 {
			buffer1[k] = missingLabel
		}

		if batch.Resource != nil {
			for _, a := range batch.Resource.Attributes {
				v := a.Value.GetStringValue()
				if v == "" {
					continue
				}
				for _, m := range mapping {
					if m.from == a.Key {
						buffer1[m.to] = v
					}
				}
			}
		}

		var (
			bucket      *bucket
			bucketDirty = false
		)

		for i, ss := range batch.ScopeSpans {
			for j, s := range ss.Spans {
				sz := s.Size()
				sz += protoLengthMath(sz)
				sz += batchPortion // Incrementally add 1/Nth worth of the unaccounted for batch data
				if i == 0 && j == 0 {
					sz += firstSpanPortion
				}

				// Every span can be in a different series and needs to be reset to the batch
				for k, v := range buffer1 {
					if buffer2[k] != v {
						buffer2[k] = v
						bucketDirty = true
					}
				}

				for _, a := range s.Attributes {
					v := a.Value.GetStringValue()
					if v == "" {
						continue
					}
					for _, m := range mapping {
						if m.from == a.Key && buffer2[m.to] != v {
							buffer2[m.to] = v
							bucketDirty = true
						}
					}
				}

				// Every span can be a different series.
				// If the values buffer hasn't changed then we
				// know it's the same bucket and avoid hashing again.
				// This shows up in 2 common cases:
				//  - Dimensions are only resource attributes
				//  - Runs of spans with the same attributes
				if bucket == nil || bucketDirty {
					bucket = data.getSeries(buffer2, max)
					bucketDirty = false
				}
				bucket.Inc(uint64(sz), now)
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
			if s.lastUpdated <= stale {
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

// hash the given key-value pairs buffer. sortedKeys slice is provided to ensure that
// map is always iterated in the same order.
func hash(sortedKeys []string, buffer map[string]string) uint64 {
	h := xxhash.New()

	for _, k := range sortedKeys {
		v := buffer[k]
		if v != "" {
			_, _ = h.WriteString(k)
			_, _ = h.Write([]byte{255})
			_, _ = h.WriteString(buffer[k])
			_, _ = h.Write([]byte{255})
		}
	}

	return h.Sum64()
}

// flatten the buffer map values into a slice, iterating in the given order.
func flatten(sortedKeys []string, buffer map[string]string) []string {
	outValues := make([]string, len(sortedKeys))

	for i, k := range sortedKeys {
		outValues[i] = buffer[k]
	}

	return outValues
}

// nonSpanDataLength returns the number of proto bytes in the batch
// that aren't attributable to specific spans.  It's complicated but much faster
// to do this because it ensures we only measure each part of the proto once.
// The first (and simpler) approach was to call batch.Size() and then subtract
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
