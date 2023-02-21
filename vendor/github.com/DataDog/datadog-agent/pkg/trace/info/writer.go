// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package info

import (
	"encoding/json"

	"go.uber.org/atomic"
)

// TraceWriterInfo represents statistics from the trace writer.
type TraceWriterInfo struct {
	// all atomic values are included as values in this struct, to simplify
	// initialization of the type.  The atomic values _must_ occur first in the
	// struct.

	Payloads          atomic.Int64
	Traces            atomic.Int64
	Events            atomic.Int64
	Spans             atomic.Int64
	Errors            atomic.Int64
	Retries           atomic.Int64
	Bytes             atomic.Int64
	BytesUncompressed atomic.Int64
	SingleMaxSize     atomic.Int64
}

// StatsWriterInfo represents statistics from the stats writer.
type StatsWriterInfo struct {
	// all atomic values are included as values in this struct, to simplify
	// initialization of the type.  The atomic values _must_ occur first in the
	// struct.

	Payloads       atomic.Int64
	ClientPayloads atomic.Int64
	StatsBuckets   atomic.Int64
	StatsEntries   atomic.Int64
	Errors         atomic.Int64
	Retries        atomic.Int64
	Splits         atomic.Int64
	Bytes          atomic.Int64
}

// UpdateTraceWriterInfo updates internal trace writer stats
func UpdateTraceWriterInfo(tws TraceWriterInfo) {
	infoMu.Lock()
	defer infoMu.Unlock()
	traceWriterInfo = tws
}

func publishTraceWriterInfo() interface{} {
	infoMu.RLock()
	defer infoMu.RUnlock()
	return traceWriterInfo
}

// MarshalJSON implements encoding/json.MarshalJSON.
func (twi TraceWriterInfo) MarshalJSON() ([]byte, error) {
	asMap := map[string]float64{
		"Payloads":          float64(twi.Payloads.Load()),
		"Traces":            float64(twi.Traces.Load()),
		"Events":            float64(twi.Events.Load()),
		"Spans":             float64(twi.Spans.Load()),
		"Errors":            float64(twi.Errors.Load()),
		"Retries":           float64(twi.Retries.Load()),
		"Bytes":             float64(twi.Bytes.Load()),
		"BytesUncompressed": float64(twi.BytesUncompressed.Load()),
		"SingleMaxSize":     float64(twi.SingleMaxSize.Load()),
	}
	return json.Marshal(asMap)
}

// UpdateStatsWriterInfo updates internal stats writer stats
func UpdateStatsWriterInfo(sws StatsWriterInfo) {
	infoMu.Lock()
	defer infoMu.Unlock()
	statsWriterInfo = sws
}

func publishStatsWriterInfo() interface{} {
	infoMu.RLock()
	defer infoMu.RUnlock()
	return statsWriterInfo
}

// MarshalJSON implements encoding/json.MarshalJSON.
func (swi StatsWriterInfo) MarshalJSON() ([]byte, error) {
	asMap := map[string]float64{
		"Payloads":       float64(swi.Payloads.Load()),
		"ClientPayloads": float64(swi.ClientPayloads.Load()),
		"StatsBuckets":   float64(swi.StatsBuckets.Load()),
		"StatsEntries":   float64(swi.StatsEntries.Load()),
		"Errors":         float64(swi.Errors.Load()),
		"Retries":        float64(swi.Retries.Load()),
		"Splits":         float64(swi.Splits.Load()),
		"Bytes":          float64(swi.Bytes.Load()),
	}
	return json.Marshal(asMap)
}
