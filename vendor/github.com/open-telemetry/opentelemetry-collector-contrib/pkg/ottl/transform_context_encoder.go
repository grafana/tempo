// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"reflect"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	transformContextKey = "TransformContext"
	deletedReplacement  = "<deleted>"
)

// newTransformContextField creates a zapcore.Field that can be used with any OTTL parser
// transform context object. It marshals the PData safely by filtering out invalid values,
// avoiding panics. It is intended for debug logging only and should not be used in
// production code.
func newTransformContextField(tCtx any) zap.Field {
	return zap.Inline(&transformContextMarshaller{tCtx})
}

type transformContextMarshaller struct {
	tCtx any
}

func (m *transformContextMarshaller) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	tCtxEncoder := &transformContextEncoder{ObjectEncoder: encoder}
	zap.Any(transformContextKey, m.tCtx).AddTo(tCtxEncoder)
	return nil
}

type transformContextEncoder struct {
	zapcore.ObjectEncoder
	zapcore.ArrayEncoder
}

func (m *transformContextEncoder) withArrayEncoder(enc zapcore.ArrayEncoder) *transformContextEncoder {
	return &transformContextEncoder{ObjectEncoder: m.ObjectEncoder, ArrayEncoder: enc}
}

func (m *transformContextEncoder) withObjectEncoder(enc zapcore.ObjectEncoder) *transformContextEncoder {
	return &transformContextEncoder{ObjectEncoder: enc, ArrayEncoder: m.ArrayEncoder}
}

func (m *transformContextEncoder) AddArray(k string, v zapcore.ArrayMarshaler) error {
	return m.ObjectEncoder.AddArray(k, zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
		encoder := m.withArrayEncoder(enc)
		if isInvalidPData(v) {
			encoder.AppendString(deletedReplacement)
			return nil
		}
		return v.MarshalLogArray(encoder)
	}))
}

func (m *transformContextEncoder) AddObject(k string, v zapcore.ObjectMarshaler) error {
	return m.ObjectEncoder.AddObject(k, zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		encoder := m.withObjectEncoder(enc)
		if isInvalidPData(v) {
			encoder.AddString(deletedReplacement, deletedReplacement)
			return nil
		}
		return v.MarshalLogObject(encoder)
	}))
}

func (m *transformContextEncoder) AddReflected(k string, v any) error {
	if isInvalidPData(v) {
		m.AddString(k, deletedReplacement)
		return nil
	}
	return m.ObjectEncoder.AddReflected(k, v)
}

func (m *transformContextEncoder) OpenNamespace(k string) {
	m.ObjectEncoder.OpenNamespace(k)
}

func (m *transformContextEncoder) AppendArray(v zapcore.ArrayMarshaler) error {
	return m.ArrayEncoder.AppendArray(zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
		encoder := m.withArrayEncoder(enc)
		if isInvalidPData(v) {
			encoder.AppendString(deletedReplacement)
			return nil
		}
		return v.MarshalLogArray(encoder)
	}))
}

func (m *transformContextEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
	return m.ArrayEncoder.AppendObject(zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		encoder := m.withObjectEncoder(enc)
		if isInvalidPData(v) {
			encoder.AddString(deletedReplacement, deletedReplacement)
			return nil
		}
		return v.MarshalLogObject(encoder)
	}))
}

func (m *transformContextEncoder) AppendReflected(v any) error {
	if isInvalidPData(v) {
		m.AppendString(deletedReplacement)
		return nil
	}
	return m.ArrayEncoder.AppendReflected(v)
}

// isInvalidPData reports whether `data` refers to an invalid PData value.
// A PData becomes invalid after it is deleted from its parent; in that state the underlying
// `orig` field is set to nil, and attempting to marshal it may panic.
// There is no stable public API to detect this today, so this function uses reflection to
// look for an `orig` field and treat a nil `orig` as invalid.
func isInvalidPData(data any) bool {
	if data == nil {
		return true
	}

	v := reflect.ValueOf(data)
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return false
	}

	origField := v.FieldByName("orig")
	if !origField.IsValid() {
		return false
	}

	origKind := origField.Kind()
	return (origKind == reflect.Pointer ||
		origKind == reflect.Interface ||
		origKind == reflect.Slice ||
		origKind == reflect.Map ||
		origKind == reflect.Func ||
		origKind == reflect.Chan) && origField.IsNil()
}
