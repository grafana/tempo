// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// package header defines HTTP headers known convention used by the Trace Agent and Datadog's APM intake.
package header

const (
	// TraceCount is the header client implementation should fill
	// with the number of traces contained in the payload.
	TraceCount = "X-Datadog-Trace-Count"

	// ContainerID specifies the name of the header which contains the ID of the
	// container where the request originated.
	ContainerID = "Datadog-Container-ID"

	// Lang specifies the name of the header which contains the language from
	// which the traces originate.
	Lang = "Datadog-Meta-Lang"

	// LangVersion specifies the name of the header which contains the origin
	// language's version.
	LangVersion = "Datadog-Meta-Lang-Version"

	// LangInterpreter specifies the name of the HTTP header containing information
	// about the language interpreter, where applicable.
	LangInterpreter = "Datadog-Meta-Lang-Interpreter"

	// LangInterpreterVendor specifies the name of the HTTP header containing information
	// about the language interpreter vendor, where applicable.
	LangInterpreterVendor = "Datadog-Meta-Lang-Interpreter-Vendor"

	// TracerVersion specifies the name of the header which contains the version
	// of the tracer sending the payload.
	TracerVersion = "Datadog-Meta-Tracer-Version"

	// ComputedTopLevel specifies that the client has marked top-level spans, when set.
	// Any non-empty value will mean 'yes'.
	ComputedTopLevel = "Datadog-Client-Computed-Top-Level"

	// ComputedStats specifies whether the client has computed stats so that the agent
	// doesn't have to.
	ComputedStats = "Datadog-Client-Computed-Stats"

	// DroppedP0Traces contains the number of P0 trace chunks dropped by the client.
	// This value is used to adjust priority rates computed by the agent.
	DroppedP0Traces = "Datadog-Client-Dropped-P0-Traces"

	// DroppedP0Spans contains the number of P0 spans dropped by the client.
	// This value is used for metrics and could be used in the future to adjust priority rates.
	DroppedP0Spans = "Datadog-Client-Dropped-P0-Spans"

	// RatesPayloadVersion contains the version of sampling rates.
	// If both agent and client have the same version, the agent won't return rates in API response.
	RatesPayloadVersion = "Datadog-Rates-Payload-Version"
)
