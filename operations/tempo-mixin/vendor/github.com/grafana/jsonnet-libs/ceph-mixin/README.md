# Ceph Mixin

The Ceph Mixin is a set of configurable, reusable, and extensible alerts and dashboards based on the default metrics exported by Ceph`s prometheus plugin. The mixin creates alerting rules for Prometheus and suitable dashboard descriptions for Grafana.
For more information on the plugin please refer to the [Prometheus plugin page](https://docs.ceph.com/en/mimic/mgr/prometheus/).

The dashboard was taken from the community: [ID 2842](https://grafana.com/grafana/dashboards/2842).

The alerts were based on the information published at [https://sysdig.com/blog/monitoring-ceph-prometheus/](https://sysdig.com/blog/monitoring-ceph-prometheus/).

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
