package registry

import "github.com/cespare/xxhash/v2"

// separatorByte is a byte that cannot occur in valid UTF-8 sequences
var separatorByte = []byte{255}

// hashLabelValues generates a unique hash for the label values of a metric series. It expects that
// labelValues will always have the same length.
func hashLabelValues(labels LabelPair) uint64 {
	h := xxhash.New()

	for _, v := range labels.names {
		_, _ = h.WriteString(v)
		_, _ = h.Write(separatorByte)
	}

	for _, v := range labels.values {
		_, _ = h.WriteString(v)
		_, _ = h.Write(separatorByte)
	}
	return h.Sum64()
}
