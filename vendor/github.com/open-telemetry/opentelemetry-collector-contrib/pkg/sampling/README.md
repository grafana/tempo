# pkg/sampling

## Overview

This package contains utilities for parsing and interpreting the W3C
[TraceState](https://www.w3.org/TR/trace-context/#tracestate-header)
and all sampling-relevant fields specified by OpenTelemetry that may
be found in the OpenTelemetry section of the W3C TraceState.

See the associated OpenTelemetry specifications:

- [Consistent probability sampling](https://opentelemetry.io/docs/specs/otel/trace/tracestate-probability-sampling/)
- [OpenTelemetry TraceState handling](https://opentelemetry.io/docs/specs/otel/trace/tracestate-handling/#pre-defined-opentelemetry-sub-keys)

This package supports sampler components that apply sampling on the
collection path through reading and writing the OpenTelemetry (`ot`)
TraceState sub-keys:

- `th`: the Threshold used to determine whether a TraceID is sampled
- `rv`: an explicit randomness value, which overrides randomness in the TraceID

See
[probabilisticsamplerprocessor](../../processor/probabilisticsamplerprocessor/README.md)
for an example application.
