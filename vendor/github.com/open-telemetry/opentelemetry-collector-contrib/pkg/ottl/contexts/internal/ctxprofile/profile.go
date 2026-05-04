// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxprofile // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxprofile"

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcommon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxerror"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxprofilecommon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxutil"
)

func PathGetSetter[K Context](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	if path == nil {
		return nil, ctxerror.New("nil", "nil", Name, DocRef)
	}
	switch path.Name() {
	case "sample_type":
		return accessSampleType(path)
	case "sample":
		return accessSample[K](), nil
	case "time_unix_nano":
		return accessTimeUnixNano[K](), nil
	case "time":
		return accessTime[K](), nil
	case "duration_unix_nano":
		return accessDurationUnixNano[K](), nil
	case "duration":
		return accessDuration[K](), nil
	case "period_type":
		return accessPeriodType[K](path)
	case "period":
		return accessPeriod[K](), nil
	case "profile_id":
		nextPath := path.Next()
		if nextPath != nil {
			if nextPath.Name() == "string" {
				return accessStringProfileID[K](), nil
			}
			return nil, ctxerror.New(nextPath.Name(), nextPath.String(), Name, DocRef)
		}
		return accessProfileID[K](), nil
	case "attribute_indices":
		return accessAttributeIndices[K](), nil
	case "dropped_attributes_count":
		return accessDroppedAttributesCount[K](), nil
	case "original_payload_format":
		return accessOriginalPayloadFormat[K](), nil
	case "original_payload":
		return accessOriginalPayload[K](), nil
	case "attributes":
		attributable := func(ctx K) (pprofile.ProfilesDictionary, ctxprofilecommon.ProfileAttributable) {
			return ctx.GetProfilesDictionary(), ctx.GetProfile()
		}
		if path.Keys() == nil {
			return ctxprofilecommon.AccessAttributes[K](attributable), nil
		}
		return ctxprofilecommon.AccessAttributesKey[K](path.Keys(), attributable), nil
	default:
		return nil, ctxerror.New(path.Name(), path.String(), Name, DocRef)
	}
}

func accessSample[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetProfile().Samples(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			v, err := ctxutil.ExpectType[pprofile.SampleSlice](val)
			if err != nil {
				return err
			}
			v.CopyTo(tCtx.GetProfile().Samples())
			return nil
		},
	}
}

func accessTimeUnixNano[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetProfile().Time().AsTime().UnixNano(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			i, err := ctxutil.ExpectType[int64](val)
			if err != nil {
				return err
			}
			tCtx.GetProfile().SetTime(pcommon.NewTimestampFromTime(time.Unix(0, i)))
			return nil
		},
	}
}

func accessTime[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetProfile().Time().AsTime(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			i, err := ctxutil.ExpectType[time.Time](val)
			if err != nil {
				return err
			}
			tCtx.GetProfile().SetTime(pcommon.NewTimestampFromTime(i))
			return nil
		},
	}
}

func accessDurationUnixNano[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return int64(tCtx.GetProfile().DurationNano()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			t, err := ctxutil.ExpectType[int64](val)
			if err != nil {
				return err
			}
			if t < 0 {
				return errors.New("duration_unix_nano must be non-negative")
			}
			tCtx.GetProfile().SetDurationNano(uint64(t))
			return nil
		},
	}
}

func accessDuration[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return int64(tCtx.GetProfile().DurationNano()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			t, err := ctxutil.ExpectType[int64](val)
			if err != nil {
				return err
			}
			if t < 0 {
				return errors.New("duration_unix_nano must be non-negative")
			}
			tCtx.GetProfile().SetDurationNano(uint64(t))
			return nil
		},
	}
}

func accessPeriodType[K Context](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	periodTypeTarget := func(tCtx K) pprofile.ValueType {
		return tCtx.GetProfile().PeriodType()
	}
	return valueTypeGetterSetter(path, periodTypeTarget)
}

func accessPeriod[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetProfile().Period(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			i, err := ctxutil.ExpectType[int64](val)
			if err != nil {
				return err
			}
			tCtx.GetProfile().SetPeriod(i)
			return nil
		},
	}
}

func accessSampleType[K Context](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	sampleTypeTarget := func(tCtx K) pprofile.ValueType {
		return tCtx.GetProfile().SampleType()
	}
	return valueTypeGetterSetter(path, sampleTypeTarget)
}

func accessProfileID[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetProfile().ProfileID(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			id, err := ctxutil.ExpectType[pprofile.ProfileID](val)
			if err != nil {
				return err
			}
			if id.IsEmpty() {
				return errors.New("profile ids must not be empty")
			}
			tCtx.GetProfile().SetProfileID(id)
			return nil
		},
	}
}

func accessStringProfileID[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			id := tCtx.GetProfile().ProfileID()
			return hex.EncodeToString(id[:]), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			s, err := ctxutil.ExpectType[string](val)
			if err != nil {
				return err
			}
			id, err := ctxcommon.ParseProfileID(s)
			if err != nil {
				return err
			}
			if id.IsEmpty() {
				return errors.New("profile ids must not be empty")
			}
			tCtx.GetProfile().SetProfileID(id)
			return nil
		},
	}
}

func accessAttributeIndices[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return ctxutil.GetCommonIntSliceValues[int32](tCtx.GetProfile().AttributeIndices()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			return ctxutil.SetCommonIntSliceValues[int32](tCtx.GetProfile().AttributeIndices(), val)
		},
	}
}

func accessDroppedAttributesCount[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return int64(tCtx.GetProfile().DroppedAttributesCount()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			i, err := ctxutil.ExpectType[int64](val)
			if err != nil {
				return err
			}
			tCtx.GetProfile().SetDroppedAttributesCount(uint32(i))
			return nil
		},
	}
}

func accessOriginalPayloadFormat[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetProfile().OriginalPayloadFormat(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			v, err := ctxutil.ExpectType[string](val)
			if err != nil {
				return err
			}
			tCtx.GetProfile().SetOriginalPayloadFormat(v)
			return nil
		},
	}
}

func accessOriginalPayload[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetProfile().OriginalPayload().AsRaw(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			v, err := ctxutil.ExpectType[[]byte](val)
			if err != nil {
				return err
			}
			tCtx.GetProfile().OriginalPayload().FromRaw(v)
			return nil
		},
	}
}
