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
To learn more about instrumentation, refer to [About instrumentation](./about-instrumentation/).

Instrumentation is the act of modifying the source code of a service to emit span information tied to a common trace ID.
Traces themselves are a meta-object, comprised of nothing but spans that hold the same ID.

To instrument your application or service, you need to:

<!--Commented out - this page has draft:true in the frontmatter. 1. [Choose an instrumentation method to use with your application](./choose-instrumentation-method/)-->
1. [Set up instrumentation](./set-up-instrumentation/) to generate traces
1. [Set up a collector](./set-up-collector/) to receive traces from your application
