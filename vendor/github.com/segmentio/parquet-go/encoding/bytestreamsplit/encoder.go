package bytestreamsplit

import (
	"io"
	"math"

	"github.com/segmentio/parquet-go/encoding"
)

type Encoder struct {
	encoding.NotSupportedEncoder
	writer io.Writer
	buffer []byte
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		writer: w,
	}
}

func (e *Encoder) Write(b []byte) (int, error) {
	return e.writer.Write(b)
}

func (e *Encoder) Reset(w io.Writer) {
	e.writer = w
	e.buffer = e.buffer[:0]
}

func (e *Encoder) EncodeFloat(data []float32) error {
	_, err := e.writer.Write(e.encode32(data))
	return err
}

func (e *Encoder) EncodeDouble(data []float64) error {
	_, err := e.writer.Write(e.encode64(data))
	return err
}

func (e *Encoder) encode32(data []float32) []byte {
	length := len(data)
	if length == 0 {
		return []byte{}
	}

	if len(e.buffer) < length*4 {
		e.buffer = make([]byte, length*4)
	}

	for i, f := range data {
		bits := math.Float32bits(f)
		e.buffer[i] = byte(bits)
		e.buffer[i+length] = byte(bits >> 8)
		e.buffer[i+length*2] = byte(bits >> 16)
		e.buffer[i+length*3] = byte(bits >> 24)
	}

	return e.buffer[:length*4]
}

func (e *Encoder) encode64(data []float64) []byte {
	length := len(data)
	if length == 0 {
		return []byte{}
	}

	if len(e.buffer) < length*8 {
		e.buffer = make([]byte, length*8)
	}

	for i, f := range data {
		bits := math.Float64bits(f)
		e.buffer[i] = byte(bits)
		e.buffer[i+length] = byte(bits >> 8)
		e.buffer[i+length*2] = byte(bits >> 16)
		e.buffer[i+length*3] = byte(bits >> 24)
		e.buffer[i+length*4] = byte(bits >> 32)
		e.buffer[i+length*5] = byte(bits >> 40)
		e.buffer[i+length*6] = byte(bits >> 48)
		e.buffer[i+length*7] = byte(bits >> 56)
	}

	return e.buffer[:length*8]
}
