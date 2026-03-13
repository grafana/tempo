package overrides

import (
	"reflect"
	"testing"
	"time"

	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/tempodb/backend"
)

// TestMergeOverrides verifies the handwritten merge methods directly.
func TestMergeOverrides(t *testing.T) {
	t.Run("nil other returns base", func(t *testing.T) {
		base := &Overrides{Ingestion: IngestionOverrides{RateLimitBytes: ptrTo(100)}}
		result := base.Merge(nil)
		require.Same(t, base, result)
	})

	t.Run("nil base returns other", func(t *testing.T) {
		other := &Overrides{Ingestion: IngestionOverrides{RateLimitBytes: ptrTo(200)}}
		result := (*Overrides)(nil).Merge(other)
		require.Same(t, other, result)
	})

	t.Run("both nil returns nil", func(t *testing.T) {
		result := (*Overrides)(nil).Merge(nil)
		require.Nil(t, result)
	})

	t.Run("value types: other non-zero wins", func(t *testing.T) {
		base := &Overrides{
			Ingestion: IngestionOverrides{
				RateLimitBytes: ptrTo(100),
				BurstSizeBytes: ptrTo(200),
			},
		}
		other := &Overrides{
			Ingestion: IngestionOverrides{
				RateLimitBytes: ptrTo(500),
				// BurstSizeBytes not set (zero)
			},
		}
		result := base.Merge(other)
		require.Equal(t, ptrTo(500), result.Ingestion.RateLimitBytes, "other wins")
		require.Equal(t, ptrTo(200), result.Ingestion.BurstSizeBytes, "base preserved when other is zero")
	})

	t.Run("pointer types: other non-nil wins, including ptrTo(0)", func(t *testing.T) {
		base := &Overrides{
			Ingestion: IngestionOverrides{
				MaxLocalTracesPerUser:  ptrTo(3000),
				MaxGlobalTracesPerUser: ptrTo(4000),
			},
			Global: GlobalOverrides{
				MaxBytesPerTrace: ptrTo(9000),
			},
		}
		other := &Overrides{
			Ingestion: IngestionOverrides{
				MaxLocalTracesPerUser: ptrTo(0), // explicitly zero
				// MaxGlobalTracesPerUser not set (nil)
			},
		}
		result := base.Merge(other)
		require.Equal(t, ptrTo(0), result.Ingestion.MaxLocalTracesPerUser, "other ptrTo(0) wins over base")
		require.Equal(t, ptrTo(4000), result.Ingestion.MaxGlobalTracesPerUser, "base preserved when other is nil")
		require.Equal(t, ptrTo(9000), result.Global.MaxBytesPerTrace, "nested struct field preserved")
	})

	t.Run("slice types: other non-nil wins, including empty slice", func(t *testing.T) {
		base := &Overrides{
			Forwarders: []string{"default-fwd"},
			Storage: StorageOverrides{
				DedicatedColumns: backend.DedicatedColumns{{Scope: "resource", Name: "ns", Type: "string"}},
			},
		}
		other := &Overrides{
			Forwarders: []string{"custom-fwd"},
			Storage: StorageOverrides{
				DedicatedColumns: backend.DedicatedColumns{}, // explicitly empty
			},
		}
		result := base.Merge(other)
		require.Equal(t, []string{"custom-fwd"}, result.Forwarders)
		require.Equal(t, backend.DedicatedColumns{}, result.Storage.DedicatedColumns, "empty slice overrides base")
	})

	t.Run("map types: other non-nil wins", func(t *testing.T) {
		base := &Overrides{
			CostAttribution: CostAttributionOverrides{
				Dimensions: map[string]string{"key1": "val1"},
			},
		}
		other := &Overrides{
			CostAttribution: CostAttributionOverrides{
				Dimensions: map[string]string{"key2": "val2"},
			},
		}
		result := base.Merge(other)
		require.Equal(t, map[string]string{"key2": "val2"}, result.CostAttribution.Dimensions, "other map replaces base map")
	})

	t.Run("zero for cannot override non-zero for non pointer types", func(t *testing.T) {
		base := &Overrides{
			MetricsGenerator: MetricsGeneratorOverrides{
				Forwarder: ForwarderOverrides{QueueSize: 100, Workers: 2},
			},
		}
		other := &Overrides{
			MetricsGenerator: MetricsGeneratorOverrides{
				Forwarder: ForwarderOverrides{QueueSize: 0, Workers: 0},
			},
		}
		result := base.Merge(other)
		require.Equal(t, 100, result.MetricsGenerator.Forwarder.QueueSize, "zero value type must not override non-zero base")
		require.Equal(t, 2, result.MetricsGenerator.Forwarder.Workers, "zero value type must not override non-zero base")
	})

	t.Run("returns a new struct without mutating inputs", func(t *testing.T) {
		base := &Overrides{
			Ingestion: IngestionOverrides{
				RateLimitBytes:        ptrTo(100),
				MaxLocalTracesPerUser: ptrTo(3000),
			},
			Forwarders: []string{"base-fwd"},
			CostAttribution: CostAttributionOverrides{
				Dimensions: map[string]string{"k1": "v1"},
			},
		}
		other := &Overrides{
			Ingestion: IngestionOverrides{
				RateLimitBytes: ptrTo(200),
			},
			Forwarders: []string{"other-fwd"},
		}

		result := base.Merge(other)

		// Result must be a distinct pointer from both base and other.
		require.NotSame(t, base, result, "result must be a new allocation, not base")
		require.NotSame(t, other, result, "result must be a new allocation, not other")

		// Result has the correct merged values.
		require.Equal(t, ptrTo(200), result.Ingestion.RateLimitBytes)
		require.Equal(t, ptrTo(3000), result.Ingestion.MaxLocalTracesPerUser)
		require.Equal(t, []string{"other-fwd"}, result.Forwarders)
		require.Equal(t, map[string]string{"k1": "v1"}, result.CostAttribution.Dimensions)

		// Inputs are unchanged.
		require.Equal(t, 100, *base.Ingestion.RateLimitBytes, "base must not be mutated")
		require.Equal(t, 3000, *base.Ingestion.MaxLocalTracesPerUser, "base must not be mutated")
		require.Equal(t, "base-fwd", base.Forwarders[0], "base must not be mutated")
		require.Equal(t, "v1", base.CostAttribution.Dimensions["k1"], "base must not be mutated")
		require.Equal(t, 200, *other.Ingestion.RateLimitBytes, "other must not be mutated")
	})
}

func TestBuildMergedOverrides(t *testing.T) {
	defaults := &Overrides{
		Ingestion: IngestionOverrides{
			RateLimitBytes:        ptrTo(1000),
			MaxLocalTracesPerUser: ptrTo(3000),
		},
		Forwarders: []string{"default-fwd"},
	}

	t.Run("nil overlay returns defaults", func(t *testing.T) {
		result := defaults.Merge(nil)
		require.Same(t, defaults, result)
	})

	t.Run("empty tenant limits", func(t *testing.T) {
		merged := buildMergedOverrides(TenantOverrides{}, defaults)
		require.Empty(t, merged)
	})

	t.Run("wildcard merged with defaults", func(t *testing.T) {
		merged := buildMergedOverrides(TenantOverrides{
			"*": {Ingestion: IngestionOverrides{RateLimitBytes: ptrTo(200)}},
		}, defaults)

		require.Len(t, merged, 1)
		wm := merged[wildcardTenant]
		require.Equal(t, ptrTo(200), wm.Ingestion.RateLimitBytes)
		require.Equal(t, ptrTo(3000), wm.Ingestion.MaxLocalTracesPerUser, "default preserved")
		require.Equal(t, []string{"default-fwd"}, wm.Forwarders, "default preserved")
	})
}

// TestMergeCoversAllFields uses reflection to verify that every field in the Overrides
// hierarchy is handled by the merge methods. If a new field is added to any struct
// but not handled in the corresponding Merge method, this test will fail.
func TestMergeCoversAllFields(t *testing.T) {
	full := generateFullOverrides()

	t.Run("full overlay onto empty base", func(t *testing.T) {
		result := (&Overrides{}).Merge(full)
		assertAllFieldsNonZero(t, reflect.ValueOf(result).Elem(), "Overrides")
	})

	t.Run("empty overlay onto full base", func(t *testing.T) {
		result := full.Merge(&Overrides{})
		assertAllFieldsNonZero(t, reflect.ValueOf(result).Elem(), "Overrides")
	})
}

// generateFullOverrides returns an Overrides with every field set to a non-zero value.
func generateFullOverrides() *Overrides {
	return &Overrides{
		Global: GlobalOverrides{
			MaxBytesPerTrace: ptrTo(1),
		},
		Ingestion: IngestionOverrides{
			RateStrategy:           GlobalIngestionRateStrategy,
			RateLimitBytes:         ptrTo(1),
			BurstSizeBytes:         ptrTo(1),
			MaxLocalTracesPerUser:  ptrTo(1),
			MaxGlobalTracesPerUser: ptrTo(1),
			TenantShardSize:        ptrTo(1),
			MaxAttributeBytes:      ptrTo(1),
			ArtificialDelay:        ptrTo(time.Second),
			RetryInfoEnabled:       ptrTo(true),
		},
		Read: ReadOverrides{
			MaxBytesPerTagValuesQuery:  ptrTo(1),
			MaxBlocksPerTagValuesQuery: ptrTo(1),
			MaxSearchDuration:          ptrTo(model.Duration(1)),
			MaxMetricsDuration:         ptrTo(model.Duration(1)),
			UnsafeQueryHints:           ptrTo(true),
			LeftPadTraceIDs:            ptrTo(true),
		},
		MetricsGenerator: MetricsGeneratorOverrides{
			RingSize:                        ptrTo(1),
			Processors:                      map[string]struct{}{"test": {}},
			MaxActiveSeries:                 ptrTo(uint32(1)),
			MaxActiveEntities:               ptrTo(uint32(1)),
			CollectionInterval:              ptrTo(time.Second),
			DisableCollection:               ptrTo(true),
			GenerateNativeHistograms:        histograms.HistogramMethodNative,
			TraceIDLabelName:                "traceID",
			RemoteWriteHeaders:              RemoteWriteHeaders{"X-Test": config.Secret("val")},
			IngestionSlack:                  ptrTo(time.Second),
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  1,
			NativeHistogramMinResetDuration: ptrTo(time.Second),
			SpanNameSanitization:            ptrTo("sanitize"),
			MaxCardinalityPerLabel:          ptrTo(uint64(1)),
			Forwarder: ForwarderOverrides{
				QueueSize: 1,
				Workers:   1,
			},
			Processor: ProcessorOverrides{
				ServiceGraphs: ServiceGraphsOverrides{
					HistogramBuckets:                      []float64{1.0},
					Dimensions:                            []string{"dim"},
					PeerAttributes:                        []string{"peer"},
					FilterPolicies:                        []filterconfig.FilterPolicy{{}},
					EnableClientServerPrefix:              ptrTo(true),
					EnableMessagingSystemLatencyHistogram: ptrTo(true),
					EnableVirtualNodeLabel:                ptrTo(true),
					SpanMultiplierKey:                     ptrTo("key"),
				},
				SpanMetrics: SpanMetricsOverrides{
					HistogramBuckets:             []float64{1.0},
					Dimensions:                   []string{"dim"},
					IntrinsicDimensions:          map[string]bool{"service": true},
					FilterPolicies:               []filterconfig.FilterPolicy{{}},
					DimensionMappings:            []sharedconfig.DimensionMappings{{}},
					EnableTargetInfo:             ptrTo(true),
					TargetInfoExcludedDimensions: []string{"dim"},
					EnableInstanceLabel:          ptrTo(true),
					SpanMultiplierKey:            ptrTo("key"),
				},
				HostInfo: HostInfoOverrides{
					HostIdentifiers: []string{"host"},
					MetricName:      "metric",
				},
			},
		},
		Forwarders: []string{"fwd"},
		Compaction: CompactionOverrides{
			BlockRetention:     ptrTo(model.Duration(1)),
			CompactionWindow:   ptrTo(model.Duration(1)),
			CompactionDisabled: ptrTo(true),
		},
		Storage: StorageOverrides{
			DedicatedColumns: backend.DedicatedColumns{{Scope: "resource", Name: "col", Type: "string"}},
		},
		CostAttribution: CostAttributionOverrides{
			MaxCardinality: ptrTo(uint64(1)),
			Dimensions:     map[string]string{"k": "v"},
		},
	}
}

// assertAllFieldsNonZero recursively walks a struct value and fails if any field is zero.
func assertAllFieldsNonZero(t *testing.T, v reflect.Value, path string) {
	for i := range v.NumField() {
		field := v.Field(i)
		fieldName := v.Type().Field(i).Name
		fieldPath := path + "." + fieldName

		switch field.Kind() {
		case reflect.Struct:
			assertAllFieldsNonZero(t, field, fieldPath)
		default:
			require.Falsef(t, field.IsZero(), "field %s is zero - add it to the Merge method and generateFullOverrides()", fieldPath)
		}
	}
}
