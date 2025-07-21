---
title: Choose an instrumentation method
description: Learn about the different instrumentation methods for sending traces to Tempo.
weight: 300
draft: true
---

<!-- Hidden page. We need to add focus this page so it succinctly answers how to choose an instrumentation method. -->

# Choose an instrumentation method

You need to instrument your app to enable it to emit tracing data.
This data is then gathered by a collector and sent to Tempo.

## Instrumentation methods comparison

You can instrument your code using one or more of the methods described in the table.

| Instrumentation method     | Description                                                                                                   | Benefits                                                                                   | Drawbacks                                               |
| ------------------------- | ------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ | ------------------------------------------------------- |
| Automatic instrumentation  | Applies instrumentation automatically using agents or middleware, without code changes.                       |<ul><li>Quick setup: Enables tracing without code changes</li><li>Low overhead: Minimal performance impact</li></ul> | <ul><li>Limited customization: May not capture all use cases</li></ul>  |
| Zero-code instrumentation  | Uses eBPF technology to instrument applications without code changes.                                         | <ul><li>Non-intrusive: No code changes needed</li><li>High performance: Low overhead and efficient</li></ul> | <ul><li>Limited visibility: May not capture all behavior</li><li>Complexity: Requires eBPF knowledge</li></ul> |
| Manual instrumentation     | Involves adding code to create spans and traces, giving full control over collected data.                     | <ul><li>Full control: Define exactly what data is collected</li><li>Custom spans: Capture specific behavior</li></ul> | <ul><li>Higher effort: Requires code changes and maintenance</li><li>Potential for errors: Can introduce bugs</li></ul> |
| Hybrid instrumentation     | Combines automatic and manual methods, using automatic for most code and manual for custom tracing logic.     | <ul><li>Flexibility: Leverage benefits of both methods</li><ul>                                        | <ul><li> Complexity: May require managing both approaches</li><ul>      |


## Next steps

After you choose your instrumentation method, refer to [Set up instrumentation](/docs/tempo/<TEMPO_VERSION>/instrument-send/set-up-instrumentation/).