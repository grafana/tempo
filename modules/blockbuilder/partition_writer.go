package blockbuilder

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
)

type partitionSectionWriter interface {
	pushBytes(ts time.Time, tenant string, req *tempopb.PushBytesRequest) error
	flush(ctx context.Context, store tempodb.Writer) error
}

type writer struct {
	logger log.Logger

	blockCfg              BlockConfig
	partition, cycleEndTs uint64

	overrides Overrides
	wal       *wal.WAL
	enc       encoding.VersionedEncoding

	mtx sync.Mutex
	m   map[string]*tenantStore
}

func newPartitionSectionWriter(logger log.Logger, partition, cycleEndTs uint64, blockCfg BlockConfig, overrides Overrides, wal *wal.WAL, enc encoding.VersionedEncoding) *writer {
	return &writer{
		logger:     logger,
		partition:  partition,
		cycleEndTs: cycleEndTs,
		blockCfg:   blockCfg,
		overrides:  overrides,
		wal:        wal,
		enc:        enc,
		mtx:        sync.Mutex{},
		m:          make(map[string]*tenantStore),
	}
}

func (p *writer) pushBytes(ts time.Time, tenant string, req *tempopb.PushBytesRequest) error {
	level.Debug(p.logger).Log(
		"msg", "pushing bytes",
		"tenant", tenant,
		"num_traces", len(req.Traces),
		"id", idsToString(req.Ids),
	)

	i, err := p.instanceForTenant(tenant)
	if err != nil {
		return err
	}

	for j, trace := range req.Traces {
		if err := i.AppendTrace(req.Ids[j], trace.Slice, ts); err != nil {
			return err
		}
	}

	return nil
}

func (p *writer) cutidle(since time.Time, immediate bool) error {
	for _, i := range p.m {
		if err := i.CutIdle(since, immediate); err != nil {
			return err
		}
	}
	return nil
}

func (p *writer) flush(ctx context.Context, store tempodb.Writer) error {
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
	p.mtx.Lock()
	defer p.mtx.Unlock()

	if i, ok := p.m[tenant]; ok {
		return i, nil
	}

	i, err := newTenantStore(tenant, p.partition, p.cycleEndTs, p.blockCfg, p.logger, p.wal, p.enc, p.overrides)
	if err != nil {
		return nil, err
	}

	p.m[tenant] = i

	return i, nil
}

func idsToString(ids [][]byte) string {
	b := strings.Builder{}
	b.WriteString("[")
	for i, id := range ids {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(util.TraceIDToHexString(id))
	}
	b.WriteString("]")

	return b.String()
}
