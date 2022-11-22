# Asterisk Mixin

The Asterisk Mixin is a set of configurable, reusable, and extensible alerts and
dashboards based on the metrics exported by the [prometheus exporter in Asterisk](https://wiki.asterisk.org/wiki/display/AST/Asterisk+18+Configuration_res_prometheus). The mixin also creates a dashboard that
uses Loki to monitor Asterisk logs.

The mixin contains 2 dashboards:

## Asterisk Overview
This dashboard gives a general overview of the Asterisk instance based on all the metrics exposed by the embedded prometheus exporter in Asterisk. This [file](https://storage.googleapis.com/grafanalabs-integration-assets/asterisk/files/asterisk_prometheus_metrics) shows sample metrics exposed by the exporter of an Asterisk instance.

In order for this dashboard to work, you must enable the embedded [prometheus exporter in Asterisk](https://wiki.asterisk.org/wiki/display/AST/Asterisk+18+Configuration_res_prometheus) to collect and expose Asterisk metrics.

The embedded prometheus exporter also requires the embedded [asterisk http server](https://wiki.asterisk.org/wiki/display/AST/Asterisk+18+Configuration_res_prometheus) to be [enabled](https://wiki.asterisk.org/wiki/display/AST/Setting+up+the+Asterisk+HTTP+server).

Sample [res_prometheus](https://storage.googleapis.com/grafanalabs-integration-assets/asterisk/files/prometheus.conf) configration `/etc/asterisk/prometheus`.

Sample [http_server](https://storage.googleapis.com/grafanalabs-integration-assets/asterisk/files/http_additional.conf) configration `/etc/asterisk/http_additional.conf`.

The dashboard contains multiple sections as shown below:

***Asterisk Overview***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/asterisk/screenshots/asterisk_overview_1.png)

***Channels Information***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/asterisk/screenshots/asterisk_overview_2.png)

***Endpoints Information***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/asterisk/screenshots/asterisk_overview_3.png)

***Bridges Information***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/asterisk/screenshots/asterisk_overview_4.png)

***Asterisk System Information***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/asterisk/screenshots/asterisk_overview_5.png)

## Asterisk Logs
This dashboard provides metrics and details on Asterisk log files. Currently only the main Asterisk log file `/var/log/asterisk/full` is tracked in this dashboard.

This dashboard requires [Promtail and Loki to be installed](https://grafana.com/docs/loki/latest/installation/) and provisioned for logs with your Grafana instance.

The dashboard contains multiple sections as shown below:

***Logs Overview***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/asterisk/screenshots/asterisk_logs_1.png)

***Errors***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/asterisk/screenshots/asterisk_logs_2.png)

***Warnings***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/asterisk/screenshots/asterisk_logs_3.png)

***Complete Log File***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/asterisk/screenshots/asterisk_logs_4.png)

## Dashboard Links
On the top right of dashboards, you will find a link to quickly switch between the two dashboards while keeping the time range selection the same
## How to use this mixin
The mixin creates recording and alerting rules for Prometheus and suitable 
dashboards for Grafana.

To use them, you need to have `mixtool` and `jsonnetfmt` installed. If you
have a working Go development environment, it's easiest to run the following:
```bash
$ go get github.com/monitoring-mixins/mixtool/cmd/mixtool
$ go get github.com/google/go-jsonnet/cmd/jsonnetfmt
```

You can then build the Prometheus rules files `alerts.yaml` and
`rules.yaml` and a directory `dashboard_out` with the JSON dashboard files
for Grafana:
```bash
$ make build
```

For more advanced uses of mixins, see
https://github.com/monitoring-mixins/docs.