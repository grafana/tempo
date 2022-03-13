package registry

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/atomic"
)

// metric represents a collection of metric series with the same name and a fixed set of labels.
//
// metric is optimized for updating series instead of scraping them as we assume a metric might be
// updated thousands of times for every time it is scraped.
type metric struct {
	name   string
	labels []string

	// seriesMtx is used to sync modifications to the map, not to the data in serie
	seriesMtx sync.RWMutex
	series    map[uint64]*serie

	onAddSerie    func() bool
	onRemoveSerie func()
}

// serie represents one active serie of a metric.
type serie struct {
	// labelValues should not be modified after creation
	labelValues []string
	value       *atomic.Float64
	lastUpdated *atomic.Int64
}

func newMetric(name string, labels []string, onAddSerie func() bool, onRemoveSerie func()) *metric {
	if onAddSerie == nil {
		onAddSerie = func() bool {
			return true
		}
	}
	if onRemoveSerie == nil {
		onRemoveSerie = func() {}
	}

	return &metric{
		name:          name,
		labels:        labels,
		series:        make(map[uint64]*serie),
		onAddSerie:    onAddSerie,
		onRemoveSerie: onRemoveSerie,
	}
}

// add value to the serie identified by the given labelValues.
func (m *metric) add(labelValues []string, value float64) {
	if len(m.labels) != len(labelValues) {
		panic(fmt.Sprintf("length of given label values does not match with labels, labels: %v, label values: %v", m.labels, labelValues))
	}

	hash := hashLabelValues(labelValues)

	m.seriesMtx.RLock()
	s, ok := m.series[hash]
	m.seriesMtx.RUnlock()

	if ok {
		s.value.Add(value)
		s.lastUpdated.Store(time.Now().UnixMilli())
		return
	}

	if !m.onAddSerie() {
		return
	}

	m.seriesMtx.Lock()
	defer m.seriesMtx.Unlock()

	s, ok = m.series[hash]
	if ok {
		s.value.Add(value)
		s.lastUpdated.Store(time.Now().UnixMilli())
		return
	}

	labelValuesCopy := make([]string, len(labelValues))
	copy(labelValuesCopy, labelValues)

	m.series[hash] = &serie{
		labelValues: labelValuesCopy,
		value:       atomic.NewFloat64(value),
		lastUpdated: atomic.NewInt64(time.Now().UnixMilli()),
	}
}

// scrape the metric and write one sample for every serie into the given appender.
func (m *metric) scrape(appender storage.Appender, timeMs int64, externalLabels map[string]string) (activeSeries int, err error) {
	m.seriesMtx.RLock()
	defer m.seriesMtx.RUnlock()

	activeSeries = len(m.series)

	for _, s := range m.series {
		lbls := make(labels.Labels, 0, 1+len(externalLabels)+len(m.labels))

		// add metric name
		lbls = append(lbls, labels.Label{Name: "__name__", Value: m.name})
		// add external labels
		for name, value := range externalLabels {
			lbls = append(lbls, labels.Label{Name: name, Value: value})
		}
		// add serie specific labels
		for i, name := range m.labels {
			lbls = append(lbls, labels.Label{Name: name, Value: s.labelValues[i]})
		}

		// Prometheus labels should be sorted to be valid
		sort.Sort(lbls)

		_, err = appender.Append(0, lbls, timeMs, s.value.Load())
		if err != nil {
			return
		}

		// TODO support exemplars
	}

	return
}

// removeStaleSeries will remove all series that haven't been updated since staleTimeMs.
func (m *metric) removeStaleSeries(staleTimeMs int64) {
	m.seriesMtx.Lock()
	defer m.seriesMtx.Unlock()

	for hash, s := range m.series {
		if s.lastUpdated.Load() < staleTimeMs {
			delete(m.series, hash)
			m.onRemoveSerie()
		}
	}
}

// separatorByte is a byte that cannot occur in valid UTF-8 sequences and is
// used to separate label names, label values, and other strings from each other
// when calculating their combined hash value (aka signature aka fingerprint).
const separatorByte byte = 255

var separatorByteSlice = []byte{separatorByte} // For convenient use with xxhash.

// hashLabelValues generates a unique hash for the label values of a metrics serie. It expects that
// labelValues will always have the same length and labels can not contain new lines ('\n').
func hashLabelValues(labelValues []string) uint64 {
	// TODO add some sort of memoization:
	//  - a histogram will update several metrics each with the same label values
	//  - we often update a counter and a histogram with the same label values
	h := xxhash.New()
	for _, v := range labelValues {
		_, _ = h.WriteString(v)
		_, _ = h.Write(separatorByteSlice)
	}
	return h.Sum64()
}
