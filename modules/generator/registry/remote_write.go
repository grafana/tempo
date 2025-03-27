package registry

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/model/exemplar"
	prom_histogram "github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/metadata"
	"github.com/prometheus/prometheus/prompb"
	"github.com/prometheus/prometheus/storage"
)

// remoteWriteAppendable implements the Appendable interface for remote write endpoint
type remoteWriteAppendable struct {
	endpoint string
	client   *http.Client
	logger   log.Logger
}

// remoteWriteAppender implements the Appender interface for remote write endpoint
type remoteWriteAppender struct {
	appendable  *remoteWriteAppendable
	series      []prompb.TimeSeries
	metadata    []prompb.MetricMetadata
	seriesMutex sync.Mutex
}

var _ storage.Appendable = (*remoteWriteAppendable)(nil)
var _ storage.Appender = (*remoteWriteAppender)(nil)

func newRemoteWriteAppendable(endpoint string, logger log.Logger) (*remoteWriteAppendable, error) {
	return &remoteWriteAppendable{
		endpoint: endpoint,
		client:   &http.Client{Timeout: 30 * time.Second},
		logger:   logger,
	}, nil
}

func (r *remoteWriteAppendable) Appender(ctx context.Context) storage.Appender {
	return &remoteWriteAppender{
		appendable: r,
		series:     make([]prompb.TimeSeries, 0),
		metadata:   make([]prompb.MetricMetadata, 0),
	}
}

func (r *remoteWriteAppender) Append(ref storage.SeriesRef, l labels.Labels, t int64, v float64) (storage.SeriesRef, error) {
	r.seriesMutex.Lock()
	defer r.seriesMutex.Unlock()

	// Convert labels to prompb format
	lbls := labelsToProto(l)

	// Check if we already have this series
	var ts *prompb.TimeSeries
	for i := range r.series {
		if labelsEqual(r.series[i].Labels, lbls) {
			ts = &r.series[i]
			break
		}
	}

	// If not, create a new series
	if ts == nil {
		r.series = append(r.series, prompb.TimeSeries{
			Labels:  lbls,
			Samples: make([]prompb.Sample, 0),
		})
		ts = &r.series[len(r.series)-1]
	}

	// Add the sample
	ts.Samples = append(ts.Samples, prompb.Sample{
		Timestamp: t,
		Value:     v,
	})

	return ref, nil
}

func (r *remoteWriteAppender) AppendExemplar(ref storage.SeriesRef, l labels.Labels, ex exemplar.Exemplar) (storage.SeriesRef, error) {
	// For now, we're ignoring exemplars
	return ref, nil
}

func (r *remoteWriteAppender) AppendHistogram(ref storage.SeriesRef, l labels.Labels, t int64, h *prom_histogram.Histogram, fh *prom_histogram.FloatHistogram) (storage.SeriesRef, error) {
	// For simplicity, we're not implementing histograms in this implementation
	level.Warn(r.appendable.logger).Log("msg", "histograms are not supported in this remote write implementation")
	return ref, nil
}

func (r *remoteWriteAppender) UpdateMetadata(ref storage.SeriesRef, l labels.Labels, m metadata.Metadata) (storage.SeriesRef, error) {
	r.seriesMutex.Lock()
	defer r.seriesMutex.Unlock()

	// Extract the metric name from the labels
	var metricName string
	for _, label := range l {
		if label.Name == "__name__" {
			metricName = label.Value
			break
		}
	}

	// Convert metric name to base name (without _count, _sum, _bucket suffixes for histograms)
	baseName := metricName
	for _, suffix := range []string{"_count", "_sum", "_bucket"} {
		if len(metricName) > len(suffix) && metricName[len(metricName)-len(suffix):] == suffix {
			baseName = metricName[:len(metricName)-len(suffix)]
			break
		}
	}

	// Determine metric type based on the name or suffixes
	var metricType prompb.MetricMetadata_MetricType
	if metricName == baseName+"_count" || metricName == baseName+"_sum" || metricName == baseName+"_bucket" {
		metricType = prompb.MetricMetadata_HISTOGRAM
	} else if metricName == baseName {
		// Try to infer type from labels - presence of 'le' label usually indicates histogram bucket
		for _, label := range l {
			if label.Name == "le" {
				metricType = prompb.MetricMetadata_HISTOGRAM
				break
			}
		}
		
		if metricType == 0 {
			// Default to counter if we can't determine
			metricType = prompb.MetricMetadata_COUNTER
		}
	} else {
		metricType = prompb.MetricMetadata_UNKNOWN
	}

	// Add metadata
	r.metadata = append(r.metadata, prompb.MetricMetadata{
		Type:             metricType,
		MetricFamilyName: baseName,
		Help:             m.Help,
		Unit:             m.Unit,
	})

	return ref, nil
}

func (r *remoteWriteAppender) Commit() error {
	r.seriesMutex.Lock()
	defer r.seriesMutex.Unlock()

	// Only send if we have data
	if len(r.series) == 0 && len(r.metadata) == 0 {
		return nil
	}

	// Create write request
	req := prompb.WriteRequest{
		Timeseries: r.series,
		Metadata:   r.metadata,
	}

	// Marshal to protobuf
	data, err := proto.Marshal(&req)
	if err != nil {
		return fmt.Errorf("error marshaling write request: %w", err)
	}

	// Compress with snappy
	compressed := snappy.Encode(nil, data)

	// Create HTTP request
	httpReq, err := http.NewRequest(http.MethodPost, r.appendable.endpoint, bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %w", err)
	}

	// Set required headers
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("Content-Encoding", "snappy")
	httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	// Send request
	resp, err := r.appendable.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("error sending remote write request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("remote write request failed, status code: %d", resp.StatusCode)
	}

	// Clear the series and metadata after successful write
	r.series = make([]prompb.TimeSeries, 0)
	r.metadata = make([]prompb.MetricMetadata, 0)

	return nil
}

func (r *remoteWriteAppender) Rollback() error {
	r.seriesMutex.Lock()
	defer r.seriesMutex.Unlock()

	// Clear series and metadata
	r.series = make([]prompb.TimeSeries, 0)
	r.metadata = make([]prompb.MetricMetadata, 0)

	return nil
}

func (r *remoteWriteAppender) SetOptions(_ *storage.AppendOptions) {
	// Not needed for our implementation
}

func (r *remoteWriteAppender) AppendCTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64) (storage.SeriesRef, error) {
	// Not implementing this for now
	return 0, nil
}

func (r *remoteWriteAppender) AppendHistogramCTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64, _ *prom_histogram.Histogram, _ *prom_histogram.FloatHistogram) (storage.SeriesRef, error) {
	// Not implementing this for now
	return 0, nil
}

// Helper functions

func labelsToProto(ls labels.Labels) []prompb.Label {
	sort.Sort(ls)
	result := make([]prompb.Label, 0, len(ls))
	for _, l := range ls {
		result = append(result, prompb.Label{
			Name:  l.Name,
			Value: l.Value,
		})
	}
	return result
}

func labelsEqual(a, b []prompb.Label) bool {
	if len(a) != len(b) {
		return false
	}
	for i, labelA := range a {
		if labelA.Name != b[i].Name || labelA.Value != b[i].Value {
			return false
		}
	}
	return true
}