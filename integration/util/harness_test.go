package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeMaps(t *testing.T) {
	base := map[any]any{
		"distributor": map[any]any{
			"receivers": map[any]any{
				"otlp": map[any]any{
					"protocols": map[any]any{
						"grpc": map[any]any{
							"endpoint": "0.0.0.0:4317",
						},
					},
				},
			},
		},
		"server": map[any]any{
			"http_listen_port": 3200,
		},
	}

	overlay := map[any]any{
		"distributor": map[any]any{
			"retry_after_on_resource_exhausted": "1s",
		},
		"overrides": map[any]any{
			"defaults": map[any]any{
				"ingestion": map[any]any{
					"max_traces_per_user": 1,
				},
			},
		},
	}

	result := mergeMaps(base, overlay)

	// Check that base distributor.receivers is preserved
	distributor, _ := toMapAnyAny(result["distributor"])
	assert.NotNil(t, distributor["receivers"], "distributor.receivers should be preserved")

	receivers, _ := toMapAnyAny(distributor["receivers"])
	otlp, _ := toMapAnyAny(receivers["otlp"])
	protocols, _ := toMapAnyAny(otlp["protocols"])
	grpc, _ := toMapAnyAny(protocols["grpc"])
	assert.Equal(t, "0.0.0.0:4317", grpc["endpoint"], "distributor.receivers.otlp.protocols.grpc.endpoint should be preserved")

	// Check that overlay distributor.retry_after_on_resource_exhausted is added
	assert.Equal(t, "1s", distributor["retry_after_on_resource_exhausted"], "distributor.retry_after_on_resource_exhausted should be added from overlay")

	// Check that server config is preserved
	server, _ := toMapAnyAny(result["server"])
	assert.Equal(t, 3200, server["http_listen_port"], "server.http_listen_port should be preserved")

	// Check that overrides is added
	overrides, _ := toMapAnyAny(result["overrides"])
	defaults, _ := toMapAnyAny(overrides["defaults"])
	ingestion, _ := toMapAnyAny(defaults["ingestion"])
	assert.Equal(t, 1, ingestion["max_traces_per_user"], "overrides.defaults.ingestion.max_traces_per_user should be added from overlay")
}

func TestToMapAnyAny(t *testing.T) {
	// Test map[interface{}]interface{} conversion
	input := map[any]any{
		"key1": "value1",
		"key2": map[any]any{
			"nested": "value",
		},
	}

	result, ok := toMapAnyAny(input)
	assert.True(t, ok)
	assert.Equal(t, "value1", result["key1"])

	nested, ok := toMapAnyAny(result["key2"])
	assert.True(t, ok)
	assert.Equal(t, "value", nested["nested"])

	// Test map[string]any conversion
	stringMap := map[string]any{
		"key": "value",
	}
	result, ok = toMapAnyAny(stringMap)
	assert.True(t, ok)
	assert.Equal(t, "value", result["key"])

	// Test non-map
	_, ok = toMapAnyAny("not a map")
	assert.False(t, ok)
}
