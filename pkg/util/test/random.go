package test

import (
	"context"
	"math/rand"
	"time"

	v1_common "github.com/grafana/tempo/pkg/tempopb/opentelemetry/proto/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/opentelemetry/proto/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

// RandomBatcher is a helper for generating random batches of spans.
type RandomBatcher struct {
	stringReceiverChan    chan string
	attributeReceiverChan chan *v1_common.KeyValue
	anyvalueReceiverChan  chan *v1_common.AnyValue
	spanReceiverChan      chan *v1_trace.Span
}

func NewRandomBatcher() (*RandomBatcher, context.CancelFunc) {
	r := &RandomBatcher{
		stringReceiverChan:    make(chan string, 100),
		attributeReceiverChan: make(chan *v1_common.KeyValue, 100),
		anyvalueReceiverChan:  make(chan *v1_common.AnyValue, 100),
		spanReceiverChan:      make(chan *v1_trace.Span, 100),
	}

	ctx, cancel := context.WithCancel(context.Background())

	go r.randomStringGenerator(ctx, 1, 64)
	go r.randomSpanAttributeGenerator(ctx)
	go r.randomAnyValueGenerator(ctx)
	go r.randomSpanGenerator(ctx)

	return r, cancel
}

func (r *RandomBatcher) GenerateBatch(spanCount int64) *v1_trace.ResourceSpans {
	batch := &v1_trace.ResourceSpans{
		Resource: &v1_resource.Resource{
			Attributes: []*v1_common.KeyValue{
				{
					Key: "service.name",
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_StringValue{
							StringValue: "test-service",
						},
					},
				},
			},
		},
	}

	for i := int64(0); i < spanCount; i++ {
		s := <-r.spanReceiverChan
		batch.ScopeSpans = append(batch.ScopeSpans, &v1_trace.ScopeSpans{
			Scope: &v1_common.InstrumentationScope{
				Name:    "super library",
				Version: "0.0.1",
			},
			Spans: []*v1_trace.Span{
				s,
			},
		})
	}

	return batch
}

func (r *RandomBatcher) randomSpanGenerator(ctx context.Context) {
	min := 0
	max := 30

	rising := true
	length := min

	rr := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		select {
		case <-ctx.Done():
			return
		default:

			attributes := []*v1_common.KeyValue{}

			for i := 0; i < length; i++ {
				attributes = append(attributes, <-r.attributeReceiverChan)
			}

			now := time.Now()

			span := &v1_trace.Span{
				TraceId:           []byte("12345678901234567890123456789012"),
				SpanId:            []byte("1234567890123456"),
				ParentSpanId:      []byte("1234567890123456"),
				Name:              <-r.stringReceiverChan,
				StartTimeUnixNano: uint64(now.UnixNano()),
				EndTimeUnixNano:   uint64(now.Add(time.Duration(rr.Intn(1000000000)) * time.Nanosecond).UnixNano()),
				Attributes:        attributes,
			}
			r.spanReceiverChan <- span

			if rising {
				length++
			} else {
				length--
			}

			switch length {
			case max:
				rising = false
			case min:
				rising = true
			}

		}
	}
}

func (r *RandomBatcher) randomAnyValueGenerator(ctx context.Context) {
	rr := rand.New(rand.NewSource(time.Now().UnixNano()))

	var anyValue *v1_common.AnyValue

	for {
		select {
		case <-ctx.Done():
			return
		default:

			switch rr.Intn(4) {
			case 0:
				anyValue = &v1_common.AnyValue{
					Value: &v1_common.AnyValue_StringValue{
						StringValue: <-r.stringReceiverChan,
					},
				}
			case 1:
				anyValue = &v1_common.AnyValue{
					Value: &v1_common.AnyValue_BoolValue{
						BoolValue: bool(rr.Intn(2) == 1),
					},
				}
			case 2:
				anyValue = &v1_common.AnyValue{
					Value: &v1_common.AnyValue_IntValue{
						IntValue: int64(rr.Intn(1000000000)),
					},
				}
			case 3:
				anyValue = &v1_common.AnyValue{
					Value: &v1_common.AnyValue_DoubleValue{
						DoubleValue: rr.Float64(),
					},
				}
			}

			r.anyvalueReceiverChan <- anyValue
		}
	}
}

func (r *RandomBatcher) randomSpanAttributeGenerator(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			attr := &v1_common.KeyValue{
				Key:   <-r.stringReceiverChan,
				Value: <-r.anyvalueReceiverChan,
			}
			r.attributeReceiverChan <- attr
		}
	}
}

func (r *RandomBatcher) randomStringGenerator(ctx context.Context, min, max int) {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

	rising := true
	length := min
	for {
		select {
		case <-ctx.Done():
			return
		default:
			s := make([]rune, length)
			for i := range s {
				s[i] = letters[rand.Intn(len(letters))]
			}

			r.stringReceiverChan <- string(s)

			if rising {
				length++
			} else {
				length--
			}

			switch length {
			case max:
				rising = false
			case min:
				rising = true
			}
		}
	}
}
