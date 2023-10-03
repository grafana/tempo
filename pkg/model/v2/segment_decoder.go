package v2

import (
	"errors"
	"fmt"
	"math"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
)

// SegmentDecoder maintains the relationship between distributor -> ingester
// Segment format:
// | uint32 | uint32 | variable length          |
// | start  | end    | marshalled tempopb.Trace |
// start and end are unix epoch seconds
type SegmentDecoder struct{}

var segmentDecoder = &SegmentDecoder{}

func NewSegmentDecoder() *SegmentDecoder {
	return segmentDecoder
}

func (d *SegmentDecoder) PrepareForWrite(trace *tempopb.Trace, start uint32, end uint32) ([]byte, error) {
	return marshalWithStartEnd(trace, start, end)
}

func (d *SegmentDecoder) PrepareForRead(segments [][]byte) (*tempopb.Trace, error) {
	combiner := trace.NewCombiner(0)
	for i, obj := range segments {
		obj, _, _, err := stripStartEnd(obj)
		if err != nil {
			return nil, fmt.Errorf("error stripping start/end: %w", err)
		}

		t := &tempopb.Trace{}
		err = proto.Unmarshal(obj, t)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling trace: %w", err)
		}

		_, err = combiner.ConsumeWithFinal(t, i == len(segments)-1)
		if err != nil {
			return nil, fmt.Errorf("error combining trace: %w", err)
		}
	}

	combinedTrace, _ := combiner.Result()

	return combinedTrace, nil
}

// ToObject creates a byte slice that can be interpreted by ObjectDecoder in this package
// see object_decoder.go for details on the format.
func (d *SegmentDecoder) ToObject(segments [][]byte) ([]byte, error) {
	// strip start/end from individual segments and place it in a TraceBytesWrapper
	var err error
	var minStart, maxEnd uint32
	minStart = math.MaxUint32

	for i, b := range segments {
		var start, end uint32

		segments[i], start, end, err = stripStartEnd(b)
		if err != nil {
			return nil, err
		}
		if start < minStart {
			minStart = start
		}
		if end > maxEnd {
			maxEnd = end
		}
	}

	return marshalWithStartEnd(&tempopb.TraceBytes{
		Traces: segments,
	}, minStart, maxEnd)
}

func (d *SegmentDecoder) FastRange(buff []byte) (uint32, uint32, error) {
	_, start, end, err := stripStartEnd(buff)
	return start, end, err
}

func marshalWithStartEnd(pb proto.Message, start uint32, end uint32) ([]byte, error) {
	const uint32Size = 4

	sz := proto.Size(pb)
	buff := make([]byte, 0, sz+uint32Size*2) // proto buff size + start/end uint32s

	buffer := proto.NewBuffer(buff)

	_ = buffer.EncodeFixed32(uint64(start)) // EncodeFixed32 can't return an error
	_ = buffer.EncodeFixed32(uint64(end))
	err := buffer.Marshal(pb)
	if err != nil {
		return nil, err
	}

	buff = buffer.Bytes()

	return buff, nil
}

func stripStartEnd(buff []byte) ([]byte, uint32, uint32, error) {
	if len(buff) < 8 {
		return nil, 0, 0, errors.New("buffer too short to have start/end")
	}

	buffer := proto.NewBuffer(buff)
	start, err := buffer.DecodeFixed32()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to read start from buffer: %w", err)
	}
	end, err := buffer.DecodeFixed32()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to read end from buffer: %w", err)
	}

	return buff[8:], uint32(start), uint32(end), nil
}
