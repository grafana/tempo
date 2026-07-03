// Package tracing provides OpenTelemetry tracing setup for the application.
package tracing

import (
	"fmt"
	"os"

	"github.com/go-kit/log/level"
	dstracing "github.com/grafana/dskit/tracing"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/prometheus/common/version"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

// InstallOpenTelemetryTracer initialises the global OpenTelemetry tracer from OTel or Jaeger
// environment variables, and is a no-op when neither is configured. Delegation to dskit keeps
// sampler support (incl. jaeger_remote) consistent with Mimir.
func InstallOpenTelemetryTracer(appName, target string, spanProfiling bool) (func(), error) {
	name := serviceName(appName, target)

	opts := []dstracing.OTelOption{
		dstracing.WithResourceAttributes(
			semconv.ServiceVersionKey.String(fmt.Sprintf("%s-%s", version.Version, version.Revision)),
		),
	}
	// resource.Default() lacks host.name, keep emitting it as before
	if host, err := os.Hostname(); err == nil {
		opts = append(opts, dstracing.WithResourceAttributes(semconv.HostNameKey.String(host)))
	}
	if !spanProfiling {
		opts = append(opts, dstracing.WithPyroscopeDisabled())
	}

	closer, err := dstracing.NewOTelOrJaegerFromEnv(name, log.Logger, opts...)
	if err != nil {
		return nil, err
	}

	shutdown := func() {
		if err := closer.Close(); err != nil {
			level.Error(log.Logger).Log("msg", "OpenTelemetry trace provider failed to shutdown", "err", err)
			os.Exit(1)
		}
	}

	return shutdown, nil
}

// serviceName lets the standard env vars override the default tempo-<target> name, as Mimir does.
func serviceName(appName, target string) string {
	if name := os.Getenv("OTEL_SERVICE_NAME"); name != "" {
		return name
	}
	if name := os.Getenv("JAEGER_SERVICE_NAME"); name != "" {
		return name
	}
	return fmt.Sprintf("%s-%s", appName, target)
}
