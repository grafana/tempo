# GitLab Mixin

The GitLab mixin is a set of configurable Grafana dashboards and alerts based on the [metrics available from an instance of GitLab EE](https://docs.gitlab.com/ee/administration/monitoring/prometheus/gitlab_metrics.html).

The GitLab mixin contains the following dashboards:
- GitLab Overview

## GitLab Overview

The GitLab Overview dashboard provides details on request traffic, pipeline activity, and rails error logs. To get GitLab rails error logs, [Promtail and Loki needs to be installed](https://grafana.com/docs/loki/latest/installation/) and provisioned for logs with your Grafana instance. The default GitLab rails error log path is `/var/log/gitlab/gitlab-rails/exceptions_json.log`.

![First screenshot of the overview dashboard](https://storage.googleapis.com/grafanalabs-integration-assets/gitlab/screenshots/gitlab_overview_1.png)
![Second screenshot of the overview dashboard](https://storage.googleapis.com/grafanalabs-integration-assets/gitlab/screenshots/gitlab_overview_2.png)
![Third screenshot of the overview dashboard](https://storage.googleapis.com/grafanalabs-integration-assets/gitlab/screenshots/gitlab_overview_3.png)

GitLab rails error logs are enabled by default in the `config.libsonnet` and can be removed by setting `enableLokiLogs` to `false`. Then run `make` again to regenerate the dashboard:

```
{
  _config+:: {
    enableLokiLogs: false,
  },
}
```

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
into your Grafana server. The exact details will be depending on your environment.

`prometheus_alerts.yaml` needs to be imported into Prometheus.

## Generate dashboards and alerts

Edit `config.libsonnet` if required and then build JSON dashboard files for Grafana:

```bash
make
```

For more advanced uses of mixins, see
https://github.com/monitoring-mixins/docs.
