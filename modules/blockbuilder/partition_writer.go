package blockbuilder

import (
	"context"
	"fmt"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
)

type partitionWriter interface {
	PushBytes(tenant string, req *tempopb.PushBytesRequest) error
	Flush(ctx context.Context, store tempodb.Writer) error
}

type writer struct {
	logger log.Logger

	blockCfg BlockConfig

	overrides Overrides
	wal       *wal.WAL
	enc       encoding.VersionedEncoding

	// TODO - Lock
	m map[string]*tenantStore
}

func newPartitionProcessor(logger log.Logger, blockCfg BlockConfig, overrides Overrides, wal *wal.WAL, enc encoding.VersionedEncoding) *writer {
	return &writer{
		logger:    logger,
		blockCfg:  blockCfg,
		overrides: overrides,
		wal:       wal,
		enc:       enc,
		m:         make(map[string]*tenantStore),
	}
}

func (p *writer) PushBytes(tenant string, req *tempopb.PushBytesRequest) error {
	level.Info(p.logger).Log("msg", "pushing bytes", "tenant", tenant, "num_traces", len(req.Traces))

	i, err := p.instanceForTenant(tenant)
	if err != nil {
		return err
	}

	for j, trace := range req.Traces {
		tr := new(tempopb.Trace) // TODO - Pool?
		if err := proto.Unmarshal(trace.Slice, tr); err != nil {
			return fmt.Errorf("failed to unmarshal trace: %w", err)
		}

		if err := i.AppendTrace(req.Ids[j].Slice, tr, 0, 0); err != nil {
			return err
		}
	}

	return nil
}

func (p *writer) Flush(ctx context.Context, store tempodb.Writer) error {
	// TODO - Retry with backoff?
	for _, i := range p.m {
		level.Info(p.logger).Log("msg", "flushing tenant", "tenant", i.tenantID)
		if err := i.Flush(ctx, store); err != nil {
			return err
		}
	}
	return nil
}

func (p *writer) instanceForTenant(tenant string) (*tenantStore, error) {
	if i, ok := p.m[tenant]; ok {
		return i, nil
	}

	i, err := newTenantStore(tenant, p.blockCfg, p.logger, p.wal, p.enc, p.overrides)
	if err != nil {
		return nil, err
	}

	p.m[tenant] = i

	return i, nil
}
