package tempofb

// TagContainer is anything with KeyValues (tags). This is implemented by both
// SearchPage and SearchEntry.
type TagContainer interface {
	Contains(k []byte, v []byte, buffer *KeyValues) bool
}

type Trace interface {
	TagContainer
	StartTimeUnixNano() uint64
	EndTimeUnixNano() uint64
}

var _ Trace = (*SearchEntry)(nil)

type Page interface {
	TagContainer
}

var _ Page = (*SearchPage)(nil)

type Block interface {
	TagContainer
	MinDurationNanos() uint64
	MaxDurationNanos() uint64
}

var _ Block = (*SearchBlockHeader)(nil)
var _ Block = (*SearchBlockHeaderMutable)(nil)
