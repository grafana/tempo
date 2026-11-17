package ingest

import (
	"testing"

	"github.com/grafana/dskit/flagext"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	"github.com/twmb/franz-go/pkg/kversion"

	"github.com/grafana/tempo/v3/pkg/util/test"
)

// apiVersionsMax returns the highest ApiVersions version a client built from
// opts may send.
func apiVersionsMax(t *testing.T, opts []kgo.Opt) int16 {
	t.Helper()

	client, err := kgo.NewClient(append(opts, kgo.SeedBrokers("localhost:9092"))...)
	require.NoError(t, err)
	t.Cleanup(client.Close)

	versions, ok := client.OptValue(kgo.MaxVersions).(*kversion.Versions)
	require.True(t, ok)
	maxVersion, ok := versions.LookupMaxKeyVersion(kmsg.ApiVersions.Int16())
	require.True(t, ok)
	return maxVersion
}

func TestAPIVersionsMaxVersionOpts(t *testing.T) {
	stableMax, ok := kversion.Stable().LookupMaxKeyVersion(kmsg.ApiVersions.Int16())
	require.True(t, ok)

	tests := []struct {
		name       string
		maxVersion int
		want       int16
	}{
		{name: "default", maxVersion: DefaultAPIVersionsMaxVersion, want: 4},
		{name: "lower cap", maxVersion: 3, want: 3},
		{name: "too high is capped", maxVersion: 5, want: 4},
		{name: "zero removes the cap", maxVersion: 0, want: stableMax},
		{name: "negative removes the cap", maxVersion: -1, want: stableMax},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, apiVersionsMax(t, apiVersionsMaxVersionOpts(tc.maxVersion)))
		})
	}
}

func TestCommonKafkaClientOptions_APIVersionsMaxVersion(t *testing.T) {
	cfg := KafkaConfig{}
	flagext.DefaultValues(&cfg)
	cfg.Topic = "test"

	opts, err := commonKafkaClientOptions(cfg, nil, test.NewTestingLogger(t))
	require.NoError(t, err)
	require.Equal(t, int16(DefaultAPIVersionsMaxVersion), apiVersionsMax(t, opts))
}

func TestKafkaConfig_Validate_APIVersionsMaxVersion(t *testing.T) {
	cfg := KafkaConfig{}
	flagext.DefaultValues(&cfg)
	cfg.Topic = "test"
	require.Equal(t, DefaultAPIVersionsMaxVersion, cfg.APIVersionsMaxVersion)

	for _, v := range []int{0, 1, 4} {
		cfg.APIVersionsMaxVersion = v
		require.NoError(t, cfg.Validate(), "value %d", v)
	}
	for _, v := range []int{-1, 5} {
		cfg.APIVersionsMaxVersion = v
		require.ErrorIs(t, cfg.Validate(), ErrInvalidAPIVersionsMaxVersion, "value %d", v)
	}
}
