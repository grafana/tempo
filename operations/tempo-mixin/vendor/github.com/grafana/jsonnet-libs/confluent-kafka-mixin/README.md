# Confluent Kafka Mixin

This Mixin was designed to work with the Confluent Cloud metrics API. It contains a dashboard that monitors all metrics available on the API at the time of creation, giving the user a fine grained control on which metrics are seen, selecting the cluster, topic, partition and principalID wanted. This dashboard was based on [this community dashboard](https://github.com/Dabz/ccloudexporter/blob/master/grafana/ccloud-exporter.json).

To use it, you need to have `mixtool` and `jsonnetfmt` installed. If you have a working Go development environment, it's easiest to run the following:

```bash
$ go get github.com/monitoring-mixins/mixtool/cmd/mixtool
$ go get github.com/google/go-jsonnet/cmd/jsonnetfmt
```

You can then build a directory `dashboard_out` with the JSON dashboard files for Grafana:

```bash
$ make build
```

For more advanced uses of mixins, see [Prometheus Monitoring Mixins docs](https://github.com/monitoring-mixins/docs).
