package spanlogger

import (
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

type noopTracer struct{}

type (
	noopSpan        struct{}
	noopSpanContext struct{}
)

var (
	defaultNoopSpanContext = noopSpanContext{}
	defaultNoopSpan        = noopSpan{}
	defaultNoopTracer      = noopTracer{}
)

const (
	emptyString = ""
)

func (n noopSpanContext) ForeachBaggageItem(func(k, v string) bool) {}

func (n noopSpan) Context() opentracing.SpanContext               { return defaultNoopSpanContext }
func (n noopSpan) SetBaggageItem(string, string) opentracing.Span { return defaultNoopSpan }
func (n noopSpan) BaggageItem(string) string                      { return emptyString }
func (n noopSpan) SetTag(string, interface{}) opentracing.Span    { return n }
func (n noopSpan) LogFields(...log.Field)                         {}
func (n noopSpan) LogKV(...interface{})                           {}
func (n noopSpan) Finish()                                        {}
func (n noopSpan) FinishWithOptions(opentracing.FinishOptions)    {}
func (n noopSpan) SetOperationName(string) opentracing.Span       { return n }
func (n noopSpan) Tracer() opentracing.Tracer                     { return defaultNoopTracer }
func (n noopSpan) LogEvent(string)                                {}
func (n noopSpan) LogEventWithPayload(string, interface{})        {}
func (n noopSpan) Log(opentracing.LogData)                        {}

// StartSpan belongs to the Tracer interface.
func (n noopTracer) StartSpan(string, ...opentracing.StartSpanOption) opentracing.Span {
	return defaultNoopSpan
}

// Inject belongs to the Tracer interface.
func (n noopTracer) Inject(opentracing.SpanContext, interface{}, interface{}) error {
	return nil
}

// Extract belongs to the Tracer interface.
func (n noopTracer) Extract(interface{}, interface{}) (opentracing.SpanContext, error) {
	return nil, opentracing.ErrSpanContextNotFound
}
