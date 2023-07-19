package poller

import (
	"context"

	"github.com/grafana/dskit/services"

	"github.com/grafana/tempo/tempodb"
)

type Poller struct {
	services.Service

	cfg Config

	tempodb.Poller
}

// NewStore creates a new Tempo Store using configuration supplied.
func New(cfg Config, poller tempodb.Poller) (*Poller, error) {
	p := &Poller{
		cfg:    cfg,
		Poller: poller,
	}

	p.Service = services.NewIdleService(p.starting, p.stopping)
	return p, nil
}

func (p *Poller) starting(_ context.Context) error {
	return nil
}

func (p *Poller) stopping(_ error) error {
	return nil
}
