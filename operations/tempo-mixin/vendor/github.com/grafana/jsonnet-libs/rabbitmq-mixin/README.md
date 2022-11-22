# RabbitMQ Mixin

The RabbitMQ Mixin is a set of configurable, reusable, and extensible alerts and dashboards based on the default metrics exported by RabbitMQ's prometheus monitoring plugin, shipped natively as of version 3.8.0. The mixin creates alerting rules for Prometheus and suitable dashboard descriptions for Grafana.
For more information on the plugin please refer to [Monitoring with Prometheus & Grafana](https://www.rabbitmq.com/prometheus.html).

The dashboards were based on those available at RabbitMQ's Grafana organization profile. See [RabbitMQ Organization](https://grafana.com/orgs/rabbitmq/dashboards).

The alerts were based on those published at [https://awesome-prometheus-alerts.grep.to/rules#rabbitmq](https://awesome-prometheus-alerts.grep.to/rules#rabbitmq).

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
