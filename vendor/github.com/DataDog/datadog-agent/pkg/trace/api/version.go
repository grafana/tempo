// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package api

// Version is a dumb way to version our collector handlers
type Version string

const (
	// v01 is DEPRECATED
	v01 Version = "v0.1"

	// v02 is DEPRECATED
	v02 Version = "v0.2"

	// v03
	// Traces: msgpack/JSON (Content-Type) slice of traces
	v03 Version = "v0.3"

	// v04
	//
	// Request: Trace chunks.
	// 	Content-Type: application/msgpack
	// 	Payload: An array of arrays of Span (pkg/trace/pb/span.proto)
	//
	// Response: Service sampling rates.
	// 	Content-Type: application/json
	// 	Payload: Object mapping span pattern to sample rate.
	//
	// The response payload is an object whose keys are of the form
	// "service:my-service,env:my-env", where "my-service" is the name of the
	// affected service, and "my-env" is the name of the relevant deployment
	// environment. The value at each key is the sample rate to apply to traces
	// beginning with a span that matches the key.
	//
	//  {
	//    "service:foosvc,env:prod": 0.223443,
	//    "service:barsvc,env:staging": 0.011249
	//  }
	//
	// There is a special key, "service:,env:", that denotes the sample rate for
	// traces that do not match any other key.
	//
	//  {
	//    "service:foosvc,env:prod": 0.223443,
	//    "service:barsvc,env:staging": 0.011249,
	//    "service:,env:": 0.5
	//  }
	//
	// Neither the "service:,env:" key nor any other need be present in the
	// response.
	//
	//  {}
	//
	v04 Version = "v0.4"

	// v05
	//
	// Request: Trace chunks with a shared dictionary of strings.
	// 	Content-Type: application/msgpack
	// 	Payload: Traces with strings de-duplicated into a dictionary (see below).
	//
	// Response: Service sampling rates (see description in v04).
	//
	// The request payload is an array containing exactly 2 elements:
	//
	// 	1. An array of all unique strings present in the payload (a dictionary referred to by index).
	// 	2. An array of traces, where each trace is an array of spans. A span is encoded as an array having
	// 	   exactly 12 elements, representing all span properties, in this exact order:
	//
	// 		 0: Service   (uint32)
	// 		 1: Name      (uint32)
	// 		 2: Resource  (uint32)
	// 		 3: TraceID   (uint64)
	// 		 4: SpanID    (uint64)
	// 		 5: ParentID  (uint64)
	// 		 6: Start     (int64)
	// 		 7: Duration  (int64)
	// 		 8: Error     (int32)
	// 		 9: Meta      (map[uint32]uint32)
	// 		10: Metrics   (map[uint32]float64)
	// 		11: Type      (uint32)
	//
	// 	Considerations:
	//
	// 	- The "uint32" typed values in "Service", "Name", "Resource", "Type", "Meta" and "Metrics" represent
	// 	  the index at which the corresponding string is found in the dictionary. If any of the values are the
	// 	  empty string, then the empty string must be added into the dictionary.
	//
	// 	- None of the elements can be nil. If any of them are unset, they should be given their "zero-value". Here
	// 	  is an example of a span with all unset values:
	//
	// 		 0: 0                    // Service is "" (index 0 in dictionary)
	// 		 1: 0                    // Name is ""
	// 		 2: 0                    // Resource is ""
	// 		 3: 0                    // TraceID
	// 		 4: 0                    // SpanID
	// 		 5: 0                    // ParentID
	// 		 6: 0                    // Start
	// 		 7: 0                    // Duration
	// 		 8: 0                    // Error
	// 		 9: map[uint32]uint32{}  // Meta (empty map)
	// 		10: map[uint32]float64{} // Metrics (empty map)
	// 		11: 0                    // Type is ""
	//
	// 		The dictionary in this case would be []string{""}, having only the empty string at index 0.
	//
	v05 Version = "v0.5"

	// V07 API
	//
	// Request: Tracer Payload.
	// 	Content-Type: application/msgpack
	// 	Payload: TracerPayload (pkg/trace/pb/tracer_payload.proto)
	//
	// Response: Service sampling rates (see description in v04).
	//
	V07 Version = "v0.7"
)
