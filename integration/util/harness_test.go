package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeMaps(t *testing.T) {
	base := map[string]any{
		"distributor": map[string]any{
			"receivers": map[string]any{
				"otlp": map[string]any{
					"protocols": map[string]any{
						"grpc": map[string]any{
							"endpoint": "0.0.0.0:4317",
						},
					},
				},
			},
		},
		"server": map[string]any{
			"http_listen_port": 3200,
		},
	}

	overlay := map[string]any{
		"distributor": map[string]any{
			"retry_after_on_resource_exhausted": "1s",
		},
		"overrides": map[string]any{
			"defaults": map[string]any{
				"ingestion": map[string]any{
					"max_traces_per_user": 1,
				},
			},
		},
	}

	result := mergeMaps(base, overlay)

	// Check that base distributor.receivers is preserved
	distributor := result["distributor"].(map[string]any)
	assert.NotNil(t, distributor["receivers"], "distributor.receivers should be preserved")

	receivers := distributor["receivers"].(map[string]any)
	otlp := receivers["otlp"].(map[string]any)
	protocols := otlp["protocols"].(map[string]any)
	grpc := protocols["grpc"].(map[string]any)
	assert.Equal(t, "0.0.0.0:4317", grpc["endpoint"], "distributor.receivers.otlp.protocols.grpc.endpoint should be preserved")

	// Check that overlay distributor.retry_after_on_resource_exhausted is added
	assert.Equal(t, "1s", distributor["retry_after_on_resource_exhausted"], "distributor.retry_after_on_resource_exhausted should be added from overlay")

	// Check that server config is preserved
	server := result["server"].(map[string]any)
	assert.Equal(t, 3200, server["http_listen_port"], "server.http_listen_port should be preserved")

	// Check that overrides is added
	overrides := result["overrides"].(map[string]any)
	defaults := overrides["defaults"].(map[string]any)
	ingestion := defaults["ingestion"].(map[string]any)
	assert.Equal(t, 1, ingestion["max_traces_per_user"], "overrides.defaults.ingestion.max_traces_per_user should be added from overlay")
}
