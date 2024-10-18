package backend

import (
	"sync"

	"github.com/klauspost/compress/zstd"
)

var _ Codec = (*ZstdCodec)(nil)

type Codec interface {
	Encode([]byte, []byte) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

type ZstdCodec struct {
	encoders sync.Pool // *zstd.Encoder
	decoders sync.Pool // *zstd.Decoder
}

func (c *ZstdCodec) Encode(src, dst []byte) ([]byte, error) {
	e, _ := c.encoders.Get().(*zstd.Encoder)
	if e == nil {
		var err error
		e, err = zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1))
		if err != nil {
			return nil, err
		}
	}
	defer c.encoders.Put(e)
	return e.EncodeAll(src, dst), nil
}

func (c *ZstdCodec) Decode(buf []byte) ([]byte, error) {
	d, _ := c.decoders.Get().(*zstd.Decoder)
	if d == nil {
		var err error
		d, err = zstd.NewReader(nil, zstd.WithDecoderConcurrency(0))
		if err != nil {
			return nil, err
		}
	}
	defer c.decoders.Put(d)
	return d.DecodeAll(buf, nil)
}
