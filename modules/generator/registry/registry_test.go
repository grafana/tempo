package registry

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/prometheus/prometheus/model/exemplar"
	prom_histogram "github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/metadata"
	"github.com/prometheus/prometheus/prompb"
	"github.com/prometheus/prometheus/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedRegistry_concurrency(*testing.T) {
	// Set up a test server that acts as a remote write endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	cfg := &Config{
		StaleDuration: 1 * time.Millisecond,
		RemoteWriteEndpoint: server.URL,
	}
	registry := New(cfg, &mockOverrides{}, "test", nil, log.NewNopLogger())
	defer registry.Close()

	end := make(chan struct{})

	accessor := func(f func()) {
		for {
			select {
			case <-end:
				return
			default:
				f()
			}
		}
	}

	// this goroutine constantly creates new counters
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	go accessor(func() {
		s := make([]rune, 6)
		for i := range s {
			s[i] = letters[rand.Intn(len(letters))]
		}
		registry.NewCounter(string(s), "", "")
	})

	go accessor(func() {
		registry.CollectMetrics(context.Background())
	})

	go accessor(func() {
		registry.removeStaleSeries(context.Background())
	})

	time.Sleep(200 * time.Millisecond)
	close(end)
}

func TestManagedRegistry_counter(t *testing.T) {
	appender := &capturingAppender{}

	// Set up a test server that acts as a remote write endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Forward the requests to our capturing appender for verification
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()
		
		// Decode the snappy-compressed data
		decoded, err := snappy.Decode(nil, body)
		require.NoError(t, err)
		
		// Parse the protobuf
		var req prompb.WriteRequest
		err = proto.Unmarshal(decoded, &req)
		require.NoError(t, err)
		
		// Process samples
		for _, ts := range req.Timeseries {
			// Convert labels back to model.Labels
			lset := make(labels.Labels, 0, len(ts.Labels))
			for _, l := range ts.Labels {
				lset = append(lset, labels.Label{Name: l.Name, Value: l.Value})
			}
			sort.Sort(lset)
			
			// Add samples to the capturing appender
			for _, s := range ts.Samples {
				appender.Append(0, lset, s.Timestamp, s.Value)
			}
		}
		
		// Process metadata
		for _, m := range req.Metadata {
			lset := labels.Labels{labels.Label{Name: "__name__", Value: m.MetricFamilyName}}
			appender.UpdateMetadata(0, lset, metadata.Metadata{
				Help: m.Help,
				Unit: m.Unit,
			})
		}
	}))
	defer server.Close()

	registry := New(&Config{RemoteWriteEndpoint: server.URL}, &mockOverrides{}, "test", nil, log.NewNopLogger())
	defer registry.Close()

	counter := registry.NewCounter("my_counter", "", "")

	counter.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 0.0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 1.0),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_histogram(t *testing.T) {
	appender := &capturingAppender{}

	// Set up a test server that acts as a remote write endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Forward the requests to our capturing appender for verification
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()
		
		// Decode the snappy-compressed data
		decoded, err := snappy.Decode(nil, body)
		require.NoError(t, err)
		
		// Parse the protobuf
		var req prompb.WriteRequest
		err = proto.Unmarshal(decoded, &req)
		require.NoError(t, err)
		
		// Process samples
		for _, ts := range req.Timeseries {
			// Convert labels back to model.Labels
			lset := make(labels.Labels, 0, len(ts.Labels))
			for _, l := range ts.Labels {
				lset = append(lset, labels.Label{Name: l.Name, Value: l.Value})
			}
			sort.Sort(lset)
			
			// Add samples to the capturing appender
			for _, s := range ts.Samples {
				appender.Append(0, lset, s.Timestamp, s.Value)
			}
		}
		
		// Process metadata
		for _, m := range req.Metadata {
			lset := labels.Labels{labels.Label{Name: "__name__", Value: m.MetricFamilyName}}
			appender.UpdateMetadata(0, lset, metadata.Metadata{
				Help: m.Help,
				Unit: m.Unit,
			})
		}
	}))
	defer server.Close()

	registry := New(&Config{RemoteWriteEndpoint: server.URL}, &mockOverrides{}, "test", nil, log.NewNopLogger())
	defer registry.Close()

	histogram := registry.NewHistogram("histogram", "", "", []float64{1.0, 2.0}, HistogramModeClassic)

	histogram.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0, "", 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "histogram_count", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 0),
		newSample(map[string]string{"__name__": "histogram_count", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 1, 1.0),
		newSample(map[string]string{"__name__": "histogram_sum", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 1, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "__metrics_gen_instance": mustGetHostname(), "le": "1"}, 0, 0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "__metrics_gen_instance": mustGetHostname(), "le": "1"}, 1, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "__metrics_gen_instance": mustGetHostname(), "le": "2"}, 0, 0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "__metrics_gen_instance": mustGetHostname(), "le": "2"}, 1, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "__metrics_gen_instance": mustGetHostname(), "le": "+Inf"}, 0, 0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "__metrics_gen_instance": mustGetHostname(), "le": "+Inf"}, 1, 1.0),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_removeStaleSeries(t *testing.T) {
	appender := &capturingAppender{}

	// Set up a test server that acts as a remote write endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Forward the requests to our capturing appender for verification
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()
		
		// Decode the snappy-compressed data
		decoded, err := snappy.Decode(nil, body)
		require.NoError(t, err)
		
		// Parse the protobuf
		var req prompb.WriteRequest
		err = proto.Unmarshal(decoded, &req)
		require.NoError(t, err)
		
		// Process samples
		for _, ts := range req.Timeseries {
			// Convert labels back to model.Labels
			lset := make(labels.Labels, 0, len(ts.Labels))
			for _, l := range ts.Labels {
				lset = append(lset, labels.Label{Name: l.Name, Value: l.Value})
			}
			sort.Sort(lset)
			
			// Add samples to the capturing appender
			for _, s := range ts.Samples {
				appender.Append(0, lset, s.Timestamp, s.Value)
			}
		}
		
		// Process metadata
		for _, m := range req.Metadata {
			lset := labels.Labels{labels.Label{Name: "__name__", Value: m.MetricFamilyName}}
			appender.UpdateMetadata(0, lset, metadata.Metadata{
				Help: m.Help,
				Unit: m.Unit,
			})
		}
	}))
	defer server.Close()

	cfg := &Config{
		StaleDuration: 75 * time.Millisecond,
		RemoteWriteEndpoint: server.URL,
	}
	registry := New(cfg, &mockOverrides{}, "test", nil, log.NewNopLogger())
	defer registry.Close()

	counter1 := registry.NewCounter("metric_1", "", "")
	counter2 := registry.NewCounter("metric_2", "", "")

	counter1.Inc(nil, 1)
	counter2.Inc(nil, 2)

	registry.removeStaleSeries(context.Background())

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "metric_1", "__metrics_gen_instance": mustGetHostname()}, 0, 0),
		newSample(map[string]string{"__name__": "metric_1", "__metrics_gen_instance": mustGetHostname()}, 0, 1),
		newSample(map[string]string{"__name__": "metric_2", "__metrics_gen_instance": mustGetHostname()}, 0, 0),
		newSample(map[string]string{"__name__": "metric_2", "__metrics_gen_instance": mustGetHostname()}, 0, 2),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)

	appender.samples = nil

	time.Sleep(50 * time.Millisecond)
	counter2.Inc(nil, 2)
	time.Sleep(50 * time.Millisecond)

	registry.removeStaleSeries(context.Background())

	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "metric_2", "__metrics_gen_instance": mustGetHostname()}, 0, 4),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_externalLabels(t *testing.T) {
	appender := &capturingAppender{}

	// Set up a test server that acts as a remote write endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Forward the requests to our capturing appender for verification
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()
		
		// Decode the snappy-compressed data
		decoded, err := snappy.Decode(nil, body)
		require.NoError(t, err)
		
		// Parse the protobuf
		var req prompb.WriteRequest
		err = proto.Unmarshal(decoded, &req)
		require.NoError(t, err)
		
		// Process samples
		for _, ts := range req.Timeseries {
			// Convert labels back to model.Labels
			lset := make(labels.Labels, 0, len(ts.Labels))
			for _, l := range ts.Labels {
				lset = append(lset, labels.Label{Name: l.Name, Value: l.Value})
			}
			sort.Sort(lset)
			
			// Add samples to the capturing appender
			for _, s := range ts.Samples {
				appender.Append(0, lset, s.Timestamp, s.Value)
			}
		}
		
		// Process metadata
		for _, m := range req.Metadata {
			lset := labels.Labels{labels.Label{Name: "__name__", Value: m.MetricFamilyName}}
			appender.UpdateMetadata(0, lset, metadata.Metadata{
				Help: m.Help,
				Unit: m.Unit,
			})
		}
	}))
	defer server.Close()

	cfg := &Config{
		ExternalLabels: map[string]string{
			"__foo": "bar",
		},
		RemoteWriteEndpoint: server.URL,
	}
	registry := New(cfg, &mockOverrides{}, "test", nil, log.NewNopLogger())
	defer registry.Close()

	counter := registry.NewCounter("my_counter", "", "")
	counter.Inc(nil, 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "__metrics_gen_instance": mustGetHostname(), "__foo": "bar"}, 0, 0),
		newSample(map[string]string{"__name__": "my_counter", "__metrics_gen_instance": mustGetHostname(), "__foo": "bar"}, 0, 1),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_injectTenantIDAs(t *testing.T) {
	appender := &capturingAppender{}

	// Set up a test server that acts as a remote write endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Forward the requests to our capturing appender for verification
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()
		
		// Decode the snappy-compressed data
		decoded, err := snappy.Decode(nil, body)
		require.NoError(t, err)
		
		// Parse the protobuf
		var req prompb.WriteRequest
		err = proto.Unmarshal(decoded, &req)
		require.NoError(t, err)
		
		// Process samples
		for _, ts := range req.Timeseries {
			// Convert labels back to model.Labels
			lset := make(labels.Labels, 0, len(ts.Labels))
			for _, l := range ts.Labels {
				lset = append(lset, labels.Label{Name: l.Name, Value: l.Value})
			}
			sort.Sort(lset)
			
			// Add samples to the capturing appender
			for _, s := range ts.Samples {
				appender.Append(0, lset, s.Timestamp, s.Value)
			}
		}
		
		// Process metadata
		for _, m := range req.Metadata {
			lset := labels.Labels{labels.Label{Name: "__name__", Value: m.MetricFamilyName}}
			appender.UpdateMetadata(0, lset, metadata.Metadata{
				Help: m.Help,
				Unit: m.Unit,
			})
		}
	}))
	defer server.Close()

	cfg := &Config{
		InjectTenantIDAs: "__tempo_tenant",
		RemoteWriteEndpoint: server.URL,
	}
	registry := New(cfg, &mockOverrides{}, "test", nil, log.NewNopLogger())
	defer registry.Close()

	counter := registry.NewCounter("my_counter", "", "")
	counter.Inc(nil, 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "__metrics_gen_instance": mustGetHostname(), "__tempo_tenant": "test"}, 0, 0),
		newSample(map[string]string{"__name__": "my_counter", "__metrics_gen_instance": mustGetHostname(), "__tempo_tenant": "test"}, 0, 1),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_maxSeries(t *testing.T) {
	appender := &capturingAppender{}

	// Set up a test server that acts as a remote write endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Forward the requests to our capturing appender for verification
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()
		
		// Decode the snappy-compressed data
		decoded, err := snappy.Decode(nil, body)
		require.NoError(t, err)
		
		// Parse the protobuf
		var req prompb.WriteRequest
		err = proto.Unmarshal(decoded, &req)
		require.NoError(t, err)
		
		// Process samples
		for _, ts := range req.Timeseries {
			// Convert labels back to model.Labels
			lset := make(labels.Labels, 0, len(ts.Labels))
			for _, l := range ts.Labels {
				lset = append(lset, labels.Label{Name: l.Name, Value: l.Value})
			}
			sort.Sort(lset)
			
			// Add samples to the capturing appender
			for _, s := range ts.Samples {
				appender.Append(0, lset, s.Timestamp, s.Value)
			}
		}
		
		// Process metadata
		for _, m := range req.Metadata {
			lset := labels.Labels{labels.Label{Name: "__name__", Value: m.MetricFamilyName}}
			appender.UpdateMetadata(0, lset, metadata.Metadata{
				Help: m.Help,
				Unit: m.Unit,
			})
		}
	}))
	defer server.Close()

	overrides := &mockOverrides{
		maxActiveSeries: 1,
	}
	registry := New(&Config{RemoteWriteEndpoint: server.URL}, overrides, "test", nil, log.NewNopLogger())
	defer registry.Close()

	counter1 := registry.NewCounter("metric_1", "", "")
	counter2 := registry.NewCounter("metric_2", "", "")

	counter1.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)
	// these series should be discarded
	counter1.Inc(newLabelValueCombo(nil, []string{"value-2"}), 1.0)
	counter2.Inc(nil, 1.0)

	assert.Equal(t, uint32(1), registry.activeSeries.Load())
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "metric_1", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 0),
		newSample(map[string]string{"__name__": "metric_1", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 1),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_disableCollection(t *testing.T) {
	appender := &capturingAppender{}

	// Set up a test server that acts as a remote write endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Forward the requests to our capturing appender for verification
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()
		
		// Decode the snappy-compressed data
		decoded, err := snappy.Decode(nil, body)
		require.NoError(t, err)
		
		// Parse the protobuf
		var req prompb.WriteRequest
		err = proto.Unmarshal(decoded, &req)
		require.NoError(t, err)
		
		// Process samples
		for _, ts := range req.Timeseries {
			// Convert labels back to model.Labels
			lset := make(labels.Labels, 0, len(ts.Labels))
			for _, l := range ts.Labels {
				lset = append(lset, labels.Label{Name: l.Name, Value: l.Value})
			}
			sort.Sort(lset)
			
			// Add samples to the capturing appender
			for _, s := range ts.Samples {
				appender.Append(0, lset, s.Timestamp, s.Value)
			}
		}
		
		// Process metadata
		for _, m := range req.Metadata {
			lset := labels.Labels{labels.Label{Name: "__name__", Value: m.MetricFamilyName}}
			appender.UpdateMetadata(0, lset, metadata.Metadata{
				Help: m.Help,
				Unit: m.Unit,
			})
		}
	}))
	defer server.Close()

	overrides := &mockOverrides{
		disableCollection: true,
	}
	registry := New(&Config{RemoteWriteEndpoint: server.URL}, overrides, "test", nil, log.NewNopLogger())
	defer registry.Close()

	counter := registry.NewCounter("metric_1", "", "")
	counter.Inc(nil, 1.0)

	// active series are still tracked
	assert.Equal(t, uint32(1), registry.activeSeries.Load())
	// but no samples are collected and sent out
	registry.CollectMetrics(context.Background())
	assert.Empty(t, appender.samples)
	assert.Empty(t, appender.exemplars)
}

func TestManagedRegistry_maxLabelNameLength(t *testing.T) {
	appender := &capturingAppender{}

	// Set up a test server that acts as a remote write endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Forward the requests to our capturing appender for verification
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()
		
		// Decode the snappy-compressed data
		decoded, err := snappy.Decode(nil, body)
		require.NoError(t, err)
		
		// Parse the protobuf
		var req prompb.WriteRequest
		err = proto.Unmarshal(decoded, &req)
		require.NoError(t, err)
		
		// Process samples
		for _, ts := range req.Timeseries {
			// Convert labels back to model.Labels
			lset := make(labels.Labels, 0, len(ts.Labels))
			for _, l := range ts.Labels {
				lset = append(lset, labels.Label{Name: l.Name, Value: l.Value})
			}
			sort.Sort(lset)
			
			// Add samples to the capturing appender
			for _, s := range ts.Samples {
				appender.Append(0, lset, s.Timestamp, s.Value)
			}
		}
		
		// Process metadata
		for _, m := range req.Metadata {
			lset := labels.Labels{labels.Label{Name: "__name__", Value: m.MetricFamilyName}}
			appender.UpdateMetadata(0, lset, metadata.Metadata{
				Help: m.Help,
				Unit: m.Unit,
			})
		}
	}))
	defer server.Close()

	cfg := &Config{
		MaxLabelNameLength:  8,
		MaxLabelValueLength: 5,
		RemoteWriteEndpoint: server.URL,
	}
	registry := New(cfg, &mockOverrides{}, "test", nil, log.NewNopLogger())
	defer registry.Close()

	counter := registry.NewCounter("counter", "", "")
	histogram := registry.NewHistogram("histogram", "", "", []float64{1.0}, HistogramModeClassic)

	counter.Inc(registry.NewLabelValueCombo([]string{"very_lengthy_label"}, []string{"very_length_value"}), 1.0)
	histogram.ObserveWithExemplar(registry.NewLabelValueCombo([]string{"another_very_lengthy_label"}, []string{"another_very_lengthy_value"}), 1.0, "", 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "counter", "very_len": "very_", "__metrics_gen_instance": mustGetHostname()}, 0, 0.0),
		newSample(map[string]string{"__name__": "counter", "very_len": "very_", "__metrics_gen_instance": mustGetHostname()}, 1, 1.0),
		newSample(map[string]string{"__name__": "histogram_count", "another_": "anoth", "__metrics_gen_instance": mustGetHostname()}, 0, 0),
		newSample(map[string]string{"__name__": "histogram_count", "another_": "anoth", "__metrics_gen_instance": mustGetHostname()}, 1, 1.0),
		newSample(map[string]string{"__name__": "histogram_sum", "another_": "anoth", "__metrics_gen_instance": mustGetHostname()}, 1, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "another_": "anoth", "__metrics_gen_instance": mustGetHostname(), "le": "1"}, 0, 0),
		newSample(map[string]string{"__name__": "histogram_bucket", "another_": "anoth", "__metrics_gen_instance": mustGetHostname(), "le": "1"}, 1, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "another_": "anoth", "__metrics_gen_instance": mustGetHostname(), "le": "+Inf"}, 0, 0),
		newSample(map[string]string{"__name__": "histogram_bucket", "another_": "anoth", "__metrics_gen_instance": mustGetHostname(), "le": "+Inf"}, 1, 1.0),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestValidLabelValueCombo(t *testing.T) {
	appender := &capturingAppender{}

	registry := New(&Config{}, &mockOverrides{}, "test", appender, log.NewNopLogger())
	defer registry.Close()

	assert.Panics(t, func() {
		registry.NewLabelValueCombo([]string{"one-label"}, []string{"one-value", "two-value"})
	})
}

func TestHistogramOverridesConfig(t *testing.T) {
	cases := []struct {
		name                string
		nativeHistogramMode HistogramMode
		typeOfHistogram     interface{}
	}{
		{
			"classic",
			HistogramModeClassic,
			&histogram{},
		},
		{
			"native",
			HistogramModeNative,
			&nativeHistogram{},
		},
		{
			"both",
			HistogramModeBoth,
			&nativeHistogram{},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			appender := &capturingAppender{}
			overrides := &mockOverrides{}
			
			// Set up a test server that acts as a remote write endpoint
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			
			registry := New(&Config{RemoteWriteEndpoint: server.URL}, overrides, "test", nil, log.NewNopLogger())
			defer registry.Close()

			tt := registry.NewHistogram("histogram", "", "", []float64{1.0, 2.0}, c.nativeHistogramMode)
			require.IsType(t, c.typeOfHistogram, tt)
		})
	}
}

func TestManagedRegistry_Metadata(t *testing.T) {
	appender := &capturingAppender{}

	// Set up a test server that acts as a remote write endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Forward the requests to our capturing appender for verification
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()
		
		// Decode the snappy-compressed data
		decoded, err := snappy.Decode(nil, body)
		require.NoError(t, err)
		
		// Parse the protobuf
		var req prompb.WriteRequest
		err = proto.Unmarshal(decoded, &req)
		require.NoError(t, err)
		
		// Process samples
		for _, ts := range req.Timeseries {
			// Convert labels back to model.Labels
			lset := make(labels.Labels, 0, len(ts.Labels))
			for _, l := range ts.Labels {
				lset = append(lset, labels.Label{Name: l.Name, Value: l.Value})
			}
			sort.Sort(lset)
			
			// Add samples to the capturing appender
			for _, s := range ts.Samples {
				appender.Append(0, lset, s.Timestamp, s.Value)
			}
		}
		
		// Process metadata
		for _, m := range req.Metadata {
			lset := labels.Labels{labels.Label{Name: "__name__", Value: m.MetricFamilyName}}
			appender.UpdateMetadata(0, lset, metadata.Metadata{
				Help: m.Help,
				Unit: m.Unit,
			})
		}
	}))
	defer server.Close()

	registry := New(&Config{RemoteWriteEndpoint: server.URL}, &mockOverrides{}, "test", nil, log.NewNopLogger())
	defer registry.Close()

	// Create metrics with help text and unit
	counter := registry.NewCounter("test_counter", "Help text for counter", "count")
	gauge := registry.NewGauge("test_gauge", "Help text for gauge", "bytes")
	histogram := registry.NewHistogram("test_histogram", "Help text for histogram", "seconds", []float64{1.0, 2.0}, HistogramModeClassic)

	// Use the metrics to generate series
	counter.Inc(nil, 1.0)
	gauge.Set(nil, 42.0)
	histogram.ObserveWithExemplar(nil, 1.5, "", 1.0)

	// Collect metrics and verify metadata
	registry.CollectMetrics(context.Background())

	// Verify that metadata was collected
	require.NotEmpty(t, appender.metadata, "No metadata was collected")

	// Helper function to find metadata for a given metric
	findMetadata := func(metricName string) *metadata.Metadata {
		for _, m := range appender.metadata {
			name := ""
			for _, label := range m.l {
				if label.Name == "__name__" {
					name = label.Value
					break
				}
			}
			if name == metricName {
				return &m.m
			}
		}
		return nil
	}

	// Verify counter metadata
	counterMetadata := findMetadata("test_counter")
	require.NotNil(t, counterMetadata, "Counter metadata not found")
	assert.Equal(t, "Help text for counter", counterMetadata.Help)
	assert.Equal(t, "count", counterMetadata.Unit)

	// Verify gauge metadata
	gaugeMetadata := findMetadata("test_gauge")
	require.NotNil(t, gaugeMetadata, "Gauge metadata not found")
	assert.Equal(t, "Help text for gauge", gaugeMetadata.Help)
	assert.Equal(t, "bytes", gaugeMetadata.Unit)

	// Verify histogram metadata - histograms have multiple series
	histogramCountMetadata := findMetadata("test_histogram_count")
	require.NotNil(t, histogramCountMetadata, "Histogram count metadata not found")
	assert.Equal(t, "Help text for histogram", histogramCountMetadata.Help)
	assert.Equal(t, "seconds", histogramCountMetadata.Unit)

	// Check histogram bucket metadata
	histogramBucketMetadata := findMetadata("test_histogram_bucket")
	require.NotNil(t, histogramBucketMetadata, "Histogram bucket metadata not found")
	assert.Equal(t, "Help text for histogram", histogramBucketMetadata.Help)
	assert.Equal(t, "seconds", histogramBucketMetadata.Unit)
}

func TestManagedRegistry_MetadataSendOnce(t *testing.T) {
	appender := &capturingAppender{}

	// Set up a test server that acts as a remote write endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Forward the requests to our capturing appender for verification
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()
		
		// Decode the snappy-compressed data
		decoded, err := snappy.Decode(nil, body)
		require.NoError(t, err)
		
		// Parse the protobuf
		var req prompb.WriteRequest
		err = proto.Unmarshal(decoded, &req)
		require.NoError(t, err)
		
		// Process samples
		for _, ts := range req.Timeseries {
			// Convert labels back to model.Labels
			lset := make(labels.Labels, 0, len(ts.Labels))
			for _, l := range ts.Labels {
				lset = append(lset, labels.Label{Name: l.Name, Value: l.Value})
			}
			sort.Sort(lset)
			
			// Add samples to the capturing appender
			for _, s := range ts.Samples {
				appender.Append(0, lset, s.Timestamp, s.Value)
			}
		}
		
		// Process metadata
		for _, m := range req.Metadata {
			lset := labels.Labels{labels.Label{Name: "__name__", Value: m.MetricFamilyName}}
			appender.UpdateMetadata(0, lset, metadata.Metadata{
				Help: m.Help,
				Unit: m.Unit,
			})
		}
	}))
	defer server.Close()

	registry := New(&Config{RemoteWriteEndpoint: server.URL}, &mockOverrides{}, "test", nil, log.NewNopLogger())
	defer registry.Close()

	// Create a gauge and update it multiple times
	gauge := registry.NewGauge("test_gauge", "Help text for gauge", "bytes")
	gauge.Set(nil, 42.0)

	// First collection - should include metadata
	registry.CollectMetrics(context.Background())
	initialMetadataCount := len(appender.metadata)
	require.NotZero(t, initialMetadataCount, "No metadata collected in first collection")

	// Keep track of metadata sent for the gauge
	countMetadataForGauge := 0
	for _, m := range appender.metadata {
		name := ""
		for _, label := range m.l {
			if label.Name == "__name__" {
				name = label.Value
				break
			}
		}
		if name == "test_gauge" {
			countMetadataForGauge++
		}
	}
	require.Equal(t, 1, countMetadataForGauge, "Expected exactly one metadata entry for gauge")

	// Reset appender
	appender.metadata = nil

	// Update the gauge again and collect
	gauge.Set(nil, 84.0)
	registry.CollectMetrics(context.Background())

	// Verify no new metadata was sent for the existing series
	for _, m := range appender.metadata {
		name := ""
		for _, label := range m.l {
			if label.Name == "__name__" {
				name = label.Value
				break
			}
		}
		require.NotEqual(t, "test_gauge", name, "Should not have sent metadata again for existing gauge series")
	}

	// Reset appender again
	appender.metadata = nil

	// Now create a new series with different labels
	gauge.Set(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 100.0)
	registry.CollectMetrics(context.Background())

	// Verify metadata was sent for the new series
	newSeriesMetadataFound := false
	for _, m := range appender.metadata {
		name := ""
		hasLabel := false
		for _, label := range m.l {
			if label.Name == "__name__" && label.Value == "test_gauge" {
				name = label.Value
			}
			if label.Name == "label" && label.Value == "value-1" {
				hasLabel = true
			}
		}
		if name == "test_gauge" && hasLabel {
			newSeriesMetadataFound = true
			assert.Equal(t, "Help text for gauge", m.m.Help)
			assert.Equal(t, "bytes", m.m.Unit)
		}
	}
	assert.True(t, newSeriesMetadataFound, "Metadata for new gauge series with new labels not found")
}

func TestCounter_MetadataSendOnce(t *testing.T) {
	appender := &capturingAppender{}

	// Set up a test server that acts as a remote write endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Forward the requests to our capturing appender for verification
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()
		
		// Decode the snappy-compressed data
		decoded, err := snappy.Decode(nil, body)
		require.NoError(t, err)
		
		// Parse the protobuf
		var req prompb.WriteRequest
		err = proto.Unmarshal(decoded, &req)
		require.NoError(t, err)
		
		// Process samples
		for _, ts := range req.Timeseries {
			// Convert labels back to model.Labels
			lset := make(labels.Labels, 0, len(ts.Labels))
			for _, l := range ts.Labels {
				lset = append(lset, labels.Label{Name: l.Name, Value: l.Value})
			}
			sort.Sort(lset)
			
			// Add samples to the capturing appender
			for _, s := range ts.Samples {
				appender.Append(0, lset, s.Timestamp, s.Value)
			}
		}
		
		// Process metadata
		for _, m := range req.Metadata {
			lset := labels.Labels{labels.Label{Name: "__name__", Value: m.MetricFamilyName}}
			appender.UpdateMetadata(0, lset, metadata.Metadata{
				Help: m.Help,
				Unit: m.Unit,
			})
		}
	}))
	defer server.Close()

	registry := New(&Config{RemoteWriteEndpoint: server.URL}, &mockOverrides{}, "test", nil, log.NewNopLogger())
	defer registry.Close()

	// Create a counter with help text and unit
	counter := registry.NewCounter("test_counter", "Help text for counter", "operations")
	
	// Use the counter to generate series
	counter.Inc(nil, 1.0)
	
	// Collect metrics and verify metadata
	registry.CollectMetrics(context.Background())
	
	// Verify that metadata was collected
	require.NotEmpty(t, appender.metadata, "No metadata was collected")
	
	// Helper function to find metadata for a given metric
	findMetadata := func(metricName string) *metadata.Metadata {
		for _, m := range appender.metadata {
			name := ""
			for _, label := range m.l {
				if label.Name == "__name__" {
					name = label.Value
					break
				}
			}
			if name == metricName {
				return &m.m
			}
		}
		return nil
	}
	
	// Verify counter metadata
	counterMetadata := findMetadata("test_counter")
	require.NotNil(t, counterMetadata, "Counter metadata not found")
	assert.Equal(t, "Help text for counter", counterMetadata.Help)
	assert.Equal(t, "operations", counterMetadata.Unit)
	
	// Reset appender
	appender.metadata = nil
	
	// Update the counter again and collect
	counter.Inc(nil, 2.0)
	registry.CollectMetrics(context.Background())
	
	// Verify no new metadata was sent for the existing series
	for _, m := range appender.metadata {
		name := ""
		for _, label := range m.l {
			if label.Name == "__name__" {
				name = label.Value
				break
			}
		}
		require.NotEqual(t, "test_counter", name, "Should not have sent metadata again for existing counter series")
	}
	
	// Reset appender again
	appender.metadata = nil
	
	// Now create a new series with different labels
	counter.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 3.0)
	registry.CollectMetrics(context.Background())
	
	// Verify metadata was sent for the new series
	newSeriesMetadataFound := false
	for _, m := range appender.metadata {
		name := ""
		hasLabel := false
		for _, label := range m.l {
			if label.Name == "__name__" && label.Value == "test_counter" {
				name = label.Value
			}
			if label.Name == "label" && label.Value == "value-1" {
				hasLabel = true
			}
		}
		if name == "test_counter" && hasLabel {
			newSeriesMetadataFound = true
			assert.Equal(t, "Help text for counter", m.m.Help)
			assert.Equal(t, "operations", m.m.Unit)
		}
	}
	assert.True(t, newSeriesMetadataFound, "Metadata for new counter series with new labels not found")
}

func TestHistogram_MetadataMultipleSeries(t *testing.T) {
	appender := &capturingAppender{}

	// Set up a test server that acts as a remote write endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Forward the requests to our capturing appender for verification
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()
		
		// Decode the snappy-compressed data
		decoded, err := snappy.Decode(nil, body)
		require.NoError(t, err)
		
		// Parse the protobuf
		var req prompb.WriteRequest
		err = proto.Unmarshal(decoded, &req)
		require.NoError(t, err)
		
		// Process samples
		for _, ts := range req.Timeseries {
			// Convert labels back to model.Labels
			lset := make(labels.Labels, 0, len(ts.Labels))
			for _, l := range ts.Labels {
				lset = append(lset, labels.Label{Name: l.Name, Value: l.Value})
			}
			sort.Sort(lset)
			
			// Add samples to the capturing appender
			for _, s := range ts.Samples {
				appender.Append(0, lset, s.Timestamp, s.Value)
			}
		}
		
		// Process metadata
		for _, m := range req.Metadata {
			lset := labels.Labels{labels.Label{Name: "__name__", Value: m.MetricFamilyName}}
			appender.UpdateMetadata(0, lset, metadata.Metadata{
				Help: m.Help,
				Unit: m.Unit,
			})
		}
	}))
	defer server.Close()

	registry := New(&Config{RemoteWriteEndpoint: server.URL}, &mockOverrides{}, "test", nil, log.NewNopLogger())
	defer registry.Close()

	// Create a histogram with help text and unit
	histogram := registry.NewHistogram("test_histogram", "Help text for histogram", "seconds", []float64{1.0, 2.0}, HistogramModeClassic)

	// Use the histogram to generate series
	histogram.ObserveWithExemplar(nil, 1.5, "", 1.0)

	// Collect metrics and verify metadata
	registry.CollectMetrics(context.Background())

	// Verify that metadata was collected
	require.NotEmpty(t, appender.metadata, "No metadata was collected")

	// Helper function to find metadata for a given metric
	findMetadata := func(metricName string) *metadata.Metadata {
		for _, m := range appender.metadata {
			name := ""
			for _, label := range m.l {
				if label.Name == "__name__" {
					name = label.Value
					break
				}
			}
			if name == metricName {
				return &m.m
			}
		}
		return nil
	}

	// Verify histogram metadata components
	histogramCountMetadata := findMetadata("test_histogram_count")
	require.NotNil(t, histogramCountMetadata, "Histogram count metadata not found")
	assert.Equal(t, "Help text for histogram", histogramCountMetadata.Help)
	assert.Equal(t, "seconds", histogramCountMetadata.Unit)

	// Reset appender
	appender.metadata = nil

	// Update the histogram again and collect
	histogram.ObserveWithExemplar(nil, 2.5, "", 1.0)
	registry.CollectMetrics(context.Background())

	// Verify no new metadata was sent for the existing series
	for _, m := range appender.metadata {
		name := ""
		for _, label := range m.l {
			if label.Name == "__name__" {
				name = label.Value
				break
			}
		}
		require.NotEqual(t, "test_histogram_count", name, "Should not have sent metadata again for existing histogram count series")
	}

	// Reset appender again
	appender.metadata = nil

	// Now create a new series with different labels
	histogram.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 3.5, "", 1.0)
	registry.CollectMetrics(context.Background())

	// Verify metadata was sent for the new series
	newSeriesMetadataFound := false
	for _, m := range appender.metadata {
		name := ""
		hasLabel := false
		for _, label := range m.l {
			if label.Name == "__name__" && label.Value == "test_histogram_count" {
				name = label.Value
			}
			if label.Name == "label" && label.Value == "value-1" {
				hasLabel = true
			}
		}
		if name == "test_histogram_count" && hasLabel {
			newSeriesMetadataFound = true
			assert.Equal(t, "Help text for histogram", m.m.Help)
			assert.Equal(t, "seconds", m.m.Unit)
		}
	}
	assert.True(t, newSeriesMetadataFound, "Metadata for new histogram series with new labels not found")
}

func TestMetadataEndToEnd(t *testing.T) {
	// Create a special capturing appender that tracks metadata
	appender := &endToEndAppender{}
	
	registry := New(&Config{}, &mockOverrides{}, "test", appender, log.NewNopLogger())
	defer registry.Close()
	
	// Create metrics with metadata
	counter := registry.NewCounter("test_counter", "Help text for counter", "count")
	gauge := registry.NewGauge("test_gauge", "Help text for gauge", "bytes")
	histogram := registry.NewHistogram("test_histogram", "Help text for histogram", "seconds", []float64{1.0, 2.0}, HistogramModeClassic)
	
	// Use metrics to generate samples
	counter.Inc(nil, 1.0)
	gauge.Set(nil, 42.0)
	histogram.ObserveWithExemplar(nil, 1.5, "", 1.0)
	
	// Collect metrics, which should trigger calls to the appender
	registry.CollectMetrics(context.Background())
	
	// Verify that appropriate metadata updates were called
	require.GreaterOrEqual(t, len(appender.updatedMetadata), 3, "Expected at least 3 metadata updates")
	
	// Create maps to check metadata for each metric type
	metadataByMetric := make(map[string]metadata.Metadata)
	for _, md := range appender.updatedMetadata {
		name := getMetricNameFromLabels(md.l)
		metadataByMetric[name] = md.m
	}
	
	// Verify counter metadata
	counterMd, ok := metadataByMetric["test_counter"]
	require.True(t, ok, "Counter metadata not received")
	assert.Equal(t, "Help text for counter", counterMd.Help)
	assert.Equal(t, "count", counterMd.Unit)
	
	// Verify gauge metadata
	gaugeMd, ok := metadataByMetric["test_gauge"] 
	require.True(t, ok, "Gauge metadata not received")
	assert.Equal(t, "Help text for gauge", gaugeMd.Help)
	assert.Equal(t, "bytes", gaugeMd.Unit)
	
	// Verify histogram metadata components
	histogramCountMd, ok := metadataByMetric["test_histogram_count"]
	require.True(t, ok, "Histogram count metadata not received")
	assert.Equal(t, "Help text for histogram", histogramCountMd.Help)
	assert.Equal(t, "seconds", histogramCountMd.Unit)

	// Verify bucket metadata
	bucketMd, ok := metadataByMetric["test_histogram_bucket"]
	require.True(t, ok, "Histogram bucket metadata not received")
	assert.Equal(t, "Help text for histogram", bucketMd.Help)
	assert.Equal(t, "seconds", bucketMd.Unit)
}

// TestMetadataHttpEndToEnd tests that metadata is correctly captured in prometheus remote write format
func TestMetadataHttpEndToEnd(t *testing.T) {
	// Create a test server to verify metadata is properly encoded in protobuf format
	receivedRequests := 0
	totalMetadata := 0
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read request body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()
		
		// Check content type and encoding
		require.Equal(t, "application/x-protobuf", r.Header.Get("Content-Type"))
		require.Equal(t, "snappy", r.Header.Get("Content-Encoding"))
		
		// Decode snappy-compressed data
		decoded, err := snappy.Decode(nil, body)
		require.NoError(t, err)
		
		// Parse protobuf
		var req prompb.WriteRequest
		err = proto.Unmarshal(decoded, &req)
		require.NoError(t, err)
		
		// Count metadata entries
		totalMetadata += len(req.Metadata)
		receivedRequests++
		
		// Verify some metadata is present - we expect at least 3 (counter, gauge, histogram)
		if len(req.Metadata) >= 3 {
			foundCounter := false
			foundGauge := false
			foundHistogram := false
			
			for _, m := range req.Metadata {
				// Check for our specific metric names
				if m.MetricFamilyName == "test_counter" {
					foundCounter = true
					assert.Equal(t, "Help text for counter", m.Help)
					assert.Equal(t, "count", m.Unit)
				}
				if m.MetricFamilyName == "test_gauge" {
					foundGauge = true
					assert.Equal(t, "Help text for gauge", m.Help)
					assert.Equal(t, "bytes", m.Unit)
				}
				if m.MetricFamilyName == "test_histogram" {
					foundHistogram = true
					assert.Equal(t, "Help text for histogram", m.Help)
					assert.Equal(t, "seconds", m.Unit)
				}
			}
			
			assert.True(t, foundCounter, "Counter metadata not found in protobuf")
			assert.True(t, foundGauge, "Gauge metadata not found in protobuf")
			assert.True(t, foundHistogram, "Histogram metadata not found in protobuf")
		}
		
		// Respond with success
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	// Create a specialized endToEndMetadataAppender 
	appender := &endToEndMetadataAppender{
		t:           t,
		serverURL:   server.URL,
		collectCall: make(chan struct{}, 10),
	}
	
	// Create registry with our test appender
	registry := New(&Config{}, &mockOverrides{}, "test", appender, log.NewNopLogger())
	defer registry.Close()
	
	// Create metrics with help text and unit information
	counter := registry.NewCounter("test_counter", "Help text for counter", "count")
	gauge := registry.NewGauge("test_gauge", "Help text for gauge", "bytes")
	histogram := registry.NewHistogram("test_histogram", "Help text for histogram", "seconds", []float64{1.0, 2.0}, HistogramModeClassic)
	
	// Use the metrics to generate series
	counter.Inc(nil, 1.0)
	gauge.Set(nil, 42.0)
	histogram.ObserveWithExemplar(nil, 1.5, "", 1.0)
	
	// Collect metrics - this should trigger the appender to send the data
	registry.CollectMetrics(context.Background())
	
	// Wait for collect to be called
	select {
	case <-appender.collectCall:
		// Collect call received, good
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for collect call")
	}
	
	// Verify that at least one request was received by the server
	require.GreaterOrEqual(t, receivedRequests, 1, "No requests received by test server")
	require.GreaterOrEqual(t, totalMetadata, 3, "Not enough metadata entries received")
}

// endToEndMetadataAppender is a specialized appender for the HTTP test.
// It captures metadata and forwards to a test HTTP server.
type endToEndMetadataAppender struct {
	t           *testing.T
	serverURL   string
	collectCall chan struct{}
	series      []prompb.TimeSeries
	metadata    []prompb.MetricMetadata
}

var _ storage.Appendable = (*endToEndMetadataAppender)(nil)
var _ storage.Appender = (*endToEndMetadataAppender)(nil)

func (e *endToEndMetadataAppender) Appender(context.Context) storage.Appender {
	return e
}

func (e *endToEndMetadataAppender) Append(ref storage.SeriesRef, l labels.Labels, t int64, v float64) (storage.SeriesRef, error) {
	// Convert labels to protobuf format
	var lbls []prompb.Label
	for _, label := range l {
		lbls = append(lbls, prompb.Label{
			Name:  label.Name,
			Value: label.Value,
		})
	}
	
	// Add a new time series
	e.series = append(e.series, prompb.TimeSeries{
		Labels: lbls,
		Samples: []prompb.Sample{
			{Timestamp: t, Value: v},
		},
	})
	
	return ref, nil
}

func (e *endToEndMetadataAppender) AppendExemplar(ref storage.SeriesRef, l labels.Labels, ex exemplar.Exemplar) (storage.SeriesRef, error) {
	return ref, nil
}

func (e *endToEndMetadataAppender) AppendHistogram(ref storage.SeriesRef, l labels.Labels, t int64, h *prom_histogram.Histogram, fh *prom_histogram.FloatHistogram) (storage.SeriesRef, error) {
	return ref, nil
}

func (e *endToEndMetadataAppender) UpdateMetadata(ref storage.SeriesRef, l labels.Labels, m metadata.Metadata) (storage.SeriesRef, error) {
	// Extract the metric name
	var metricName string
	for _, label := range l {
		if label.Name == "__name__" {
			metricName = label.Value
			break
		}
	}
	
	// Convert metric name to base name (without _count, _sum, _bucket suffixes)
	baseName := metricName
	if strings.HasSuffix(metricName, "_count") || 
	   strings.HasSuffix(metricName, "_sum") ||
	   strings.HasSuffix(metricName, "_bucket") {
		parts := strings.Split(metricName, "_")
		if len(parts) > 1 {
			last := parts[len(parts)-1]
			if last == "count" || last == "sum" || last == "bucket" {
				baseName = strings.Join(parts[:len(parts)-1], "_")
			}
		}
	}
	
	// Convert to protobuf MetricMetadata
	var mdType prompb.MetricMetadata_MetricType
	// Determine the metric type from metric name since m.Type may not be set correctly
	if strings.Contains(metricName, "counter") {
		mdType = prompb.MetricMetadata_COUNTER
	} else if strings.Contains(metricName, "gauge") {
		mdType = prompb.MetricMetadata_GAUGE
	} else if strings.Contains(metricName, "histogram") {
		mdType = prompb.MetricMetadata_HISTOGRAM
	} else {
		mdType = prompb.MetricMetadata_UNKNOWN
	}
	
	// Add to metadata list
	e.metadata = append(e.metadata, prompb.MetricMetadata{
		Type:             mdType,
		MetricFamilyName: baseName,
		Help:             m.Help,
		Unit:             m.Unit,
	})
	
	return ref, nil
}

func (e *endToEndMetadataAppender) Commit() error {
	// Signal that Commit was called
	e.collectCall <- struct{}{}
	
	// Only send if we have data to send
	if len(e.series) == 0 && len(e.metadata) == 0 {
		return nil
	}
	
	// Create the remote write request
	req := prompb.WriteRequest{
		Timeseries: e.series,
		Metadata:   e.metadata,
	}
	
	// Marshal to protobuf
	data, err := proto.Marshal(&req)
	if err != nil {
		return err
	}
	
	// Compress with snappy
	compressed := snappy.Encode(nil, data)
	
	// Send HTTP request
	httpReq, err := http.NewRequest("POST", e.serverURL, bytes.NewReader(compressed))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("Content-Encoding", "snappy")
	
	// Send the request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// Reset data
	e.series = nil
	e.metadata = nil
	
	return nil
}

func (e *endToEndMetadataAppender) Rollback() error {
	e.series = nil
	e.metadata = nil
	return nil
}

func (e *endToEndMetadataAppender) SetOptions(_ *storage.AppendOptions) {}

func (e *endToEndMetadataAppender) AppendCTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64) (storage.SeriesRef, error) {
	return 0, nil
}

func (e *endToEndMetadataAppender) AppendHistogramCTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64, _ *prom_histogram.Histogram, _ *prom_histogram.FloatHistogram) (storage.SeriesRef, error) {
	return 0, nil
}

// endToEndAppender is a special appender that captures metadata updates for testing
type endToEndAppender struct {
	updatedMetadata []metadataSample
}

var _ storage.Appendable = (*endToEndAppender)(nil)
var _ storage.Appender = (*endToEndAppender)(nil)

func (e *endToEndAppender) Appender(context.Context) storage.Appender {
	return e
}

func (e *endToEndAppender) Append(ref storage.SeriesRef, l labels.Labels, t int64, v float64) (storage.SeriesRef, error) {
	return ref, nil
}

func (e *endToEndAppender) AppendExemplar(ref storage.SeriesRef, l labels.Labels, ex exemplar.Exemplar) (storage.SeriesRef, error) {
	return ref, nil
}

func (e *endToEndAppender) AppendHistogram(ref storage.SeriesRef, l labels.Labels, t int64, h *prom_histogram.Histogram, fh *prom_histogram.FloatHistogram) (storage.SeriesRef, error) {
	return ref, nil
}

func (e *endToEndAppender) Commit() error {
	return nil
}

func (e *endToEndAppender) Rollback() error {
	return nil
}

func (e *endToEndAppender) SetOptions(_ *storage.AppendOptions) {}

func (e *endToEndAppender) UpdateMetadata(ref storage.SeriesRef, l labels.Labels, m metadata.Metadata) (storage.SeriesRef, error) {
	e.updatedMetadata = append(e.updatedMetadata, metadataSample{l, m})
	return ref, nil
}

func (e *endToEndAppender) AppendCTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64) (storage.SeriesRef, error) {
	return 0, nil
}

func (e *endToEndAppender) AppendHistogramCTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64, _ *prom_histogram.Histogram, _ *prom_histogram.FloatHistogram) (storage.SeriesRef, error) {
	return 0, nil
}

// Helper function to get the metric name from labels
func getMetricNameFromLabels(ls labels.Labels) string {
	for _, l := range ls {
		if l.Name == "__name__" {
			return l.Value
		}
	}
	return ""
}

func collectRegistryMetricsAndAssert(t *testing.T, r *ManagedRegistry, appender *capturingAppender, expectedSamples []sample) {
	collectionTimeMs := time.Now().UnixMilli()
	r.CollectMetrics(context.Background())

	// Ignore the collection time on expected samples, since we won't know when the collection will actually take place.
	for i := range expectedSamples {
		expectedSamples[i].t = collectionTimeMs
	}

	// Ignore the collection time on the collected samples.  Initial counter values will be offset from the collection time.
	for i := range appender.samples {
		appender.samples[i].t = collectionTimeMs
	}

	assert.Equal(t, true, appender.isCommitted)
	assert.Equal(t, false, appender.isRolledback)

	actualSamples := appender.samples
	require.Equal(t, len(expectedSamples), len(actualSamples))

	// Ensure that both slices are ordered consistently.
	for _, slice := range [][]sample{expectedSamples, actualSamples} {
		sort.Slice(slice, func(i, j int) bool {
			this := slice[i]
			next := slice[j]

			// The actual order doesn't matter, the only thing that matters is that it is consistent.
			return this.String() < next.String()
		})
	}

	for i, expected := range expectedSamples {
		actual := actualSamples[i]

		assert.Equal(t, expected.t, actual.t)
		assert.Equal(t, expected.v, actual.v)
		// Rely on the fact that Go prints map keys in sorted order.
		// See https://tip.golang.org/doc/go1.12#fmt.
		assert.Equal(t, fmt.Sprint(expected.l.Map()), fmt.Sprint(actual.l.Map()))
	}
}

type mockOverrides struct {
	maxActiveSeries          uint32
	disableCollection        bool
	generateNativeHistograms overrides.HistogramMethod
}

var _ Overrides = (*mockOverrides)(nil)

func (m *mockOverrides) MetricsGeneratorMaxActiveSeries(string) uint32 {
	return m.maxActiveSeries
}

func (m *mockOverrides) MetricsGeneratorCollectionInterval(string) time.Duration {
	return 15 * time.Second
}

func (m *mockOverrides) MetricsGeneratorDisableCollection(string) bool {
	return m.disableCollection
}

func (m *mockOverrides) MetricsGeneratorGenerateNativeHistograms(_ string) overrides.HistogramMethod {
	return m.generateNativeHistograms
}

func (m *mockOverrides) MetricsGenerationTraceIDLabelName(string) string {
	return ""
}

func mustGetHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}
