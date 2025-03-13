package hostinfo

import (
	"context"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/tempopb"
	"go.opentelemetry.io/otel"
)

const (
	Name = "host-info"

	hostInfoMetric     = "traces_host_info"
	hostIdentifierAttr = "grafana.host.id"
)

var tracer = otel.Tracer("modules/generator/processor/hostinfo")

type Processor struct {
	Cfg    Config
	logger log.Logger

	gauge      registry.Gauge
	registry   registry.Registry
	metricName string
	labels     []string
}

func (p *Processor) Name() string {
	return Name
}

func (p *Processor) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) {
	_, span := tracer.Start(ctx, "hostinfo.PushSpans")
	defer span.End()

	values := make([]string, 1, 1)

outer:
	for i := range req.Batches {
		resourceSpan := req.Batches[i]
		attrs := resourceSpan.GetResource().GetAttributes()

		for _, idAttr := range p.Cfg.HostIdentifiers {
			for _, attr := range attrs {
				if attr.GetKey() == idAttr {
					if val := attr.GetValue(); val != nil {
						if strVal := val.GetStringValue(); strVal != "" {
							values[0] = strVal
							labelValues := p.registry.NewLabelValueCombo(
								p.labels,
								values,
							)
							p.gauge.Set(labelValues, 1)
							continue outer
						}
					}
				}
			}
		}
	}
}

func (p *Processor) Shutdown(_ context.Context) {}

func New(cfg Config, reg registry.Registry, logger log.Logger) (*Processor, error) {
	labels := make([]string, 1, 1)
	labels[0] = hostIdentifierAttr
	myGauge := reg.NewGauge(cfg.MetricName)
	myGauge.SetExpiration(cfg.StaleDuration)
	p := &Processor{
		Cfg:        cfg,
		logger:     logger,
		registry:   reg,
		metricName: cfg.MetricName,
		gauge:      reg.NewGauge(cfg.MetricName),
		labels:     labels,
	}
	return p, nil
}
