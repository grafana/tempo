# WSO2 Enterprise Integrator Mixin

WSO2 Enterprise Integrator is a powerful configuration-driven approach to integration, which allows developers to build integration solutions graphically. 

This is a hybrid platform that enables API-centric integration and supports various integration architecture styles: microservices architecture, cloud-native architecture, or a centralized ESB architecture. This integration platform offers a graphical/configuration-driven approach to developing integrations for any of the architectural styles.

For more information, please refer to the [product webpage](https://ei.docs.wso2.com/en/latest/micro-integrator/overview/introduction/).

This mixin contains 5 dashboards for monitoring a Enterprise Integrator environment, including the Micro Integrator profile. They were based on the dashboards available on the Grafana dashboard registry.

[WSO2 Integration Cluster Metrics](https://grafana.com/grafana/dashboards/12783)\
[WSO2 Integration Node Metrics](https://grafana.com/grafana/dashboards/12887)\
[WSO2 API Metrics](https://grafana.com/grafana/dashboards/12888)\
[WSO2 Proxy Service Metrics](https://grafana.com/grafana/dashboards/12889)\
[WSO2 Inbound Endpoint Metrics](https://grafana.com/grafana/dashboards/12890)


To use them, you need to have `mixtool` and `jsonnetfmt` installed. If you have a working Go development environment, it's easiest to run the following:

```bash
$ go get github.com/monitoring-mixins/mixtool/cmd/mixtool
$ go get github.com/google/go-jsonnet/cmd/jsonnetfmt
```

You can then build a directory `dashboard_out` with the JSON dashboard files for Grafana:

```bash
$ make build
```

For more advanced uses of mixins, see [Prometheus Monitoring Mixins docs](https://github.com/monitoring-mixins/docs).
