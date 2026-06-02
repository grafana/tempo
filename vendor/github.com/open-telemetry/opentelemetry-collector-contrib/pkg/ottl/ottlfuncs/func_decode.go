// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"golang.org/x/text/encoding"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/textutils"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type decoder interface {
	Decode(src []byte) (any, error)
	DecodeString(src string) (any, error)
}

type textDecoder struct {
	enc encoding.Encoding
}

func (td textDecoder) Decode(src []byte) (any, error) {
	ret, err := td.enc.NewDecoder().Bytes(src)
	if err != nil {
		return nil, fmt.Errorf("could not decode: %w", err)
	}
	return string(ret), nil
}

func (td textDecoder) DecodeString(src string) (any, error) {
	decodedString, err := td.enc.NewDecoder().String(src)
	if err != nil {
		return nil, fmt.Errorf("could not decode: %w", err)
	}
	return decodedString, nil
}

type base64Decoder struct {
	enc *base64.Encoding
}

func (bd base64Decoder) Decode(src []byte) (any, error) {
	dbuf := make([]byte, bd.enc.DecodedLen(len(src)))
	n, err := bd.enc.Decode(dbuf, src)
	if err != nil {
		return nil, fmt.Errorf("could not decode: %w", err)
	}
	return string(dbuf[:n]), nil
}

func (bd base64Decoder) DecodeString(src string) (any, error) {
	buf, err := bd.enc.DecodeString(src)
	if err != nil {
		return nil, fmt.Errorf("could not decode: %w", err)
	}
	return string(buf), nil
}

type DecodeArguments[K any] struct {
	Target   ottl.Getter[K]
	Encoding ottl.StringGetter[K]
}

func NewDecodeFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Decode", &DecodeArguments[K]{}, createDecodeFunction[K])
}

func createDecodeFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*DecodeArguments[K])
	if !ok {
		return nil, errors.New("DecodeFactory args must be of type *DecodeArguments[K]")
	}

	return decode(args.Target, args.Encoding)
}

func decode[K any](target ottl.Getter[K], encoding ottl.StringGetter[K]) (ottl.ExprFunc[K], error) {
	decGet, err := newDecoderGetter(encoding)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		dec, err := decGet.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		switch v := val.(type) {
		case []byte:
			return dec.Decode(v)
		case *string:
			return dec.DecodeString(*v)
		case string:
			return dec.DecodeString(v)
		case pcommon.ByteSlice:
			return dec.Decode(v.AsRaw())
		case *pcommon.ByteSlice:
			return dec.Decode(v.AsRaw())
		case pcommon.Value:
			if v.Type() == pcommon.ValueTypeBytes {
				return dec.Decode(v.Bytes().AsRaw())
			}
			return dec.DecodeString(v.AsString())
		case *pcommon.Value:
			if v.Type() == pcommon.ValueTypeBytes {
				return dec.Decode(v.Bytes().AsRaw())
			}
			return dec.DecodeString(v.AsString())
		default:
			return nil, fmt.Errorf("unsupported type provided to Decode function: %T", v)
		}
	}, nil
}

type decoderGetter[K any] struct {
	decoder decoder
	getter  ottl.StringGetter[K]
}

func newDecoderGetter[K any](getter ottl.StringGetter[K]) (*decoderGetter[K], error) {
	if enc, isLiteral := ottl.GetLiteralValue(getter); isLiteral {
		dec, err := asDecoder(enc)
		if err != nil {
			return nil, err
		}
		return &decoderGetter[K]{decoder: dec}, nil
	}
	return &decoderGetter[K]{getter: getter}, nil
}

func (d *decoderGetter[K]) Get(ctx context.Context, tCtx K) (decoder, error) {
	if d.decoder != nil {
		return d.decoder, nil
	}
	enc, err := d.getter.Get(ctx, tCtx)
	if err != nil {
		return nil, err
	}
	return asDecoder(enc)
}

func asDecoder(encodingVal string) (decoder, error) {
	switch encodingVal {
	// base64 is not in IANA index, so we have to deal with this encoding separately
	case "base64":
		return base64Decoder{enc: base64.StdEncoding}, nil
	case "base64-raw":
		return base64Decoder{enc: base64.RawStdEncoding}, nil
	case "base64-url":
		return base64Decoder{enc: base64.URLEncoding}, nil
	case "base64-raw-url":
		return base64Decoder{enc: base64.RawURLEncoding}, nil
	default:
		e, err := textutils.LookupEncoding(encodingVal)
		if err != nil {
			return nil, err
		}
		return textDecoder{enc: e}, nil
	}
}
