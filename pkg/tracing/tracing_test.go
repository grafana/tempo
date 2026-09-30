package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

// clearTracingEnv shields tests from ambient tracing env vars on dev machines.
// Empty values are treated as unset by both dskit and the OTel SDK.
func clearTracingEnv(t *testing.T) {
	t.Helper()
	for _, v := range []string{
		"OTEL_TRACES_EXPORTER", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"JAEGER_AGENT_HOST", "JAEGER_ENDPOINT", "JAEGER_SAMPLER_MANAGER_HOST_PORT",
		"OTEL_SERVICE_NAME", "JAEGER_SERVICE_NAME",
		"OTEL_TRACES_SAMPLER", "OTEL_TRACES_SAMPLER_ARG", "OTEL_PROPAGATORS", "OTEL_RESOURCE_ATTRIBUTES",
		"JAEGER_SAMPLER_TYPE", "JAEGER_SAMPLER_PARAM", "JAEGER_SAMPLING_ENDPOINT",
	} {
		t.Setenv(v, "")
	}
}

func TestInstallOTelOrJaegerFromEnvHonorsSamplerEnvVars(t *testing.T) {
	testCases := []struct {
		name        string
		sampler     string
		samplerArg  string
		wantSampled bool
	}{
		{
			name:        "always_on samples",
			sampler:     "always_on",
			wantSampled: true,
		},
		{
			name:        "always_off does not sample",
			sampler:     "always_off",
			wantSampled: false,
		},
		{
			name:        "traceidratio honors a ratio of zero",
			sampler:     "traceidratio",
			samplerArg:  "0.0",
			wantSampled: false,
		},
		{
			name:        "parentbased_jaeger_remote honors initialSamplingRate of zero",
			sampler:     "parentbased_jaeger_remote",
			samplerArg:  "endpoint=http://127.0.0.1:1/sampling,pollingIntervalMs=60000,initialSamplingRate=0.0",
			wantSampled: false,
		},
		{
			name:        "jaeger_remote samples at initialSamplingRate of one",
			sampler:     "jaeger_remote",
			samplerArg:  "endpoint=http://127.0.0.1:1/sampling,pollingIntervalMs=60000,initialSamplingRate=1.0",
			wantSampled: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clearTracingEnv(t)
			// exporter env var is required to enable tracing, "none" keeps the test hermetic
			t.Setenv("OTEL_TRACES_EXPORTER", "none")
			t.Setenv("OTEL_TRACES_SAMPLER", tc.sampler)
			if tc.samplerArg != "" {
				t.Setenv("OTEL_TRACES_SAMPLER_ARG", tc.samplerArg)
			}

			shutdown, err := InstallOTelOrJaegerFromEnv("tempo", "test", false)
			require.NoError(t, err)
			t.Cleanup(shutdown)

			_, span := otel.Tracer("test").Start(context.Background(), "op")
			defer span.End()
			require.Equal(t, tc.wantSampled, span.SpanContext().IsSampled())
		})
	}
}

func TestInstallOTelOrJaegerFromEnvRejectsInvalidJaegerRemoteSamplerArg(t *testing.T) {
	clearTracingEnv(t)
	t.Setenv("OTEL_TRACES_EXPORTER", "none")
	t.Setenv("OTEL_TRACES_SAMPLER", "parentbased_jaeger_remote")
	// missing the required endpoint
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "pollingIntervalMs=5000,initialSamplingRate=0.0")

	_, err := InstallOTelOrJaegerFromEnv("tempo", "test", false)
	require.ErrorContains(t, err, "endpoint")
}

func TestInstallOTelOrJaegerFromEnvIsNoopWithoutEnvVars(t *testing.T) {
	clearTracingEnv(t)

	shutdown, err := InstallOTelOrJaegerFromEnv("tempo", "test", false)
	require.NoError(t, err)
	shutdown()
}

func TestServiceName(t *testing.T) {
	testCases := []struct {
		name       string
		otelName   string
		jaegerName string
		want       string
	}{
		{
			name: "defaults to appName-target",
			want: "tempo-test",
		},
		{
			name:     "OTEL_SERVICE_NAME used when JAEGER_SERVICE_NAME is unset",
			otelName: "custom-otel",
			want:     "custom-otel",
		},
		{
			name:       "JAEGER_SERVICE_NAME wins",
			jaegerName: "custom-jaeger",
			want:       "custom-jaeger",
		},
		{
			name:       "JAEGER_SERVICE_NAME wins over OTEL_SERVICE_NAME",
			otelName:   "custom-otel",
			jaegerName: "custom-jaeger",
			want:       "custom-jaeger",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clearTracingEnv(t)
			t.Setenv("OTEL_SERVICE_NAME", tc.otelName)
			t.Setenv("JAEGER_SERVICE_NAME", tc.jaegerName)

			require.Equal(t, tc.want, serviceName("tempo", "test"))
		})
	}
}
