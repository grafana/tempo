<p align="center"><img src="docs/sources/tempo/logo_and_name.png" alt="Tempo Logo"></p>
<p align="center">
  <a href="https://github.com/grafana/tempo/releases"><img src="https://img.shields.io/github/v/release/grafana/tempo?display_name=tag&sort=semver" alt="Latest Release"/></a>
  <img src="https://img.shields.io/github/license/grafana/tempo" alt="License" />
  <a href="https://hub.docker.com/r/grafana/tempo/tags"><image src="https://img.shields.io/docker/pulls/grafana/tempo" alt="Docker Pulls"/></a>
  <a href="https://grafana.slack.com/archives/C01D981PEE5"><img src="https://img.shields.io/badge/join%20slack-%23tempo-brightgreen.svg" alt="Slack" /></a>
  <a href="https://community.grafana.com/c/grafana-tempo/40"><img src="https://img.shields.io/badge/discuss-tempo%20forum-orange.svg" alt="Community Forum" /></a>
  <a href="https://goreportcard.com/report/github.com/grafana/tempo"><img src="https://goreportcard.com/badge/github.com/grafana/tempo" alt="Go Report Card" /></a>
</p>

Grafana Tempo is an open source, easy-to-use and high-scale distributed tracing backend. Tempo is cost-efficient, requiring only object storage to operate, and is deeply integrated with Grafana, Prometheus, and Loki.

Tempo is Jaeger, Zipkin, Kafka, OpenCensus and OpenTelemetry compatible.  It ingests batches in any of the mentioned formats, buffers them and then writes them to Azure, GCS, S3 or local disk. As such, it is robust, cheap and easy to operate!

Tempo implements [TraceQL](https://grafana.com/docs/tempo/latest/traceql/), a traces-first query language inspired by LogQL and PromQL. This query language allows users to very precisely and easily select spans and jump directly to the spans fulfilling the specified conditions:

![Tempo data source query editor](https://grafana.com/media/docs/grafana/data-sources/tempo/query-editor/tempo-ds-query-ed-example-v11-a.png)

## Getting started

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
