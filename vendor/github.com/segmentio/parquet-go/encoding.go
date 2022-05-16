package parquet

import (
	"sort"

	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/encoding/bytestreamsplit"
	"github.com/segmentio/parquet-go/encoding/delta"
	"github.com/segmentio/parquet-go/encoding/plain"
	"github.com/segmentio/parquet-go/encoding/rle"
	"github.com/segmentio/parquet-go/format"
)

var (
	// Plain is the default parquet encoding.
	Plain plain.Encoding

	// RLE is the hybrid bit-pack/run-length parquet encoding.
	RLE rle.Encoding

	// PlainDictionary is the plain dictionary parquet encoding.
	//
	// This encoding should not be used anymore in parquet 2.0 and later,
	// it is implemented for backwards compatibility to support reading
	// files that were encoded with older parquet libraries.
	PlainDictionary plain.DictionaryEncoding

	// RLEDictionary is the RLE dictionary parquet encoding.
	RLEDictionary rle.DictionaryEncoding

	// DeltaBinaryPacked is the delta binary packed parquet encoding.
	DeltaBinaryPacked delta.BinaryPackedEncoding

	// DeltaLengthByteArray is the delta length byte array parquet encoding.
	DeltaLengthByteArray delta.LengthByteArrayEncoding

	// DeltaByteArray is the delta byte array parquet encoding.
	DeltaByteArray delta.ByteArrayEncoding

	// ByteStreamSplit is an encoding for floating-point data.
	ByteStreamSplit bytestreamsplit.Encoding

	// Table indexing the encodings supported by this package.
	encodings = [...]encoding.Encoding{
		format.Plain:                &Plain,
		format.PlainDictionary:      &PlainDictionary,
		format.RLE:                  &RLE,
		format.RLEDictionary:        &RLEDictionary,
		format.DeltaBinaryPacked:    &DeltaBinaryPacked,
		format.DeltaLengthByteArray: &DeltaLengthByteArray,
		format.DeltaByteArray:       &DeltaByteArray,
		format.ByteStreamSplit:      &ByteStreamSplit,
	}
)

func isDictionaryEncoding(encoding encoding.Encoding) bool {
	switch encoding.Encoding() {
	case format.PlainDictionary, format.RLEDictionary:
		return true
	default:
		return false
	}
}

// LookupEncoding returns the parquet encoding associated with the given code.
//
// The function never returns nil. If the encoding is not supported,
// encoding.NotSupported is returned.
func LookupEncoding(enc format.Encoding) encoding.Encoding {
	if enc >= 0 && int(enc) < len(encodings) {
		if e := encodings[enc]; e != nil {
			return e
		}
	}
	return encoding.NotSupported{}
}

func sortEncodings(encodings []encoding.Encoding) {
	if len(encodings) > 1 {
		sort.Slice(encodings, func(i, j int) bool {
			return encodings[i].Encoding() < encodings[j].Encoding()
		})
	}
}

func dedupeSortedEncodings(encodings []encoding.Encoding) []encoding.Encoding {
	if len(encodings) > 1 {
		i := 0

		for _, c := range encodings[1:] {
			if c.Encoding() != encodings[i].Encoding() {
				i++
				encodings[i] = c
			}
		}

		clear := encodings[i+1:]
		for i := range clear {
			clear[i] = nil
		}

		encodings = encodings[:i+1]
	}
	return encodings
}
