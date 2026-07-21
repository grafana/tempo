package encoding

import (
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/encoding/vparquet5"
)

// TraceIDProjector is an optional capability: an encoding that can open a
// column-projected reader over a block's trace IDs implements this
// interface. It is deliberately not a method on VersionedEncoding --
// callers type-assert for it, so encodings that don't implement it need no
// stub method and no edit to this package. Currently only vparquet5
// implements it; vparquet3, vparquet4, and unsupported are untouched.
type TraceIDProjector interface {
	// OpenTraceIDReader opens a reader over the block's trace IDs. Unlike
	// OpenBlock (whose Iterator decodes every column of every row), this
	// never decodes anything but the trace ID.
	OpenTraceIDReader(meta *backend.BlockMeta, r backend.Reader) (common.TraceIDIterator, error)
}

var _ TraceIDProjector = vparquet5.Encoding{}
