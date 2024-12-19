package blockbuilder

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
)

type partitionSectionWriter interface {
	pushBytes(tenant string, req *tempopb.PushBytesRequest) error
	flush(ctx context.Context, store tempodb.Writer) error
}

type writer struct {
	logger log.Logger

	blockCfg                         BlockConfig
	partition                        int64
	startSectionTime, endSectionTime time.Time

	overrides Overrides
	wal       *wal.WAL
	enc       encoding.VersionedEncoding

	mtx sync.Mutex
	m   map[string]*tenantStore
}

func newPartitionSectionWriter(logger log.Logger, partition int64, endSectionTime, startSectionTime time.Time, blockCfg BlockConfig, overrides Overrides, wal *wal.WAL, enc encoding.VersionedEncoding) *writer {
	return &writer{
		logger:           logger,
		partition:        partition,
		endSectionTime:   endSectionTime,
		startSectionTime: startSectionTime,
		blockCfg:         blockCfg,
		overrides:        overrides,
		wal:              wal,
		enc:              enc,
		mtx:              sync.Mutex{},
		m:                make(map[string]*tenantStore),
	}
}

func (p *writer) pushBytes(tenant string, req *tempopb.PushBytesRequest) error {
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
		tr := new(tempopb.Trace) // TODO - Pool?
		if err := proto.Unmarshal(trace.Slice, tr); err != nil {
			return fmt.Errorf("failed to unmarshal trace: %w", err)
		}
		if err := i.AppendTrace(req.Ids[j], tr, p.startSectionTime, p.endSectionTime); err != nil {
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

	i, err := newTenantStore(tenant, p.partition, p.endSectionTime.UnixMilli(), p.blockCfg, p.logger, p.wal, p.enc, p.overrides)
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
