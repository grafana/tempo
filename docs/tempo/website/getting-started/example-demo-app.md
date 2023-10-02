---
aliases:
- /docs/tempo/v1.2.1/getting-started/example-demo-app/
- /docs/tempo/v1.2.x/getting-started/quickstart-tempo/
- /docs/tempo/v1.2.x/guides/loki-derived-fields/
title: Example Setups
weight: 200
---

# Running the Tempo Backend

These examples show various deployment and [configuration]({{< relref "../configuration" >}}) options. They include trace
generators so an existing application is not necessary to get started experimenting with Tempo. If you are interested in
instrumentation please check out these [examples]({{< relref "./instrumentation" >}}).

> **Note:** You can use [Grafana Cloud](https://grafana.com/products/cloud/features/#cloud-dashboards-grafana) to avoid installing, maintaining, and scaling your own instance of Grafana Tempo. The free forever plan includes 50GB of free traces. [Create an account to get started](https://grafana.com/auth/sign-up/create-user?pg=docs-tempo&plcmt=in-text).

## Docker Compose

The [docker-compose](https://github.com/grafana/tempo/tree/main/example/docker-compose) examples are simpler and designed to show minimal configuration.  This is a great place
to get started with Tempo and learn about various trace discovery flows.

Check the above link for all kinds of examples including:
- Trace discovery with Loki
- Basic Grafana Agent/OpenTelemetry Setup
- Various Backends (S3/GCS/Azure)

## Tanka

The Jsonnet based [example](https://github.com/grafana/tempo/tree/main/example/tk) shows a complete microservice based deployment. 
There are single binary and microservices examples.

## Helm

The Helm [example](https://github.com/grafana/tempo/tree/main/example/helm) shows a complete microservice based deployment. 
There are single binary and microservices examples.

## The New Stack (TNS) Demo

The [TNS demo](https://github.com/grafana/tns) demonstrates a fully-instrumented three-tier application and the integration of Grafana, Prometheus, Loki, and Tempo [features](https://github.com/grafana/tns#demoable-things), including metrics to traces (exemplars), logs to traces, and traces to logs.  A good place to start is the [docker-compose setup](https://github.com/grafana/tns/tree/main/production/docker-compose) which includes a pre-built dashboard, load generator, and exemplars.

Explanation:
- Metrics To Traces (Exemplars)
  - The weaveworks middleware automatically [records](https://github.com/weaveworks/common/blob/bd288de53d57de300fa286688ce2fc935687213f/middleware/instrument.go#L79) request latency with an exemplar.  Try running the following PromQL query in Grafana `Explore` and enabling the exemplars switch. It shows the p50 request latency for the "app" container:  `histogram_quantile(0.5, sum(rate(tns_request_duration_seconds_bucket{job="tns/app"}[$__rate_interval])) by (le))`.  Click the exemplar to see the trace.
- LogqlV2 and Logs to Traces
  - The http client [logs inter-service http requests](https://github.com/grafana/tns/blob/main/client/http.go#L70) in `logfmt` format, which enables the ability to perform complex queries over api traffic. Try running the following query which shows all failed api requests from app to db and took longer than 100ms: `{job="tns/app"} | logfmt | level="info" and status>=500 and status <=599 and duration > 100ms`.  Expand the log line and click the Tempo button near the trace ID to see the trace.
- Traces To Logs
  - When viewing only a trace in the Explore view (i.e. not side-by-side with logs), the Logs icon will appear next to each span.  Click it to view the matching logs.
- Status
  - Exemplar support in Prometheus is still pre-release so a custom image is used, and the feature is enabled with the `--enable-feature=exemplar-storage` command line parameter.
