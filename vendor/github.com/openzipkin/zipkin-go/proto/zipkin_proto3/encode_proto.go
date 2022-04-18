// Copyright 2022 The OpenZipkin Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zipkin_proto3

import (
	"encoding/binary"
	"errors"
	"time"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"google.golang.org/protobuf/proto"
)

var errNilProtoSpan = errors.New("expecting a non-nil Span")

// SpanSerializer implements http.SpanSerializer
type SpanSerializer struct{}

// Serialize takes an array of zipkin SpanModel objects and serializes it to a protobuf blob.
func (SpanSerializer) Serialize(sms []*zipkinmodel.SpanModel) (protoBlob []byte, err error) {
	var listOfSpans ListOfSpans

	for _, sm := range sms {
		sp, err := modelSpanToProtoSpan(sm)
		if err != nil {
			return nil, err
		}
		listOfSpans.Spans = append(listOfSpans.Spans, sp)
	}

	return proto.Marshal(&listOfSpans)
}

// ContentType returns the ContentType needed for this encoding.
func (SpanSerializer) ContentType() string {
	return "application/x-protobuf"
}

func modelSpanToProtoSpan(sm *zipkinmodel.SpanModel) (*Span, error) {
	if sm == nil {
		return nil, errNilProtoSpan
	}

	traceID := make([]byte, 16)
	binary.BigEndian.PutUint64(traceID[0:8], uint64(sm.TraceID.High))
	binary.BigEndian.PutUint64(traceID[8:16], uint64(sm.TraceID.Low))

	parentID := make([]byte, 8)
	if sm.ParentID != nil {
		binary.BigEndian.PutUint64(parentID, uint64(*sm.ParentID))
	}

	id := make([]byte, 8)
	binary.BigEndian.PutUint64(id, uint64(sm.ID))

	var timeStamp uint64
	if !sm.Timestamp.IsZero() {
		timeStamp = uint64(sm.Timestamp.Round(time.Microsecond).UnixNano() / 1e3)
	}

	return &Span{
		TraceId:        traceID,
		ParentId:       parentID,
		Id:             id,
		Debug:          sm.Debug,
		Kind:           Span_Kind(Span_Kind_value[string(sm.Kind)]),
		Name:           sm.Name,
		Timestamp:      timeStamp,
		Tags:           sm.Tags,
		Duration:       uint64(sm.Duration.Nanoseconds() / 1e3),
		LocalEndpoint:  modelEndpointToProtoEndpoint(sm.LocalEndpoint),
		RemoteEndpoint: modelEndpointToProtoEndpoint(sm.RemoteEndpoint),
		Shared:         sm.Shared,
		Annotations:    modelAnnotationsToProtoAnnotations(sm.Annotations),
	}, nil
}

func durationToMicros(d time.Duration) (uint64, error) {
	if d < time.Microsecond {
		if d < 0 {
			return 0, zipkinmodel.ErrValidDurationRequired
		} else if d > 0 {
			d = 1 * time.Microsecond
		}
	} else {
		d += 500 * time.Nanosecond
	}
	return uint64(d.Nanoseconds() / 1e3), nil
}

func modelEndpointToProtoEndpoint(ep *zipkinmodel.Endpoint) *Endpoint {
	if ep == nil {
		return nil
	}
	return &Endpoint{
		ServiceName: ep.ServiceName,
		Ipv4:        []byte(ep.IPv4),
		Ipv6:        []byte(ep.IPv6),
		Port:        int32(ep.Port),
	}
}

func modelAnnotationsToProtoAnnotations(mas []zipkinmodel.Annotation) (pas []*Annotation) {
	for _, ma := range mas {
		pas = append(pas, &Annotation{
			Timestamp: timeToMicros(ma.Timestamp),
			Value:     ma.Value,
		})
	}
	return
}

func timeToMicros(t time.Time) uint64 {
	if t.IsZero() {
		return 0
	}
	return uint64(t.UnixNano()) / 1e3
}
