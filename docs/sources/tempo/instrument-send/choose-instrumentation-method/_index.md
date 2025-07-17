---
title: Choose an instrumentation method
description: Learn about the different instrumentation methods for sending traces to Tempo.
weight: 500
---

# Choose an instrumentation method

You need to instrument your app to enable it to emit tracing data.
This data is then gathered by a collector and sent to Tempo.

You can instrument your code using one or more of these methods:

*  Automatic instrumentation
*  Zero-code instrumentation
*  Manual instrumentation
*  Hybrid instrumentation

## Instrument your code with little to no changes

You can instrument your code with little to no changes using automatic instrumentation, zero-code instrumentation, or hybrid instrumentation.

### Automatic instrumentation

Automatic instrumentation is a method where the instrumentation is applied automatically to your code without requiring any changes to the code itself. This is typically done using agents or middleware that intercept requests and collect tracing data.
This method is often used in environments where you want to quickly enable tracing without modifying the application code.

Benefits:
- Quick setup: Enables tracing without code changes.
- Low overhead: Minimal impact on application performance.

Drawbacks:
- Limited customization: May not capture all specific use cases.

### Zero-code instrumentation

Zero-code instrumentation is a method that uses eBPF technology to instrument your application without requiring any code changes. This approach is particularly useful for applications where modifying the code isn't feasible or desirable.

For more information, refer to  [Zero-code instrumentation](https://opentelemetry.io/docs/concepts/instrumentation/zero-code/) in the OpenTelemetry documentation.

Benefits:
- Non-intrusive: Does not require code changes, making it easy to implement.
- High performance: eBPF is designed for low overhead and high efficiency.

Drawbacks:
- Limited visibility: May not capture all application behavior.
- Complexity: Requires understanding of eBPF and its ecosystem.

## Customize your instrumentation

You can use manual instrumentation to retain complete control over the data your app or service emits.

### Manual instrumentation

Manual instrumentation involves adding code to your application to create spans and traces. This method provides the most control over what data is collected and how it is structured.
This approach is often used when you need to capture specific application behavior or when automatic instrumentation does not meet your needs.

Benefits:

- Full control: Allows you to define exactly what data is collected and how it is structured.
- Custom spans: You can create custom spans to capture specific application behavior.

Drawbacks:

- Higher effort: Requires code changes and ongoing maintenance.
- Potential for errors: Manual instrumentation can introduce bugs if not done carefully.

## Hybrid instrumentation

Hybrid instrumentation combines automatic and manual instrumentation. It allows you to use automatic instrumentation for most of your application while manually instrumenting specific parts of the code that require custom tracing logic.
This approach provides flexibility and allows you to leverage the benefits of both automatic and manual instrumentation.

## Next steps

After you choose your instrumentation method, refer to [Set up instrumentation](/docs/tempo/<TEMPO_VERSION>/instrument-send/set-up-instrumentation/) for the next steps.