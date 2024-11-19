<p align="center"><img src="docs/sources/tempo/logo_and_name.png" alt="Tempo Logo"></p>
<p align="center">
  <a href="https://github.com/grafana/tempo/releases"><img src="https://img.shields.io/github/v/release/grafana/tempo?display_name=tag&sort=semver" alt="Latest Release"/></a>
  <img src="https://img.shields.io/github/license/grafana/tempo" alt="License" />
  <a href="https://hub.docker.com/r/grafana/tempo/tags"><image src="https://img.shields.io/docker/pulls/grafana/tempo" alt="Docker Pulls"/></a>
  <a href="https://grafana.slack.com/archives/C01D981PEE5"><img src="https://img.shields.io/badge/join%20slack-%23tempo-brightgreen.svg" alt="Slack" /></a>
  <a href="https://community.grafana.com/c/grafana-tempo/40"><img src="https://img.shields.io/badge/discuss-tempo%20forum-orange.svg" alt="Community Forum" /></a>
  <a href="https://goreportcard.com/report/github.com/grafana/tempo"><img src="https://goreportcard.com/badge/github.com/grafana/tempo" alt="Go Report Card" /></a>
</p>

Grafana Tempo is an open source, easy-to-use, and high-scale distributed tracing backend. Tempo is cost-efficient, requiring only object storage to operate, and is deeply integrated with Grafana, Prometheus, and Loki. 


## Business value of distributed tracing

Distributed tracing helps teams quickly pinpoint performance issues and understand the flow of requests across services. The Explore Traces UI simplifies this process by offering a user-friendly interface to view and analyze trace data, making it easier to identify and resolve issues without needing to write complex queries.

Refer to [Use traces to find solutions](https://grafana.com/docs/tempo/latest/introduction/solutions-with-traces/)t o learn more about how you can use distributed tracing to investigate and solve issues. 

## Explore Traces UI: A better way to get value from your tracing data
We are excited to introduce the [Explore Traces app](https://github.com/grafana/explore-traces) as part of the Grafana Explore suite. This app provides a queryless and intuitive experience for analyzing tracing data, allowing teams to quickly identify performance issues, latency bottlenecks, and errors without needing to write complex queries or use TraceQL.

Key Features:
- **Intuitive Trace Analysis**: Spot slow or error-prone traces with easy, point-and-click interactions.
- **RED Metrics Overview**: Use Rate, Errors, and Duration metrics to highlight performance issues.
- **Automated Comparison**: Identify problematic attributes with automatic trace comparison.
- **Simplified Visualizations**: Access rich visual data without needing to construct TraceQL queries.

![image](https://github.com/user-attachments/assets/991205df-1b27-489f-8ef0-1a05ee158996)

To learn more see the following links:
- [Explore Traces repo](https://github.com/grafana/explore-traces)
- [Explore Traces documentation](https://grafana.com/docs/grafana/latest/explore/simplified-exploration/traces/)
- [Demo video](https://github.com/user-attachments/assets/8103e173-6dcf-4659-b938-7614c8a5b52d
)

## TraceQL

Tempo implements [TraceQL](https://grafana.com/docs/tempo/latest/traceql/), a traces-first query language inspired by LogQL and PromQL, which enables targeted queries or rich UI-driven analyses. 

### TraceQL metrics 

[TraceQL metrics](https://grafana.com/docs/tempo/latest/traceql/metrics-queries/) is an experimental feature in Grafana Tempo that creates metrics from traces. Metric queries extend trace queries by applying a function to trace query results. This powerful feature allows for ad hoc aggregation of any existing TraceQL query by any dimension available in your traces, much in the same way that LogQL metric queries create metrics from logs. 

Tempo is Jaeger, Zipkin, Kafka, OpenCensus, and OpenTelemetry compatible. It ingests batches in any of the mentioned formats, buffers them, and then writes them to Azure, GCS, S3, or local disk. As such, it is robust, cheap, and easy to operate!

## Getting started with Tempo

- [Get started documentation](https://grafana.com/docs/tempo/latest/getting-started/)
- [Deployment Examples](./example)
  - [Docker Compose](./example/docker-compose)
  - [Helm](./example/helm)
  - [Jsonnet](./example/tk)

## Further reading

To learn more about Tempo, consult the following documents & talks:

- [New in Grafana Tempo 2.0: Apache Parquet as the default storage format, support for TraceQL][tempo_20_announce]
- [Get to know TraceQL: A powerful new query language for distributed tracing][traceql-post]

[tempo_20_announce]: https://grafana.com/blog/2023/02/01/new-in-grafana-tempo-2.0-apache-parquet-as-the-default-storage-format-support-for-traceql/
[traceql-post]: https://grafana.com/blog/2023/02/07/get-to-know-traceql-a-powerful-new-query-language-for-distributed-tracing/

## Getting help

If you have any questions or feedback regarding Tempo:

- Grafana Labs hosts a [forum](https://community.grafana.com/c/grafana-tempo/40) for Tempo. This is a great place to post questions and search for answers.
- Ask a question on the [Tempo Slack channel](https://grafana.slack.com/archives/C01D981PEE5).
- [File an issue](https://github.com/grafana/tempo/issues/new/choose) for bugs, issues and feature suggestions.
- UI issues should be filed with [Grafana](https://github.com/grafana/grafana/issues/new/choose).

## OpenTelemetry

Tempo's receiver layer, wire format and storage format are all based directly on [standards](https://github.com/open-telemetry/opentelemetry-proto) and [code](https://github.com/open-telemetry/opentelemetry-collector) established by [OpenTelemetry](https://opentelemetry.io/).  We support open standards at Grafana!

Check out the [Integration Guides](https://grafana.com/docs/tempo/latest/guides/instrumentation/) to see examples of OpenTelemetry instrumentation with Tempo.

## Other components

### tempo-vulture
[tempo-vulture](https://github.com/grafana/tempo/tree/main/cmd/tempo-vulture) is Tempo's bird themed consistency checking tool.  It writes traces to Tempo and then queries them back in a variety of ways.

### tempo-cli
[tempo-cli](https://github.com/grafana/tempo/tree/main/cmd/tempo-cli) is the place to put any utility functionality related to Tempo. See [Documentation](https://grafana.com/docs/tempo/latest/operations/tempo_cli/) for more info.

## License

Grafana Tempo is distributed under [AGPL-3.0-only](LICENSE). For Apache-2.0 exceptions, see [LICENSING.md](LICENSING.md).
