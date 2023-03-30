// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datadogexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter"

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/trace/api"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/attributes/source"
	otlpmetrics "github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/metrics"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	zorkian "gopkg.in/zorkian/go-datadog-api.v2"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/clientutil"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/metrics"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/metrics/sketches"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/scrub"
)

type metricsExporter struct {
	params         exporter.CreateSettings
	cfg            *Config
	ctx            context.Context
	client         *zorkian.Client
	metricsAPI     *datadogV2.MetricsApi
	tr             *otlpmetrics.Translator
	scrubber       scrub.Scrubber
	retrier        *clientutil.Retrier
	onceMetadata   *sync.Once
	sourceProvider source.Provider
	// getPushTime returns a Unix time in nanoseconds, representing the time pushing metrics.
	// It will be overwritten in tests.
	getPushTime       func() uint64
	apmStatsProcessor api.StatsProcessor
}

// translatorFromConfig creates a new metrics translator from the exporter
func translatorFromConfig(logger *zap.Logger, cfg *Config, sourceProvider source.Provider) (*otlpmetrics.Translator, error) {
	options := []otlpmetrics.TranslatorOption{
		otlpmetrics.WithDeltaTTL(cfg.Metrics.DeltaTTL),
		otlpmetrics.WithFallbackSourceProvider(sourceProvider),
	}

	if cfg.Metrics.HistConfig.SendCountSum {
		options = append(options, otlpmetrics.WithCountSumMetrics())
	}

	if cfg.Metrics.SummaryConfig.Mode == SummaryModeGauges {
		options = append(options, otlpmetrics.WithQuantiles())
	}

	if cfg.Metrics.ExporterConfig.ResourceAttributesAsTags {
		options = append(options, otlpmetrics.WithResourceAttributesAsTags())
	}

	if cfg.Metrics.ExporterConfig.InstrumentationScopeMetadataAsTags {
		options = append(options, otlpmetrics.WithInstrumentationScopeMetadataAsTags())
	}

	options = append(options, otlpmetrics.WithHistogramMode(otlpmetrics.HistogramMode(cfg.Metrics.HistConfig.Mode)))

	var numberMode otlpmetrics.NumberMode
	switch cfg.Metrics.SumConfig.CumulativeMonotonicMode {
	case CumulativeMonotonicSumModeRawValue:
		numberMode = otlpmetrics.NumberModeRawValue
	case CumulativeMonotonicSumModeToDelta:
		numberMode = otlpmetrics.NumberModeCumulativeToDelta
	}

	options = append(options, otlpmetrics.WithNumberMode(numberMode))

	if hostmetadata.HostnamePreviewFeatureGate.IsEnabled() {
		options = append(options, otlpmetrics.WithPreviewHostnameFromAttributes())
	}

	return otlpmetrics.NewTranslator(logger, options...)
}

func newMetricsExporter(ctx context.Context, params exporter.CreateSettings, cfg *Config, onceMetadata *sync.Once, sourceProvider source.Provider, apmStatsProcessor api.StatsProcessor) (*metricsExporter, error) {
	tr, err := translatorFromConfig(params.Logger, cfg, sourceProvider)
	if err != nil {
		return nil, err
	}

	scrubber := scrub.NewScrubber()
	exporter := &metricsExporter{
		params:            params,
		cfg:               cfg,
		ctx:               ctx,
		tr:                tr,
		scrubber:          scrubber,
		retrier:           clientutil.NewRetrier(params.Logger, cfg.RetrySettings, scrubber),
		onceMetadata:      onceMetadata,
		sourceProvider:    sourceProvider,
		getPushTime:       func() uint64 { return uint64(time.Now().UTC().UnixNano()) },
		apmStatsProcessor: apmStatsProcessor,
	}
	errchan := make(chan error)
	if isMetricExportV2Enabled() {
		apiClient := clientutil.CreateAPIClient(
			params.BuildInfo,
			cfg.Metrics.TCPAddr.Endpoint,
			cfg.TimeoutSettings,
			cfg.LimitedHTTPClientSettings.TLSSetting.InsecureSkipVerify)
		go func() { errchan <- clientutil.ValidateAPIKey(ctx, string(cfg.API.Key), params.Logger, apiClient) }()
		exporter.metricsAPI = datadogV2.NewMetricsApi(apiClient)
	} else {
		client := clientutil.CreateZorkianClient(string(cfg.API.Key), cfg.Metrics.TCPAddr.Endpoint)
		client.ExtraHeader["User-Agent"] = clientutil.UserAgent(params.BuildInfo)
		client.HttpClient = clientutil.NewHTTPClient(cfg.TimeoutSettings, cfg.LimitedHTTPClientSettings.TLSSetting.InsecureSkipVerify)
		go func() { errchan <- clientutil.ValidateAPIKeyZorkian(params.Logger, client) }()
		exporter.client = client
	}
	if cfg.API.FailOnInvalidKey {
		err = <-errchan
		if err != nil {
			return nil, err
		}
	}
	return exporter, nil
}

func (exp *metricsExporter) pushSketches(ctx context.Context, sl sketches.SketchSeriesList) error {
	payload, err := sl.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal sketches: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost,
		exp.cfg.Metrics.TCPAddr.Endpoint+sketches.SketchSeriesEndpoint,
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return fmt.Errorf("failed to build sketches HTTP request: %w", err)
	}

	clientutil.SetDDHeaders(req.Header, exp.params.BuildInfo, string(exp.cfg.API.Key))
	clientutil.SetExtraHeaders(req.Header, clientutil.ProtobufHeaders)
	var resp *http.Response
	if isMetricExportV2Enabled() {
		resp, err = exp.metricsAPI.Client.Cfg.HTTPClient.Do(req)
	} else {
		resp, err = exp.client.HttpClient.Do(req)
	}

	if err != nil {
		return clientutil.WrapError(fmt.Errorf("failed to do sketches HTTP request: %w", err), resp)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return clientutil.WrapError(fmt.Errorf("error when sending payload to %s: %s", sketches.SketchSeriesEndpoint, resp.Status), resp)
	}
	return nil
}

func (exp *metricsExporter) PushMetricsDataScrubbed(ctx context.Context, md pmetric.Metrics) error {
	return exp.scrubber.Scrub(exp.PushMetricsData(ctx, md))
}

func (exp *metricsExporter) PushMetricsData(ctx context.Context, md pmetric.Metrics) error {
	// Start host metadata with resource attributes from
	// the first payload.
	if exp.cfg.HostMetadata.Enabled {
		exp.onceMetadata.Do(func() {
			attrs := pcommon.NewMap()
			if md.ResourceMetrics().Len() > 0 {
				attrs = md.ResourceMetrics().At(0).Resource().Attributes()
			}
			go hostmetadata.Pusher(exp.ctx, exp.params, newMetadataConfigfromConfig(exp.cfg), exp.sourceProvider, attrs)
		})
	}
	var consumer otlpmetrics.Consumer
	if isMetricExportV2Enabled() {
		consumer = metrics.NewConsumer()
	} else {
		consumer = metrics.NewZorkianConsumer()
	}
	err := exp.tr.MapMetrics(ctx, md, consumer)
	if err != nil {
		return fmt.Errorf("failed to map metrics: %w", err)
	}
	src, err := exp.sourceProvider.Source(ctx)
	if err != nil {
		return err
	}
	var tags []string
	if src.Kind == source.AWSECSFargateKind {
		tags = append(tags, exp.cfg.HostMetadata.Tags...)
	}

	var sl sketches.SketchSeriesList
	var sp []pb.ClientStatsPayload
	if isMetricExportV2Enabled() {
		var ms []datadogV2.MetricSeries
		ms, sl, sp = consumer.(*metrics.Consumer).All(exp.getPushTime(), exp.params.BuildInfo, tags)
		ms = metrics.PrepareSystemMetrics(ms)

		err = nil
		if len(ms) > 0 {
			exp.params.Logger.Debug("exporting native Datadog payload", zap.Any("metric", ms))
			_, experr := exp.retrier.DoWithRetries(ctx, func(context.Context) error {
				ctx = clientutil.GetRequestContext(ctx, string(exp.cfg.API.Key))
				_, httpresp, merr := exp.metricsAPI.SubmitMetrics(ctx, datadogV2.MetricPayload{Series: ms}, *clientutil.GZipSubmitMetricsOptionalParameters)
				return clientutil.WrapError(merr, httpresp)
			})
			err = multierr.Append(err, experr)
		}
	} else {
		var ms []zorkian.Metric
		ms, sl, sp = consumer.(*metrics.ZorkianConsumer).All(exp.getPushTime(), exp.params.BuildInfo, tags)
		ms = metrics.PrepareZorkianSystemMetrics(ms)

		err = nil
		if len(ms) > 0 {
			exp.params.Logger.Debug("exporting Zorkian Datadog payload", zap.Any("metric", ms))
			_, experr := exp.retrier.DoWithRetries(ctx, func(context.Context) error {
				return exp.client.PostMetrics(ms)
			})
			err = multierr.Append(err, experr)
		}
	}

	if len(sl) > 0 {
		exp.params.Logger.Debug("exporting sketches payload", zap.Any("sketches", sl))
		_, experr := exp.retrier.DoWithRetries(ctx, func(ctx context.Context) error {
			return exp.pushSketches(ctx, sl)
		})
		err = multierr.Append(err, experr)
	}

	if len(sp) > 0 {
		exp.params.Logger.Debug("exporting APM stats payloads", zap.Any("stats_payloads", sp))
		statsv := exp.params.BuildInfo.Command + exp.params.BuildInfo.Version
		for _, p := range sp {
			exp.apmStatsProcessor.ProcessStats(p, "", statsv)
		}
	}

	return err
}
