---
title: Choose an instrumentation method
description: Learn about the different instrumentation methods for sending traces to Tempo.
weight: 500
---

# Choose an instrumentation method

You need to instrument your app to enable it to emit tracing data.
This data is then gathered by a collector and sent to Tempo.

## Instrumentation methods

To add instrumentation, the code for a service uses a Software Development Kit (SDK) which supplies language-specific libraries that allow the:

* Creation of a new trace, starting with a new root span.
* Addition of new spans, that are siblings of children of pre-existing spans.
* Addition of span attributes to add contextual information to each span, as well as span links and events.
* Closure of spans when a unit of work is complete.

Adding code to carry out these operations is known as **manual instrumentation**, as it requires manual intervention by an engineer to write code to deal with traces, as well as to determine where in the code traces/spans should start, the attributes and other data that should be attached to spans, and where traces/spans should end.

There is an alternative/companion to manual instrumentation, **auto-instrumentation**. Auto-instrumentation is the act of allowing a tracing SDK to determine where traces/spans should start, what information should be added to spans, and where traces/spans should stop. Essentially, manual instrumentation is pre-packaged for a large number of popular frameworks and libraries which are used inside a service's code, and it is these libraries/frameworks that are actually emitting spans for a trace.

These libraries usually include those dealing with networking, so for example a request coming into a service might be via an auto-instrumented HTTP library, which would then start a trace until it sent a response back via HTTP to the requester. Along the course of the request, the service might use other libraries that process data, and if they are also auto-instrumented then new spans will be generated for the trace that include suitable attributes.

This produces a trace that can be very useful, but ultimately will not include any spans that are specific to service code that isn't executing library commands. Because of this, the best traced services are usually those that include a mixture of auto and manual instrumentation.

Traces usually exist in relation to a request made to an application, and spans are generated when new units of work occur whilst processing that request. A trace usually ends when a response to the request is sent.

The [OpenTelemetry SDK](https://opentelemetry.io/docs/instrumentation/) is Grafana's recommended solution for instrumenting, and it supports a myriad of modern languages, detailed in the linked page.

Additionally, instrumentation SDKs usually emit span information to a local receiver/collector. Examples of these are Grafana Alloy and the OpenTelemetry Collector. These receivers can then carry out additional processing on the traces (such as tail sampling, span batching, process filtering, etc.) to modify a trace and its spans before they're sent to a final destination, usually a tracing backend like Grafana Tempo.

### Propagation

Modern applications are not usually not the monolithic applications of old, where single binaries contained all of the code, and multiple instances of the monoliths were run to 'scale' to increased traffic. Modern applications are built using a large number of microservices, each one specifically designed to handle specific portions of a request/data flow.

Traces are (usually) tied to a request/response model, being initiated when a request is received by application and ending when a response is returned. Each request could flow through a number of different services (eg. an API service, a caching service, a processing service, a database service, etc.), and therefore to ensure that the trace includes every service that the data is routed through there needs to be some way of sending the details about a trace with the data that's sent to each service.

This is called trace propagation, and the actual mechanics of it alter depending on the way that data is passed around services. The majority of services in the modern application world use REST (via HTTP) or protobufs (via gRPC, which is itself handled by HTTP/2). Because a trace is just a metaobject comprised of spans, all that needs to happen to continue a trace from one service to another is the addition of a trace ID and the last current span ID to the data send to a downstream service.

![Distributed Trace Propagation](1.5%20-%20TracePropagation.png)

Many instrumentation SDKs, include propagators that can handle a number of different transport protocols, and when matched with appropriate auto-instrumentation mean that an engineer doesn't even have to consider writing any code to inject the downstream data with the trace information, it gets handled automatically (for example in the case of HTTP, headers are included that specifically deal with trace information that gets extracted by the Opentelementry SDK in the downstream service to carry on a trace).

The [W3C Trace Context](https://www.w3.org/TR/trace-context/) is the default way of sending propagation information in OpenTelemetry SDKs.
However, the SDK also allows you to provide custom propagators to it instead, which means that if an application uses a custom protocol or networking scheme, engineers can write propagators and extractors to handle these as well.

![Customer Trace Propagation](1.5%20-%20CustomPropagation.png)

## Instrumentation methods comparision

You can instrument your code using one or more of the methods described in the table.

| Instrumentation method     | Description                                                                                                   | Benefits                                                                                   | Drawbacks                                               |
| ------------------------- | ------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ | ------------------------------------------------------- |
| Automatic instrumentation  | Applies instrumentation automatically using agents or middleware, without code changes.                       |<ul><li>Quick setup: Enables tracing without code changes</li><li>Low overhead: Minimal performance impact</li></ul> | <ul><li>Limited customization: May not capture all use cases</li></ul>  |
| Zero-code instrumentation  | Uses eBPF technology to instrument applications without code changes.                                         | <ul><li>Non-intrusive: No code changes needed</li><li>High performance: Low overhead and efficient</li></ul> | <ul><li>Limited visibility: May not capture all behavior</li><li>Complexity: Requires eBPF knowledge</li></ul> |
| Manual instrumentation     | Involves adding code to create spans and traces, giving full control over collected data.                     | <ul><li>Full control: Define exactly what data is collected</li><li>Custom spans: Capture specific behavior</li></ul> | <ul><li>Higher effort: Requires code changes and maintenance</li><li>Potential for errors: Can introduce bugs</li></ul> |
| Hybrid instrumentation     | Combines automatic and manual methods, using automatic for most code and manual for custom tracing logic.     | <ul><li>Flexibility: Leverage benefits of both methods</li><ul>                                        | <ul><li> Complexity: May require managing both approaches</li><ul>      |



## Next steps

After you choose your instrumentation method, refer to [Set up instrumentation](/docs/tempo/<TEMPO_VERSION>/instrument-send/set-up-instrumentation/).