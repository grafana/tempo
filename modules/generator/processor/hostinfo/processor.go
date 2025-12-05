package hostinfo

import (
	"context"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

const (
	hostInfoMetric     = "traces_host_info"
	hostIdentifierAttr = "grafana_host_id"
	hostSourceAttr     = "host_source"
)

type Processor struct {
	Cfg    Config
	logger log.Logger

	gauge              registry.Gauge
	registry           registry.Registry
	metricName         string
	invalidUTF8Counter prometheus.Counter
}

func (p *Processor) Name() string {
	return processor.HostInfoName
}

func (p *Processor) findHostIdentifier(resourceSpans *v1.ResourceSpans) (string, string) {
	attrs := resourceSpans.GetResource().GetAttributes()
	for _, idAttr := range p.Cfg.HostIdentifiers {
		for _, attr := range attrs {
			hostSource := attr.GetKey()
			if hostSource == idAttr {
				if val := attr.GetValue(); val != nil {
					if strVal := val.GetStringValue(); strVal != "" {
						return strVal, hostSource
					}
				}
			}
		}
	}
	return "", ""
}

func (p *Processor) PushSpans(_ context.Context, req *tempopb.PushSpansRequest) {
	for i := range req.Batches {
		resourceSpans := req.Batches[i]
		if hostID, hostSource := p.findHostIdentifier(resourceSpans); hostID != "" && hostSource != "" {
			builder := p.registry.NewLabelBuilder()
			builder.Add(hostIdentifierAttr, hostID)
			builder.Add(hostSourceAttr, hostSource)
			labels, validUTF8 := builder.CloseAndBuildLabels()
			if !validUTF8 {
				p.invalidUTF8Counter.Inc()
				continue
			}
			p.gauge.Set(labels, 1)
		}
	}
}

func (p *Processor) Shutdown(_ context.Context) {}

func New(cfg Config, reg registry.Registry, logger log.Logger, invalidUTF8Counter prometheus.Counter) (*Processor, error) {
	p := &Processor{
		Cfg:                cfg,
		logger:             logger,
		registry:           reg,
		metricName:         cfg.MetricName,
		gauge:              reg.NewGauge(cfg.MetricName),
		invalidUTF8Counter: invalidUTF8Counter,
	}
	return p, nil
}
