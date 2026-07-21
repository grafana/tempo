package common

import "context"

// TraceIDIterator enumerates the trace IDs stored in a block via a
// column-projected read: only the TraceID column is decoded, never the
// full row the way Iterator does. This is the shape a caller needs to
// backfill or repair state that only cares about trace-ID membership
// (modules/bloomgateway's reconstruction and reconciliation loops, per
// its DESIGN.md § Reconstruction) without paying for a full-row scan.
//
// See tempodb/encoding.TraceIDProjector for the optional per-encoding
// capability that produces one of these.
type TraceIDIterator interface {
	// Next returns the next trace ID in the block, or io.EOF once every
	// trace ID has been returned. Implementations must check ctx on every
	// call so a caller cancelling between reads gets a prompt error
	// rather than one more row.
	Next(ctx context.Context) (ID, error)

	// Close releases resources held by the iterator. Safe to call more
	// than once.
	Close()
}
