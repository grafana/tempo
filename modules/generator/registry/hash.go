package registry

import "github.com/cespare/xxhash/v2"

// separatorByte is a byte that cannot occur in valid UTF-8 sequences
var separatorByte = []byte{255}

// hashLabelValues generates a unique hash for the label values of a metric series. It expects that
// labelValues will always have the same length.
func hashLabelValues(labelValues []string) uint64 {
	h := xxhash.New()
	for _, v := range labelValues {
		_, _ = h.WriteString(v)
		_, _ = h.Write(separatorByte)
	}
	return h.Sum64()
}
