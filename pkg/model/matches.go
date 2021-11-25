package model

import (
	"math"
	"strings"

	"github.com/grafana/tempo/pkg/tempopb"
	v1common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/search"
)

const RootSpanNotYetReceivedText = "<root span not yet received>"

// Matches determines if the passed object encoded using dataEncoding matches the tempopb.SearchRequest.
//  If the object matches the request then a non-nil tempopb.TraceSearchMetaData is returned. Otherwise
//  nil is returned.
func Matches(id []byte, obj []byte, dataEncoding string, reqStart, reqEnd uint32, req *tempopb.SearchRequest) (*tempopb.TraceSearchMetadata, error) {
	traceStart := uint64(math.MaxUint64)
	traceEnd := uint64(0)

	trace, err := Unmarshal(obj, dataEncoding)
	if err != nil {
		return nil, err
	}

	tagFound := false
	if len(req.Tags) == 0 {
		tagFound = true
	}

	var rootSpan *v1.Span
	var rootBatch *v1.ResourceSpans
	// todo: is it possible to shortcircuit this loop?
	for _, b := range trace.Batches {
		if !tagFound && searchAttributes(req.Tags, b.Resource.Attributes) {
			tagFound = true
		}

		for _, ils := range b.InstrumentationLibrarySpans {
			for _, s := range ils.Spans {
				if s.StartTimeUnixNano < traceStart {
					traceStart = s.StartTimeUnixNano
				}
				if s.EndTimeUnixNano > traceEnd {
					traceEnd = s.EndTimeUnixNano
				}
				if rootSpan == nil && len(s.ParentSpanId) == 0 {
					rootSpan = s
					rootBatch = b
				}

				if tagFound {
					continue
				}

				if searchAttributes(req.Tags, s.Attributes) {
					tagFound = true
				}
			}
		}
	}

	if !tagFound {
		return nil, nil
	}

	traceStartMs := traceStart / 1000000
	traceEndMs := traceEnd / 1000000
	durationMs := uint32(traceEndMs - traceStartMs)
	if req.MaxDurationMs != 0 && req.MaxDurationMs < durationMs {
		return nil, nil
	}
	if req.MinDurationMs != 0 && req.MinDurationMs > durationMs {
		return nil, nil
	}
	if uint32(traceStartMs/1000) > reqEnd || uint32(traceEndMs/1000) < reqStart {
		return nil, nil
	}

	// woohoo!
	rootServiceName := RootSpanNotYetReceivedText
	rootSpanName := RootSpanNotYetReceivedText
	if rootSpan != nil && rootBatch != nil {
		rootSpanName = rootSpan.Name

		for _, a := range rootBatch.Resource.Attributes {
			if a.Key == search.ServiceNameTag {
				rootServiceName = a.Value.GetStringValue()
				break
			}
		}
	}

	return &tempopb.TraceSearchMetadata{
		TraceID:           util.TraceIDToHexString(id),
		RootServiceName:   rootServiceName,
		RootTraceName:     rootSpanName,
		StartTimeUnixNano: traceStart,
		DurationMs:        durationMs,
	}, nil
}

// todo: support more attribute types. currently only string is supported
func searchAttributes(tags map[string]string, atts []*v1common.KeyValue) bool {
	for _, a := range atts {
		var v string
		var ok bool

		if v, ok = tags[a.Key]; !ok {
			continue
		}

		if strings.Contains(a.Value.GetStringValue(), v) {
			return true
		}
	}

	return false
}
