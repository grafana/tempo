# Clickhouse Mixin

Clickhouse mixin is a set of configurable, reusable and extensible alerts and dashboards that uses the [Clickhouse Exporter](https://github.com/ClickHouse/clickhouse_exporter) for Prometheus and Loki for logs (optional).

The Clickhouse mixin includes the following dashboards:
- Clickhouse Overview
- Clickhouse Latency
- Clickhouse Replica

## Clickhouse Overview:

The Clickhouse Overview dashboard provides details on queries, memory usage, networking and error logs. To get Clickhouse error logs, [Promtail and Loki needs to be installed](https://grafana.com/docs/loki/latest/installation/) and provisioned for logs with your Grafana instance. The default Clickhouse error log path is `/var/log/clickhouse-server/clickhouse-server.err.log`.

![First screenshot of Clickhouse Overview Dashboard](https://storage.googleapis.com/grafanalabs-integration-assets/clickhouse/screenshots/clickhouse-overview.01.png)
![Second screenshot of Clickhouse Overview Dashboard](https://storage.googleapis.com/grafanalabs-integration-assets/clickhouse/screenshots/clickhouse-overview.02.png)


Clickhouse error logs are enabled by default in the `config.libsonnet` and can be removed by setting `enableLokiLogs` to `false`. Then run `make` again to regenerate the dashboard:

```
{
  _config+:: {
    enableLokiLogs: true,
  },
}
```

## Clickhouse Latency:

The Clickhouse Latency dashboard provides details on latency metrics.
![Third screenshot of Clickhouse Latency Dashboard](https://storage.googleapis.com/grafanalabs-integration-assets/clickhouse/screenshots/clickhouse-latency.01.png)

## Clickhouse Replica:

The Clickhouse Latency dashboard provides details on replica metrics.
![Fourth screenshot of Clickhouse Replica Dashboard](https://storage.googleapis.com/grafanalabs-integration-assets/clickhouse/screenshots/clickhouse-replica.01.png)

## Install tools

```bash
go install github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@latest
go install github.com/monitoring-mixins/mixtool/cmd/mixtool@latest
```

For linting and formatting, you would also need and `jsonnetfmt` installed. If you
have a working Go development environment, it's easiest to run the following:

```bash
go install github.com/google/go-jsonnet/cmd/jsonnetfmt@latest
```

The files in `dashboards_out` need to be imported
into your Grafana server.  The exact details will be depending on your environment.

`prometheus_alerts.yaml` needs to be imported into Prometheus.

## Generate dashboards and alerts

Edit `config.libsonnet` if required and then build JSON dashboard files for Grafana:

```bash
make
```

For more advanced uses of mixins, see
https://github.com/monitoring-mixins/docs.
