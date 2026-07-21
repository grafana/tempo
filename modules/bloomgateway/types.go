package bloomgateway

// Handle is a per-instance, monotonically increasing, never-reclaimed
// reference to a block. Interning a block's UUID to a 32-bit handle is what
// makes the 6 B/entry (fp16 + handle32) v1 leaf encoding possible (DESIGN.md
// § Representation notes). Handles are never reused across a block's
// lifetime, even after the block is swept/reclaimed (DESIGN.md: "never
// reclaimed; 2^32 handles outlast any realistic cell lifetime").
type Handle uint32

// InvalidHandle is never allocated by the registry (WP11) and doubles as
// the zero-value sentinel for "no handle assigned yet".
const InvalidHandle Handle = 0

// BlockState is the block registry's state machine (DESIGN.md § Data model,
// as amended by AMENDMENT A1 to add the Live -> LiveUnsupportedEncoding
// demotion edge):
//
//	BlockPending -> BlockLive -> BlockDeleted (terminal)
//	BlockPending -> BlockLiveUnsupportedEncoding -> BlockDeleted (terminal)
//	BlockLive -> BlockLiveUnsupportedEncoding (demotion, AMENDMENT A1)
//
// BlockLiveUnsupportedEncoding is NOT legal to demote back to BlockLive.
// BlockDeleted is terminal from every other state: Adds for a deleted UUID
// are no-ops (no resurrection, DESIGN.md § Block object).
type BlockState int8

const (
	// BlockPending is the state from the first AddChunk seen for a block
	// UUID until every chunk_index in [0, chunk_count) has been applied.
	// This is also the zero value, so a zero-initialized Block{} correctly
	// reads as "not yet live" rather than accidentally live.
	BlockPending BlockState = iota

	// BlockLive means every chunk has been applied: the block is in the
	// registry and its handle is in every A_T bucket its time range
	// overlaps. Only BlockLive (and BlockLiveUnsupportedEncoding, which
	// never enters A_T) blocks exist in the registry's "live" view.
	BlockLive

	// BlockLiveUnsupportedEncoding is a live block whose parquet encoding
	// this reader cannot column-project (§0 D7 / AMENDMENT A1). It behaves
	// like BlockLive for registry bookkeeping purposes but is NEVER
	// inserted into (and is removed from, on demotion from BlockLive) A_T
	// — so it can never be rejected, and instead rides the existing
	// "unknown to gateway, must be searched" wire semantics for free.
	BlockLiveUnsupportedEncoding

	// BlockDeleted is terminal: reached via a Delete event from any prior
	// state. Adds for a BlockDeleted UUID are no-ops, which is what makes
	// replayed and late chunks harmless (DESIGN.md § Event processing).
	BlockDeleted
)

// String implements fmt.Stringer for readable test failures and log lines.
func (s BlockState) String() string {
	switch s {
	case BlockPending:
		return "pending"
	case BlockLive:
		return "live"
	case BlockLiveUnsupportedEncoding:
		return "live_unsupported_encoding"
	case BlockDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}
