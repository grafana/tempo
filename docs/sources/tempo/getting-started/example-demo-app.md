---
title: Example setups
aliases:
- /docs/tempo/latest/getting-started/quickstart-tempo/
- /docs/tempo/latest/guides/loki-derived-fields/
weight: 300
---

# Example setups

The following examples show various deployment and configuration options using trace generators so you can get started experimenting with Tempo without an existing application.

For more information about Tempo setup and configuration, see:

* [Set up a Tempo cluster]({{< relref "../setup">}})
* [Tempo configuration]({{< relref "../configuration" >}})

If you are interested in instrumentation, see [Tempo instrumentation]({{< relref "instrumentation" >}}).
## Docker Compose

The [docker-compose examples](https://github.com/grafana/tempo/tree/main/example/docker-compose) are simpler and designed to show minimal configuration.

Some of the examples include:

- Trace discovery with Loki
- Basic Grafana Agent/OpenTelemetry Setup
- Various Backends (S3/GCS/Azure)
- [K6 with Traces]({{< relref "docker-example" >}})

This is a great place to get started with Tempo and learn about various trace discovery flows.

## Tanka

To view an example of a complete microservice-based deployment, this [Jsonnet based example](https://github.com/grafana/tempo/tree/main/example/tk) shows a complete microservice based deployment.
There are monolithic mode and microservices examples.

To learn how to set up a Tempo cluster, see [Deploy on Kubernetes with Tanka]({{< relref "../setup/tanka" >}}).

## Helm

The Helm [example](https://github.com/grafana/tempo/tree/main/example/helm) shows a complete microservice based deployment.
There are monolithic mode and microservices examples.

To install Tempo on Kubernetes, use the [Deploy on Kubernetes using Helm](/docs/helm-charts/tempo-distributed/next/) procedure.

## The New Stack demo

The [New Stack (TNS) demo](https://github.com/grafana/tns) demonstrates a fully instrumented three-tier application and the integration of Grafana, Prometheus, Loki, and Tempo [features](https://github.com/grafana/tns#demoable-things), including metrics to traces (exemplars), logs to traces, and traces to logs.

To learn how to set up a TNS app, see [Set up a test application for a Tempo cluster]({{< relref "../setup/set-up-test-app" >}}).

A good place to start is the [docker-compose setup](https://github.com/grafana/tns/tree/main/production/docker-compose) which includes a pre-built dashboard, load generator, and exemplars.

Explanation:
- Metrics To Traces (Exemplars)
  - The weaveworks middleware automatically [records](https://github.com/weaveworks/common/blob/bd288de53d57de300fa286688ce2fc935687213f/middleware/instrument.go#L79) request latency with an exemplar.  Try running the following PromQL query in Grafana `Explore` and enabling the exemplars switch. It shows the p50 request latency for the "app" container:  `histogram_quantile(0.5, sum(rate(tns_request_duration_seconds_bucket{job="tns/app"}[$__rate_interval])) by (le))`.  Click the exemplar to see the trace.
- LogqlV2 and Logs to Traces
  - The http client [logs inter-service http requests](https://github.com/grafana/tns/blob/main/client/http.go#L70) in `logfmt` format, which enables the ability to perform complex queries over api traffic. Try running the following query which shows all failed api requests from app to db and took longer than 100ms: `{job="tns/app"} | logfmt | level="info" and status>=500 and status <=599 and duration > 100ms`.  Expand the log line and click the Tempo button near the trace ID to see the trace.
- Traces To Logs
  - When viewing only a trace in the Explore view (i.e. not side-by-side with logs), the Logs icon will appear next to each span.  Click it to view the matching logs.
- Status
  - Exemplar support in Prometheus is still pre-release so a custom image is used, and the feature is enabled with the `--enable-feature=exemplar-storage` command line parameter.
