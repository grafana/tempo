package plain

import (
	"io"

	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/format"
)

type DictionaryEncoding struct {
}

func (e DictionaryEncoding) Encoding() format.Encoding {
	return format.PlainDictionary
}

func (e DictionaryEncoding) CanEncode(t format.Type) bool {
	return true
}

func (e DictionaryEncoding) NewDecoder(r io.Reader) encoding.Decoder {
	return NewDecoder(r)
}

func (e DictionaryEncoding) NewEncoder(w io.Writer) encoding.Encoder {
	return NewEncoder(w)
}

func (e DictionaryEncoding) String() string {
	return "PLAIN_DICTIONARY"
}
