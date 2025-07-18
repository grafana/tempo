---
title: Instrument for distributed tracing
menuTitle: Instrument for tracing
description: Client instrumentation is the first building block to a functioning distributed tracing visualization pipeline.
aliases:
- ../guides/instrumentation/ # /docs/tempo/latest/guides/instrumentation/
- ../getting-started/instrumentation/ # /docs/tempo/latest/getting-started/instrumentation/
weight: 350
---

# Instrument for distributed tracing

Client instrumentation is the first building block to a functioning distributed tracing visualization pipeline.
Instrumentation handles how traces are generated.

Instrumentation is the act of modifying the source code of a service to emit span information tied to a common trace ID.
Traces themselves are a metaobject, comprised of nothing but spans that hold the same ID.

To generate and gather traces, you need to:

1. [Choose an instrumentation method to use with your application](./choose-instrumentation-method/)
1. [Set up instrumentation](./set-up-instrumentation/) to generate traces
1. [Set up a collector](./set-up-collector/) to receive traces from your application
