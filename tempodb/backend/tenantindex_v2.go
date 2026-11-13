package backend

import (
	"context"
	"fmt"
	"time"

	tempo_io "github.com/grafana/tempo/v3/pkg/io"
)

const (
	// TenantIndexNameV2 is the V2 tenant index: shared block meta fields are stored once
	// per schema instead of per block.
	TenantIndexNameV2 = "index.v2.zst"

	// EventProtoFallback is the span event added when the V2 index is missing.
	EventProtoFallback = "falling back to v1 index"
)

// Option configures how a Reader or Writer behaves.
type Option func(*options)

type options struct {
	tenantIndexV2 bool
}

// WithTenantIndexV2 makes a reader prefer the V2 tenant index, falling back to the proto
// index when it does not exist. Writers always write the V2 index, so this can be enabled
// at any time without a stale or missing file. It has no effect on a writer.
func WithTenantIndexV2(enabled bool) Option {
	return func(o *options) { o.tenantIndexV2 = enabled }
}

func newOptions(opts []Option) options {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func (r *reader) tenantIndexV2(ctx context.Context, tenantID string) (*TenantIndex, error) {
	rdr, size, err := r.r.Read(ctx, TenantIndexNameV2, KeyPath([]string{tenantID}), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read tenant index v2: %w", err)
	}
	defer rdr.Close()

	buf, err := tempo_io.ReadAllWithEstimate(rdr, size)
	if err != nil {
		return nil, fmt.Errorf("failed to read all with estimate: %w", err)
	}

	out := &TenantIndex{}
	err = out.unmarshalV2Zstd(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal tenant index v2: %w", err)
	}

	return out, nil
}

// The V2 tenant index (the TenantIndexV2 proto message) is a more compact encoding of the
// same data as TenantIndex.
//
// Block metas in a tenant index repeat a lot of identical information: the version, tenant,
// dedicated columns, replication factor, etc. are the same for most blocks. V2 hoists that
// shared information into a small list of SchemaEntry messages and each BlockEntry references its
// schema by position. The file is deduplicated by construction instead of relying on
// zstd/gzip, and decoding allocates the shared fields once per schema rather than once per
// block.
//
// New fields can be added to the proto messages like any other proto. Fields shared by many
// blocks belong in SchemaEntry, the rest in BlockEntry.
//
// Options is a general purpose key/value set chosen when a block is created, so that new
// features within an encoding don't require a new encoding. Blocks with different options
// get different schemas, and blocks with the same options share one map.

// schemaKey identifies a set of shared BlockMeta fields. Dedicated columns are compared by
// hash and options by their string, which is also how compaction decides whether blocks are alike.
type schemaKey struct {
	version, tenantID string
	replicationFactor uint32
	indexPageSize     uint32
	dedicatedColumns  uint64
	options           string
}

func schemaKeyOf(m *BlockMeta) schemaKey {
	return schemaKey{
		version:           m.Version,
		tenantID:          m.TenantID,
		replicationFactor: m.ReplicationFactor,
		indexPageSize:     m.IndexPageSize,
		dedicatedColumns:  m.DedicatedColumnsHash(),
		options:           m.OptionsString(),
	}
}

func (b *TenantIndex) toV2() *TenantIndexV2 {
	out := &TenantIndexV2{
		CreatedAtUnixNanos: timeToNanos(b.CreatedAt),
		Blocks:             make([]BlockEntry, 0, len(b.Meta)),
		CompactedBlocks:    make([]CompactedBlockEntry, 0, len(b.CompactedMeta)),
		NoCompact:          b.NoCompact,
	}

	ids := map[schemaKey]uint32{}
	entry := func(m *BlockMeta) BlockEntry {
		k := schemaKeyOf(m)
		id, ok := ids[k]
		if !ok {
			id = uint32(len(out.Schemas))
			ids[k] = id
			out.Schemas = append(out.Schemas, SchemaEntry{
				Version:           m.Version,
				TenantID:          m.TenantID,
				ReplicationFactor: m.ReplicationFactor,
				IndexPageSize:     m.IndexPageSize,
				DedicatedColumns:  m.DedicatedColumns,
				Options:           m.Options,
			})
		}
		return BlockEntry{
			Schema:          id,
			BlockID:         m.BlockID,
			StartTime:       m.StartTime,
			EndTime:         m.EndTime,
			TotalObjects:    m.TotalObjects,
			Size_:           m.Size_,
			CompactionLevel: m.CompactionLevel,
			TotalRecords:    m.TotalRecords,
			BloomShardCount: m.BloomShardCount,
			FooterSize:      m.FooterSize,
		}
	}

	for _, m := range b.Meta {
		out.Blocks = append(out.Blocks, entry(m))
	}
	for _, m := range b.CompactedMeta {
		out.CompactedBlocks = append(out.CompactedBlocks, CompactedBlockEntry{
			BlockEntry:    entry(&m.BlockMeta),
			CompactedTime: m.CompactedTime,
		})
	}
	return out
}

// fromV2 replaces b's contents. Blocks of the same schema share their strings, dedicated
// columns and options, which must therefore not be mutated.
//
// TODO: This is backfilling from the better-structured v2 index into the old
// shape of a slice of full metas, which eliminates some of the savings from the new
// schematized format. In the future we could update the interface of TenantIndex to support
// both like with iter.Seq[BlockMeta] and funcs like Overlapping(start,end) that does more
// work on the fly, and would let us keep the original schemas+entries.
func (b *TenantIndex) fromV2(in *TenantIndexV2) error {
	fill := func(m *BlockMeta, e *BlockEntry) error {
		if int(e.Schema) >= len(in.Schemas) {
			return fmt.Errorf("tenant index v2: block references schema %d of %d", e.Schema, len(in.Schemas))
		}
		s := &in.Schemas[e.Schema]
		m.Version = s.Version
		m.TenantID = s.TenantID
		m.ReplicationFactor = s.ReplicationFactor
		m.IndexPageSize = s.IndexPageSize
		m.DedicatedColumns = s.DedicatedColumns
		m.Options = s.Options

		m.BlockID = e.BlockID
		m.StartTime = e.StartTime
		m.EndTime = e.EndTime
		m.TotalObjects = e.TotalObjects
		m.Size_ = e.Size_
		m.CompactionLevel = e.CompactionLevel
		m.TotalRecords = e.TotalRecords
		m.BloomShardCount = e.BloomShardCount
		m.FooterSize = e.FooterSize
		return nil
	}

	// one allocation for all blocks rather than one each
	metas := make([]BlockMeta, len(in.Blocks))
	b.Meta = make([]*BlockMeta, len(metas))
	for i := range metas {
		if err := fill(&metas[i], &in.Blocks[i]); err != nil {
			return err
		}
		b.Meta[i] = &metas[i]
	}

	compacted := make([]CompactedBlockMeta, len(in.CompactedBlocks))
	b.CompactedMeta = make([]*CompactedBlockMeta, len(compacted))
	for i := range compacted {
		if err := fill(&compacted[i].BlockMeta, &in.CompactedBlocks[i].BlockEntry); err != nil {
			return err
		}
		compacted[i].CompactedTime = in.CompactedBlocks[i].CompactedTime
		b.CompactedMeta[i] = &compacted[i]
	}

	b.NoCompact = in.NoCompact
	b.CreatedAt = nanosToTime(in.CreatedAtUnixNanos)
	return nil
}

// marshalV2 encodes the index in the V2 format without outer compression.
func (b *TenantIndex) marshalV2() ([]byte, error) {
	return b.toV2().Marshal()
}

// unmarshalV2 decodes data produced by marshalV2, replacing b's contents.
func (b *TenantIndex) unmarshalV2(data []byte) error {
	in := &TenantIndexV2{}
	if err := in.Unmarshal(data); err != nil {
		return err
	}
	return b.fromV2(in)
}

// marshalV2Zstd encodes and compresses the index, matching how the proto index is stored.
func (b *TenantIndex) marshalV2Zstd() ([]byte, error) {
	raw, err := b.marshalV2()
	if err != nil {
		return nil, err
	}
	return Zstd.Encode(raw, nil)
}

func (b *TenantIndex) unmarshalV2Zstd(buffer []byte) error {
	raw, err := Zstd.Decode(buffer)
	if err != nil {
		return fmt.Errorf("error decoding zstd: %w", err)
	}
	return b.unmarshalV2(raw)
}

// A zero time is encoded as 0 since time.Time.UnixNano is undefined for it.
func timeToNanos(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

func nanosToTime(n int64) time.Time {
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(0, n)
}
