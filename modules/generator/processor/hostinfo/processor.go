package hostinfo

import (
	"context"
	"time"

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
	Cfg      Config
	logger   log.Logger
	registry registry.Registry
	done     chan struct{}

	// TODO list of mMaps?
	mMap *metricMap
}

func (p *Processor) Name() string {
	return Name
}

func (p *Processor) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) {
	_, span := tracer.Start(ctx, "hostinfo.PushSpans")
	defer span.End()

	for i := 0; i < len(req.Batches); i++ {
		resourceSpan := req.Batches[i]
		attrs := resourceSpan.GetResource().GetAttributes()

	idAttrLoop:
		for _, idAttr := range p.Cfg.HostIdentifiers {
			for _, attr := range attrs {
				if attr.GetKey() == idAttr {
					p.mMap.add(attr.GetValue().GetStringValue())
					break idAttrLoop
				}
			}
		}
	}
}

func (p *Processor) flush() {
	p.mMap.register(p.registry)
	p.mMap.reset()
}

func (p *Processor) Shutdown(_ context.Context) {
	p.done <- struct{}{}
}

func New(cfg Config, reg registry.Registry, logger log.Logger) (*Processor, error) {
	p := &Processor{
		Cfg:      cfg,
		done:     make(chan struct{}),
		logger:   log.With(logger, "component", Name),
		registry: reg,
		mMap:     newMetricMap(hostInfoMetric),
	}

	ticker := time.NewTicker(cfg.MetricsFlushInterval)
	go func() {
		for {
			select {
			case <-p.done:
				ticker.Stop()
				p.flush()
				return
			case <-ticker.C:
				p.flush()
			}
		}
	}()
	return p, nil
}
