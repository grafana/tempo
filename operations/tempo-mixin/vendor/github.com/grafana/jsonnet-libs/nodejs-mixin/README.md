# Node.js Mixin

The Node.js Mixin is a set of configurable, reusable, and extensible alerts and dashboards based on the default metrics exported by the [prom-client](https://github.com/siimon/prom-client). The mixin creates alerting rules for Prometheus and suitable dashboard descriptions for Grafana.

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
