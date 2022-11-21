# Python Runtime Mixin

_This mixin is a work in progress. We aim for it to become a good role model for
dashboards eventually, but it's not there yet._

Mixins are a collection of configurable, reusable Prometheus rules, alerts
and/or Grafana dashboards for a particular system, usually created by experts
in that system. By applying them to Prometheus and Grafana, you can quickly
set up appropriate monitoring for your systems.

This mixin is for Python applications, and contains a dashboard for visualizing the
runtime metrics produced by [Prometheus Python client](https://github.com/prometheus/client_python) process and platform collectors.

To use the mixin, you need to have `mixtool` and `jsonnetfmt` installed. If you
have a working Go development environment, it's easiest to run the following:
```bash
$ go get github.com/monitoring-mixins/mixtool/cmd/mixtool
$ go get github.com/google/go-jsonnet/cmd/jsonnetfmt
```

For more advanced uses of mixins, see
https://github.com/monitoring-mixins/docs.
