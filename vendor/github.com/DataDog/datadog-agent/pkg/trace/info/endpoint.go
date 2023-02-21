// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package info

// EndpointStats contains stats about the volume of data written
type EndpointStats struct {
	// TracesPayload is the number of traces payload sent, including errors.
	// If several URLs are given, each URL counts for one.
	TracesPayload int64
	// TracesPayloadError is the number of traces payload sent with an error.
	// If several URLs are given, each URL counts for one.
	TracesPayloadError int64
	// TracesBytes is the size of the traces payload data sent, including errors.
	// If several URLs are given, it does not change the size (shared for all).
	// This is the raw data, encoded, compressed.
	TracesBytes int64
	// TracesStats is the number of stats in the traces payload data sent, including errors.
	// If several URLs are given, it does not change the size (shared for all).
	TracesStats int64
}
