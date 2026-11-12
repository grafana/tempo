package backend

import (
	"slices"

	"github.com/cespare/xxhash/v2"
)

// OptionsHash hashes the block options, independent of map iteration order. It is 0 when there
// are no options. Blocks with different options must not be compacted together.
func (b *BlockMeta) OptionsHash() uint64 {
	if len(b.Options) == 0 {
		return 0
	}
	keys := make([]string, 0, len(b.Options))
	for k := range b.Options {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	h := xxhash.New()
	for _, k := range keys {
		_, _ = h.WriteString(k)
		_, _ = h.Write(separatorByte)
		_, _ = h.WriteString(b.Options[k])
		_, _ = h.Write(separatorByte)
	}
	return h.Sum64()
}
