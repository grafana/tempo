// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf/v0.3.0"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

var zeroScope instrumentation.Scope

const instrumentKindUndefined = sdkmetric.InstrumentKind(0)

func meterProvider(cfg configOptions, res *resource.Resource) (metric.MeterProvider, shutdownFunc, error) {
	if cfg.opentelemetryConfig.MeterProvider == nil {
		return noop.NewMeterProvider(), noopShutdown, nil
	}
	opts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
	}

	var errs []error
	for _, reader := range cfg.opentelemetryConfig.MeterProvider.Readers {
		r, err := metricReader(cfg.ctx, reader)
		if err == nil {
			opts = append(opts, sdkmetric.WithReader(r))
		} else {
			errs = append(errs, err)
		}
	}
	for _, vw := range cfg.opentelemetryConfig.MeterProvider.Views {
		v, err := view(vw)
		if err == nil {
			opts = append(opts, sdkmetric.WithView(v))
		} else {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return noop.NewMeterProvider(), noopShutdown, errors.Join(errs...)
	}

	mp := sdkmetric.NewMeterProvider(opts...)
	return mp, mp.Shutdown, nil
}

func metricReader(ctx context.Context, r MetricReader) (sdkmetric.Reader, error) {
	if r.Periodic != nil && r.Pull != nil {
		return nil, errors.New("must not specify multiple metric reader type")
	}

	if r.Periodic != nil {
		var opts []sdkmetric.PeriodicReaderOption
		if r.Periodic.Interval != nil {
			opts = append(opts, sdkmetric.WithInterval(time.Duration(*r.Periodic.Interval)*time.Millisecond))
		}

		if r.Periodic.Timeout != nil {
			opts = append(opts, sdkmetric.WithTimeout(time.Duration(*r.Periodic.Timeout)*time.Millisecond))
		}
		return periodicExporter(ctx, r.Periodic.Exporter, opts...)
	}

	if r.Pull != nil {
		return pullReader(ctx, r.Pull.Exporter)
	}
	return nil, errors.New("no valid metric reader")
}

func pullReader(ctx context.Context, exporter PullMetricExporter) (sdkmetric.Reader, error) {
	if exporter.Prometheus != nil {
		return prometheusReader(ctx, exporter.Prometheus)
	}
	return nil, errors.New("no valid metric exporter")
}

func periodicExporter(ctx context.Context, exporter PushMetricExporter, opts ...sdkmetric.PeriodicReaderOption) (sdkmetric.Reader, error) {
	if exporter.Console != nil && exporter.OTLP != nil {
		return nil, errors.New("must not specify multiple exporters")
	}
	if exporter.Console != nil {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		exp, err := stdoutmetric.New(
			stdoutmetric.WithEncoder(enc),
		)
		if err != nil {
			return nil, err
		}
		return sdkmetric.NewPeriodicReader(exp, opts...), nil
	}
	if exporter.OTLP != nil && exporter.OTLP.Protocol != nil {
		var err error
		var exp sdkmetric.Exporter
		switch *exporter.OTLP.Protocol {
		case protocolProtobufHTTP:
			exp, err = otlpHTTPMetricExporter(ctx, exporter.OTLP)
		case protocolProtobufGRPC:
			exp, err = otlpGRPCMetricExporter(ctx, exporter.OTLP)
		default:
			return nil, fmt.Errorf("unsupported protocol %q", *exporter.OTLP.Protocol)
		}
		if err != nil {
			return nil, err
		}
		return sdkmetric.NewPeriodicReader(exp, opts...), nil
	}
	return nil, errors.New("no valid metric exporter")
}

func otlpHTTPMetricExporter(ctx context.Context, otlpConfig *OTLPMetric) (sdkmetric.Exporter, error) {
	opts := []otlpmetrichttp.Option{}

	if otlpConfig.Endpoint != nil {
		u, err := url.ParseRequestURI(*otlpConfig.Endpoint)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlpmetrichttp.WithEndpoint(u.Host))

		if u.Scheme == "http" {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		if len(u.Path) > 0 {
			opts = append(opts, otlpmetrichttp.WithURLPath(u.Path))
		}
	}
	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case compressionGzip:
			opts = append(opts, otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression))
		case compressionNone:
			opts = append(opts, otlpmetrichttp.WithCompression(otlpmetrichttp.NoCompression))
		default:
			return nil, fmt.Errorf("unsupported compression %q", *otlpConfig.Compression)
		}
	}
	if otlpConfig.Timeout != nil {
		opts = append(opts, otlpmetrichttp.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	headersConfig, err := createHeadersConfig(otlpConfig.Headers, otlpConfig.HeadersList)
	if err != nil {
		return nil, err
	}
	if len(headersConfig) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(headersConfig))
	}
	if otlpConfig.TemporalityPreference != nil {
		switch *otlpConfig.TemporalityPreference {
		case "delta":
			opts = append(opts, otlpmetrichttp.WithTemporalitySelector(deltaTemporality))
		case "cumulative":
			opts = append(opts, otlpmetrichttp.WithTemporalitySelector(cumulativeTemporality))
		case "lowmemory":
			opts = append(opts, otlpmetrichttp.WithTemporalitySelector(lowMemory))
		default:
			return nil, fmt.Errorf("unsupported temporality preference %q", *otlpConfig.TemporalityPreference)
		}
	}

	tlsConfig, err := createTLSConfig(otlpConfig.Certificate, otlpConfig.ClientCertificate, otlpConfig.ClientKey)
	if err != nil {
		return nil, err
	}
	opts = append(opts, otlpmetrichttp.WithTLSClientConfig(tlsConfig))

	return otlpmetrichttp.New(ctx, opts...)
}

func otlpGRPCMetricExporter(ctx context.Context, otlpConfig *OTLPMetric) (sdkmetric.Exporter, error) {
	var opts []otlpmetricgrpc.Option

	if otlpConfig.Endpoint != nil {
		u, err := url.ParseRequestURI(*otlpConfig.Endpoint)
		if err != nil {
			return nil, err
		}
		// ParseRequestURI leaves the Host field empty when no
		// scheme is specified (i.e. localhost:4317). This check is
		// here to support the case where a user may not specify a
		// scheme. The code does its best effort here by using
		// otlpConfig.Endpoint as-is in that case
		if u.Host != "" {
			opts = append(opts, otlpmetricgrpc.WithEndpoint(u.Host))
		} else {
			opts = append(opts, otlpmetricgrpc.WithEndpoint(*otlpConfig.Endpoint))
		}
		if u.Scheme == "http" || (u.Scheme != "https" && otlpConfig.Insecure != nil && *otlpConfig.Insecure) {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
	}

	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case compressionGzip:
			opts = append(opts, otlpmetricgrpc.WithCompressor(*otlpConfig.Compression))
		case compressionNone:
			// none requires no options
		default:
			return nil, fmt.Errorf("unsupported compression %q", *otlpConfig.Compression)
		}
	}
	if otlpConfig.Timeout != nil && *otlpConfig.Timeout > 0 {
		opts = append(opts, otlpmetricgrpc.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	headersConfig, err := createHeadersConfig(otlpConfig.Headers, otlpConfig.HeadersList)
	if err != nil {
		return nil, err
	}
	if len(headersConfig) > 0 {
		opts = append(opts, otlpmetricgrpc.WithHeaders(headersConfig))
	}
	if otlpConfig.TemporalityPreference != nil {
		switch *otlpConfig.TemporalityPreference {
		case "delta":
			opts = append(opts, otlpmetricgrpc.WithTemporalitySelector(deltaTemporality))
		case "cumulative":
			opts = append(opts, otlpmetricgrpc.WithTemporalitySelector(cumulativeTemporality))
		case "lowmemory":
			opts = append(opts, otlpmetricgrpc.WithTemporalitySelector(lowMemory))
		default:
			return nil, fmt.Errorf("unsupported temporality preference %q", *otlpConfig.TemporalityPreference)
		}
	}

	tlsConfig, err := createTLSConfig(otlpConfig.Certificate, otlpConfig.ClientCertificate, otlpConfig.ClientKey)
	if err != nil {
		return nil, err
	}
	opts = append(opts, otlpmetricgrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))

	return otlpmetricgrpc.New(ctx, opts...)
}

func cumulativeTemporality(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func deltaTemporality(ik sdkmetric.InstrumentKind) metricdata.Temporality {
	switch ik {
	case sdkmetric.InstrumentKindCounter, sdkmetric.InstrumentKindHistogram, sdkmetric.InstrumentKindObservableCounter:
		return metricdata.DeltaTemporality
	default:
		return metricdata.CumulativeTemporality
	}
}

func lowMemory(ik sdkmetric.InstrumentKind) metricdata.Temporality {
	switch ik {
	case sdkmetric.InstrumentKindCounter, sdkmetric.InstrumentKindHistogram:
		return metricdata.DeltaTemporality
	default:
		return metricdata.CumulativeTemporality
	}
}

// newIncludeExcludeFilter returns a Filter that includes attributes
// in the include list and excludes attributes in the excludes list.
// It returns an error if an attribute is in both lists
//
// If IncludeExclude is empty a include-all filter is returned.
func newIncludeExcludeFilter(lists *IncludeExclude) (attribute.Filter, error) {
	if lists == nil {
		return func(kv attribute.KeyValue) bool { return true }, nil
	}

	included := make(map[attribute.Key]struct{})
	for _, k := range lists.Included {
		included[attribute.Key(k)] = struct{}{}
	}
	excluded := make(map[attribute.Key]struct{})
	for _, k := range lists.Excluded {
		if _, ok := included[attribute.Key(k)]; ok {
			return nil, fmt.Errorf("attribute cannot be in both include and exclude list: %s", k)
		}
		excluded[attribute.Key(k)] = struct{}{}
	}
	return func(kv attribute.KeyValue) bool {
		// check if a value is excluded first
		if _, ok := excluded[kv.Key]; ok {
			return false
		}

		if len(included) == 0 {
			return true
		}

		_, ok := included[kv.Key]
		return ok
	}, nil
}

func prometheusReader(ctx context.Context, prometheusConfig *Prometheus) (sdkmetric.Reader, error) {
	if prometheusConfig.Host == nil {
		return nil, errors.New("host must be specified")
	}
	if prometheusConfig.Port == nil {
		return nil, errors.New("port must be specified")
	}

	opts, err := prometheusReaderOpts(prometheusConfig)
	if err != nil {
		return nil, err
	}

	reg := prometheus.NewRegistry()
	opts = append(opts, otelprom.WithRegisterer(reg))

	reader, err := otelprom.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating otel prometheus exporter: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	server := http.Server{
		// Timeouts are necessary to make a server resilient to attacks.
		// We use values from this example: https://blog.cloudflare.com/exposing-go-on-the-internet/#:~:text=There%20are%20three%20main%20timeouts
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      mux,
	}

	// Remove surrounding "[]" from the host definition to allow users to define the host as "[::1]" or "::1".
	host := *prometheusConfig.Host
	if len(host) > 2 && host[0] == '[' && host[len(host)-1] == ']' {
		host = host[1 : len(host)-1]
	}

	addr := net.JoinHostPort(host, strconv.Itoa(*prometheusConfig.Port))
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("binding address %s for Prometheus exporter: %w", addr, err),
			reader.Shutdown(ctx),
		)
	}

	// Only for testing reasons, add the address to the http Server, will not be used.
	server.Addr = lis.Addr().String()

	go func() {
		if err := server.Serve(lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
			otel.Handle(fmt.Errorf("the Prometheus HTTP server exited unexpectedly: %w", err))
		}
	}()

	return readerWithServer{reader, &server}, nil
}

func prometheusReaderOpts(prometheusConfig *Prometheus) ([]otelprom.Option, error) {
	var opts []otelprom.Option
	if prometheusConfig.WithoutScopeInfo != nil && *prometheusConfig.WithoutScopeInfo {
		opts = append(opts, otelprom.WithoutScopeInfo())
	}
	if prometheusConfig.WithoutTypeSuffix != nil && *prometheusConfig.WithoutTypeSuffix {
		opts = append(opts, otelprom.WithoutCounterSuffixes())
	}
	if prometheusConfig.WithoutUnits != nil && *prometheusConfig.WithoutUnits {
		opts = append(opts, otelprom.WithoutUnits())
	}
	if prometheusConfig.WithResourceConstantLabels != nil {
		f, err := newIncludeExcludeFilter(prometheusConfig.WithResourceConstantLabels)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otelprom.WithResourceAsConstantLabels(f))
	}

	return opts, nil
}

type readerWithServer struct {
	sdkmetric.Reader
	server *http.Server
}

func (rws readerWithServer) Shutdown(ctx context.Context) error {
	return errors.Join(
		rws.Reader.Shutdown(ctx),
		rws.server.Shutdown(ctx),
	)
}

func view(v View) (sdkmetric.View, error) {
	if v.Selector == nil {
		return nil, errors.New("view: no selector provided")
	}

	inst, err := instrument(*v.Selector)
	if err != nil {
		return nil, err
	}

	s, err := stream(v.Stream)
	if err != nil {
		return nil, err
	}
	return sdkmetric.NewView(inst, s), nil
}

func instrument(vs ViewSelector) (sdkmetric.Instrument, error) {
	kind, err := instrumentKind(vs.InstrumentType)
	if err != nil {
		return sdkmetric.Instrument{}, fmt.Errorf("view_selector: %w", err)
	}
	inst := sdkmetric.Instrument{
		Name: strOrEmpty(vs.InstrumentName),
		Unit: strOrEmpty(vs.Unit),
		Kind: kind,
		Scope: instrumentation.Scope{
			Name:      strOrEmpty(vs.MeterName),
			Version:   strOrEmpty(vs.MeterVersion),
			SchemaURL: strOrEmpty(vs.MeterSchemaUrl),
		},
	}

	if instrumentIsEmpty(inst) {
		return sdkmetric.Instrument{}, errors.New("view_selector: empty selector not supporter")
	}
	return inst, nil
}

func stream(vs *ViewStream) (sdkmetric.Stream, error) {
	if vs == nil {
		return sdkmetric.Stream{}, nil
	}

	f, err := newIncludeExcludeFilter(vs.AttributeKeys)
	if err != nil {
		return sdkmetric.Stream{}, err
	}
	return sdkmetric.Stream{
		Name:            strOrEmpty(vs.Name),
		Description:     strOrEmpty(vs.Description),
		Aggregation:     aggregation(vs.Aggregation),
		AttributeFilter: f,
	}, nil
}

func aggregation(aggr *ViewStreamAggregation) sdkmetric.Aggregation {
	if aggr == nil {
		return nil
	}

	if aggr.Base2ExponentialBucketHistogram != nil {
		return sdkmetric.AggregationBase2ExponentialHistogram{
			MaxSize:  int32OrZero(aggr.Base2ExponentialBucketHistogram.MaxSize),
			MaxScale: int32OrZero(aggr.Base2ExponentialBucketHistogram.MaxScale),
			// Need to negate because config has the positive action RecordMinMax.
			NoMinMax: !boolOrFalse(aggr.Base2ExponentialBucketHistogram.RecordMinMax),
		}
	}
	if aggr.Default != nil {
		// TODO: Understand what to set here.
		return nil
	}
	if aggr.Drop != nil {
		return sdkmetric.AggregationDrop{}
	}
	if aggr.ExplicitBucketHistogram != nil {
		return sdkmetric.AggregationExplicitBucketHistogram{
			Boundaries: aggr.ExplicitBucketHistogram.Boundaries,
			// Need to negate because config has the positive action RecordMinMax.
			NoMinMax: !boolOrFalse(aggr.ExplicitBucketHistogram.RecordMinMax),
		}
	}
	if aggr.LastValue != nil {
		return sdkmetric.AggregationLastValue{}
	}
	if aggr.Sum != nil {
		return sdkmetric.AggregationSum{}
	}
	return nil
}

func instrumentKind(vsit *ViewSelectorInstrumentType) (sdkmetric.InstrumentKind, error) {
	if vsit == nil {
		// Equivalent to instrumentKindUndefined.
		return instrumentKindUndefined, nil
	}

	switch *vsit {
	case ViewSelectorInstrumentTypeCounter:
		return sdkmetric.InstrumentKindCounter, nil
	case ViewSelectorInstrumentTypeUpDownCounter:
		return sdkmetric.InstrumentKindUpDownCounter, nil
	case ViewSelectorInstrumentTypeHistogram:
		return sdkmetric.InstrumentKindHistogram, nil
	case ViewSelectorInstrumentTypeObservableCounter:
		return sdkmetric.InstrumentKindObservableCounter, nil
	case ViewSelectorInstrumentTypeObservableUpDownCounter:
		return sdkmetric.InstrumentKindObservableUpDownCounter, nil
	case ViewSelectorInstrumentTypeObservableGauge:
		return sdkmetric.InstrumentKindObservableGauge, nil
	}

	return instrumentKindUndefined, errors.New("instrument_type: invalid value")
}

func instrumentIsEmpty(i sdkmetric.Instrument) bool {
	return i.Name == "" &&
		i.Description == "" &&
		i.Kind == instrumentKindUndefined &&
		i.Unit == "" &&
		i.Scope == zeroScope
}

func boolOrFalse(pBool *bool) bool {
	if pBool == nil {
		return false
	}
	return *pBool
}

func int32OrZero(pInt *int) int32 {
	if pInt == nil {
		return 0
	}
	i := *pInt
	if i > math.MaxInt32 {
		return math.MaxInt32
	}
	if i < math.MinInt32 {
		return math.MinInt32
	}
	return int32(i) // nolint: gosec  // Overflow and underflow checked above.
}

func strOrEmpty(pStr *string) string {
	if pStr == nil {
		return ""
	}
	return *pStr
}
