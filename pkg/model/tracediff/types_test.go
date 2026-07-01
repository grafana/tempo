package tracediff

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTracePatchV0JSONShape(t *testing.T) {
	idx := 0
	changes := Result{
		Version: VersionTracePatchV0,
		Base: TraceMeta{
			TraceID:   "trace-a",
			SpanCount: 5,
		},
		Compare: TraceMeta{
			TraceID:   "trace-b",
			SpanCount: 6,
		},
		Stats: Stats{
			SpanCountA:       5,
			SpanCountB:       6,
			MatchedSpans:     4,
			ModifiedSpans:    1,
			AddedSpans:       1,
			RemovedSpans:     1,
			FieldChanges:     2,
			AttributeChanges: 3,
			EventChanges:     0,
			Truncated:        false,
		},
		Modified: []ModifiedSpan{
			{
				Span: SpanRef{
					Path:    []int{0, 1},
					Service: "inventory",
					Name:    "reserve",
					Kind:    "client",
				},
				Changes: []Change{
					{
						Op: OperationModify,
						Target: Target{
							Type: TargetField,
							Name: "duration_nanos",
						},
						Before: int64(80),
						After:  int64(390),
					},
					{
						Op: OperationModify,
						Target: Target{
							Type: TargetField,
							Name: "status",
						},
						Before: "ok",
						After:  "error",
					},
					{
						Op: OperationAdd,
						Target: Target{
							Type:  TargetAttribute,
							Scope: "span",
							Key:   "error.type",
						},
						Before: nil,
						After:  "timeout",
					},
					{
						Op: OperationModify,
						Target: Target{
							Type:  TargetAttribute,
							Scope: "span",
							Key:   "http.request.header.accept",
						},
						Before: []any{"text/html", "application/json"},
						After:  []any{"application/json", "image/webp"},
					},
					{
						Op: OperationRemove,
						Target: Target{
							Type:  TargetAttribute,
							Scope: "span",
							Key:   "cache.hit",
						},
						Before: true,
						After:  nil,
					},
				},
			},
		},
		Added: []SpanChange{
			{
				Target: SpanTarget{
					Type:       TargetSpan,
					ParentPath: []int{0, 1},
					Index:      &idx,
				},
				Span: SpanSnapshot{
					Path:          []int{0, 1, 0},
					Service:       "inventory",
					Name:          "retry reserve",
					Kind:          "client",
					DurationNanos: 120,
					Status:        "ok",
				},
			},
		},
		Removed: []SpanChange{
			{
				Target: SpanTarget{
					Type: TargetSpan,
					Path: []int{0, 2},
				},
				Span: SpanSnapshot{
					Path:          []int{0, 2},
					Service:       "fraud",
					Name:          "check_risk",
					Kind:          "client",
					DurationNanos: 35,
					Status:        "ok",
				},
			},
		},
		Warnings: []Warning{
			{
				Code:    "low_match_confidence",
				Message: "some spans were not matched by structural path",
			},
		},
	}

	got, err := json.Marshal(changes)
	require.NoError(t, err)

	assert.JSONEq(t, `{
		"version": "trace-patch-v0",
		"base": {"traceId": "trace-a", "spanCount": 5},
		"compare": {"traceId": "trace-b", "spanCount": 6},
		"stats": {
			"spanCountA": 5,
			"spanCountB": 6,
			"matchedSpans": 4,
			"modifiedSpans": 1,
			"addedSpans": 1,
			"removedSpans": 1,
			"fieldChanges": 2,
			"attributeChanges": 3,
			"eventChanges": 0,
			"truncated": false
		},
		"modified": [
			{
				"span": {"path": [0, 1], "service": "inventory", "name": "reserve", "kind": "client"},
				"changes": [
					{"op": "modify", "target": {"type": "field", "name": "duration_nanos"}, "before": 80, "after": 390},
					{"op": "modify", "target": {"type": "field", "name": "status"}, "before": "ok", "after": "error"},
					{"op": "add", "target": {"type": "attribute", "scope": "span", "key": "error.type"}, "before": null, "after": "timeout"},
					{"op": "modify", "target": {"type": "attribute", "scope": "span", "key": "http.request.header.accept"}, "before": ["text/html", "application/json"], "after": ["application/json", "image/webp"]},
					{"op": "remove", "target": {"type": "attribute", "scope": "span", "key": "cache.hit"}, "before": true, "after": null}
				]
			}
		],
		"added": [
			{
				"target": {"type": "span", "parentPath": [0, 1], "index": 0},
				"span": {"path": [0, 1, 0], "service": "inventory", "name": "retry reserve", "kind": "client", "duration_nanos": 120, "status": "ok"}
			}
		],
		"removed": [
			{
				"target": {"type": "span", "path": [0, 2]},
				"span": {"path": [0, 2], "service": "fraud", "name": "check_risk", "kind": "client", "duration_nanos": 35, "status": "ok"}
			}
		],
		"warnings": [
			{"code": "low_match_confidence", "message": "some spans were not matched by structural path"}
		]
	}`, string(got))
}
