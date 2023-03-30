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
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/trace/agent"
	traceconfig "github.com/DataDog/datadog-agent/pkg/trace/config"
	tracelog "github.com/DataDog/datadog-agent/pkg/trace/log"
	"github.com/DataDog/datadog-agent/pkg/trace/telemetry"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/attributes/source"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
	zorkian "gopkg.in/zorkian/go-datadog-api.v2"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/clientutil"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/metrics"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/scrub"
)

type traceExporter struct {
	params         exporter.CreateSettings
	cfg            *Config
	ctx            context.Context       // ctx triggers shutdown upon cancellation
	client         *zorkian.Client       // client sends runnimg metrics to backend & performs API validation
	metricsAPI     *datadogV2.MetricsApi // client sends runnimg metrics to backend
	scrubber       scrub.Scrubber        // scrubber scrubs sensitive information from error messages
	onceMetadata   *sync.Once            // onceMetadata ensures that metadata is sent only once across all exporters
	agent          *agent.Agent          // agent processes incoming traces
	sourceProvider source.Provider       // is able to source the origin of a trace (hostname, container, etc)
	retrier        *clientutil.Retrier   // retrier handles retries on requests
}

func newTracesExporter(ctx context.Context, params exporter.CreateSettings, cfg *Config, onceMetadata *sync.Once, sourceProvider source.Provider, agent *agent.Agent) (*traceExporter, error) {
	scrubber := scrub.NewScrubber()
	exp := &traceExporter{
		params:         params,
		cfg:            cfg,
		ctx:            ctx,
		agent:          agent,
		onceMetadata:   onceMetadata,
		scrubber:       scrubber,
		sourceProvider: sourceProvider,
		retrier:        clientutil.NewRetrier(params.Logger, cfg.RetrySettings, scrubber),
	}
	// client to send running metric to the backend & perform API key validation
	errchan := make(chan error)
	if isMetricExportV2Enabled() {
		apiClient := clientutil.CreateAPIClient(
			params.BuildInfo,
			cfg.Metrics.TCPAddr.Endpoint,
			cfg.TimeoutSettings,
			cfg.LimitedHTTPClientSettings.TLSSetting.InsecureSkipVerify)
		go func() { errchan <- clientutil.ValidateAPIKey(ctx, string(cfg.API.Key), params.Logger, apiClient) }()
		exp.metricsAPI = datadogV2.NewMetricsApi(apiClient)
	} else {
		client := clientutil.CreateZorkianClient(string(cfg.API.Key), cfg.Metrics.TCPAddr.Endpoint)
		go func() { errchan <- clientutil.ValidateAPIKeyZorkian(params.Logger, client) }()
		exp.client = client
	}
	if cfg.API.FailOnInvalidKey {
		if err := <-errchan; err != nil {
			return nil, err
		}
	}
	return exp, nil
}

var _ consumer.ConsumeTracesFunc = (*traceExporter)(nil).consumeTraces

func (exp *traceExporter) consumeTraces(
	ctx context.Context,
	td ptrace.Traces,
) (err error) {
	defer func() { err = exp.scrubber.Scrub(err) }()
	if exp.cfg.HostMetadata.Enabled {
		// start host metadata with resource attributes from
		// the first payload.
		exp.onceMetadata.Do(func() {
			attrs := pcommon.NewMap()
			if td.ResourceSpans().Len() > 0 {
				attrs = td.ResourceSpans().At(0).Resource().Attributes()
			}
			go hostmetadata.Pusher(exp.ctx, exp.params, newMetadataConfigfromConfig(exp.cfg), exp.sourceProvider, attrs)
		})
	}
	rspans := td.ResourceSpans()
	hosts := make(map[string]struct{})
	tags := make(map[string]struct{})
	for i := 0; i < rspans.Len(); i++ {
		rspan := rspans.At(i)
		src := exp.agent.OTLPReceiver.ReceiveResourceSpans(ctx, rspan, http.Header{})
		switch src.Kind {
		case source.HostnameKind:
			hosts[src.Identifier] = struct{}{}
		case source.AWSECSFargateKind:
			tags[src.Tag()] = struct{}{}
		}
	}

	exp.exportUsageMetrics(ctx, hosts, tags)
	return nil
}

func (exp *traceExporter) exportUsageMetrics(ctx context.Context, hosts map[string]struct{}, tags map[string]struct{}) {
	now := pcommon.NewTimestampFromTime(time.Now())
	var err error
	if isMetricExportV2Enabled() {
		series := make([]datadogV2.MetricSeries, 0, len(hosts)+len(tags))
		for host := range hosts {
			series = append(series, metrics.DefaultMetrics("traces", host, uint64(now), exp.params.BuildInfo)...)
		}
		for tag := range tags {
			ms := metrics.DefaultMetrics("traces", "", uint64(now), exp.params.BuildInfo)
			for i := range ms {
				ms[i].Tags = append(ms[i].Tags, tag)
			}
			series = append(series, ms...)
		}
		_, err = exp.retrier.DoWithRetries(ctx, func(context.Context) error {
			ctx2 := clientutil.GetRequestContext(ctx, string(exp.cfg.API.Key))
			_, httpresp, merr := exp.metricsAPI.SubmitMetrics(ctx2, datadogV2.MetricPayload{Series: series}, *clientutil.GZipSubmitMetricsOptionalParameters)
			return clientutil.WrapError(merr, httpresp)
		})
	} else {
		series := make([]zorkian.Metric, 0, len(hosts)+len(tags))
		for host := range hosts {
			series = append(series, metrics.DefaultZorkianMetrics("traces", host, uint64(now), exp.params.BuildInfo)...)
		}
		for tag := range tags {
			ms := metrics.DefaultZorkianMetrics("traces", "", uint64(now), exp.params.BuildInfo)
			for i := range ms {
				ms[i].Tags = append(ms[i].Tags, tag)
			}
			series = append(series, ms...)
		}
		_, err = exp.retrier.DoWithRetries(ctx, func(context.Context) error {
			return exp.client.PostMetrics(series)
		})
	}
	if err != nil {
		exp.params.Logger.Error("Error posting hostname/tags series", zap.Error(err))
	}
}

func newTraceAgent(ctx context.Context, params exporter.CreateSettings, cfg *Config, sourceProvider source.Provider) (*agent.Agent, error) {
	acfg := traceconfig.New()
	src, err := sourceProvider.Source(ctx)
	if err != nil {
		return nil, err
	}
	if src.Kind == source.HostnameKind {
		acfg.Hostname = src.Identifier
	}
	acfg.OTLPReceiver.SpanNameRemappings = cfg.Traces.SpanNameRemappings
	acfg.OTLPReceiver.SpanNameAsResourceName = cfg.Traces.SpanNameAsResourceName
	acfg.OTLPReceiver.UsePreviewHostnameLogic = hostmetadata.HostnamePreviewFeatureGate.IsEnabled()
	acfg.Endpoints[0].APIKey = string(cfg.API.Key)
	acfg.Ignore["resource"] = cfg.Traces.IgnoreResources
	acfg.ReceiverPort = 0 // disable HTTP receiver
	acfg.AgentVersion = fmt.Sprintf("datadogexporter-%s-%s", params.BuildInfo.Command, params.BuildInfo.Version)
	if v := cfg.Traces.flushInterval; v > 0 {
		acfg.TraceWriter.FlushPeriodSeconds = v
	}
	if addr := cfg.Traces.Endpoint; addr != "" {
		acfg.Endpoints[0].Host = addr
	}
	tracelog.SetLogger(&zaplogger{params.Logger})
	return agent.NewAgent(ctx, acfg, telemetry.NewNoopCollector()), nil
}
