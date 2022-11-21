# MongoDB Mixin

The MongoDB Mixin is a set of configurable, reusable, and extensible alerts and dashboards based on the metrics exported by [Percona MongoDB Exporter](https://github.com/percona/mongodb_exporter).

This mixin includes 3 dashboards suited for MongoDB, namely MongoDB Cluster, MongoDB Instance and MongoDB ReplicaSet.

The alerts were based on those published at [https://awesome-prometheus-alerts.grep.to/rules.html#mongodb](https://awesome-prometheus-alerts.grep.to/rules.html#mongodb).

To use them, you need to have `mixtool` and `jsonnetfmt` installed. If you have a working Go development environment, it's easiest to run the following:

```bash
$ go get github.com/monitoring-mixins/mixtool/cmd/mixtool
$ go get github.com/google/go-jsonnet/cmd/jsonnetfmt
```

You can then build the Prometheus rules file `alerts.yaml` and a directory `dashboard_out` with the JSON dashboard files for Grafana:

```bash
$ make build
```

For more advanced uses of mixins, see [Prometheus Monitoring Mixins docs](https://github.com/monitoring-mixins/docs).
