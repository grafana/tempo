package generator

import (
	"iter"

	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/tempopb"
)

// codec is the interface used to convert data from Kafka records to the
// tempopb.PushSpansRequest expected by the generator processors.
type codec interface {
	decode([]byte) (iter.Seq2[*tempopb.PushSpansRequest, error], error)
}

// tempoDecoder unmarshals tempopb.PushBytesRequest.
type tempoDecoder struct {
	dec *ingest.Decoder
}

func newTempoDecoder() *tempoDecoder {
	return &tempoDecoder{dec: ingest.NewDecoder()}
}

// decode implements codec.
func (d *tempoDecoder) decode(data []byte) (iter.Seq2[*tempopb.PushSpansRequest, error], error) {
	d.dec.Reset()
	spanBytes, err := d.dec.Decode(data)
	if err != nil {
		return nil, err
	}

	trace := tempopb.Trace{}
	return func(yield func(*tempopb.PushSpansRequest, error) bool) {
		for _, tr := range spanBytes.Traces {
			trace.Reset()
			err = trace.Unmarshal(tr.Slice)
			yield(&tempopb.PushSpansRequest{Batches: trace.ResourceSpans}, err)
		}
	}, nil
}

// otlpDecoder unmarshals ptrace.Traces.
type otlpDecoder struct {
	trace tempopb.Trace
}

func newOTLPDecoder() *otlpDecoder {
	return &otlpDecoder{trace: tempopb.Trace{}}
}

// decode implements codec.
func (d *otlpDecoder) decode(data []byte) (iter.Seq2[*tempopb.PushSpansRequest, error], error) {
	d.trace.ResourceSpans = d.trace.ResourceSpans[:0]
	err := d.trace.Unmarshal(data)
	if err != nil {
		return nil, err
	}

	return func(yield func(*tempopb.PushSpansRequest, error) bool) {
		yield(&tempopb.PushSpansRequest{Batches: d.trace.ResourceSpans}, nil)
	}, nil
}
