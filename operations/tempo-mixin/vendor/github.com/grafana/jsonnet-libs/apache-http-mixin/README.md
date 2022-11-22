# Apache HTTP server mixin

Apache HTTP mixin is a set of configurable Grafana dashboards and alerts based on the metrics exported by the [Apache exporter](https://github.com/Lusitaniae/apache_exporter).

Based on Apache dashboard by rfrail3: https://github.com/rfrail3/grafana-dashboards.

![image](https://user-images.githubusercontent.com/14870891/170320166-91bf48a6-0e21-48fd-873b-2483ee402339.png)

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

## Apache logs

![image](https://user-images.githubusercontent.com/14870891/170279623-7aa6cc8f-7928-4d90-9c9b-94c5148b4488.png)

Logs support is disabled by default. Change enableLokiLogs to `true` inside `config.libsonnet` to enable log panels. Then run `make` again to regenerate the dashboards:


```bash
make
```

The easiest to way to collect logs and metrics from apache server is to use [Grafana Agent](https://github.com/grafana/agent) with the config like below:
The most important part is that `job` and `instance` labels must match for agent integration, logs and metrics configs.
```yaml
# For a full configuration reference, see: https://github.com/grafana/agent/blob/main/docs/configuration-reference.md.
server:
  log_level: warn
metrics:
  wal_directory: /var/lib/grafana-agent/wal
  global:
    scrape_interval: 1m
    external_labels:
      instance: <yourhostname>
    remote_write: 
      - url: https://cortex/api/v1/write
    scrape_configs:
      # scrape apache_exporter
      - job_name: integrations/apache
        static_configs:
          - targets: ['localhost:9117']

integrations:
  
  agent:
    enabled: true
    metric_relabel_configs:
    # required for apache integrations,
    # scraping agent endpoint is required for apache histogram metric collection.
    - source_labels: [exported_job]
        target_label: job
        replacement: '$1'
    - source_labels: [exported_instance]
        target_label: instance
    - regex: (exported_instance|exported_job)
        action: labeldrop
logs:
  configs:
  - name: integrations
    clients:
      - url: http://loki:3100/loki/api/v1/push
        external_labels:
          instance: <yourhostname>
    positions:
      filename: /var/lib/grafana-agent/logs/positions.yaml
    scrape_configs:
    - job_name: integrations/apache_error
      static_configs:
      - targets:
        - localhost
        labels:
          __path__: /var/log/apache2/error.log
          job: integrations/apache
      pipeline_stages:
        - regex:
            # https://regex101.com/r/zNIq1V/1
            expression: '^\[[^ ]* (?P<timestamp>[^\]]*)\] \[(?:(?P<module>[^:\]]+):)?(?P<level>[^\]]+)\](?: \[pid (?P<pid>[^\]]*)\])?(?: \[client (?P<client>[^\]]*)\])? (?P<message>.*)$'
        - labels:
            module:
            level:
        - static_labels:
            logtype: error
    - job_name: integrations/apache_access
      static_configs:
      - targets:
        - localhost
        labels:
          __path__: /var/log/apache2/access.log
          job: integrations/apache
      pipeline_stages:
        - regex:
            # https://regex101.com/r/9G75bY/1
            expression: '^(?P<ip>[^ ]*) [^ ]* (?P<user>[^ ]*) \[(?P<timestamp>[^\]]*)\] "(?P<method>\S+)(?: +(?P<path>[^ ]*) +\S*)?" (?P<code>[^ ]*) (?P<size>[^ ]*)(?: "(?P<referer>[^\"]*)" "(?P<agent>.*)")?$'
        - metrics:
            response_http_codes:
              type: Histogram
              description: "Apache reponses by HTTP codes"
              prefix: apache_
              source: code
              config:
                buckets: [199,299,399,499,599]
        - labels:
            method:
        - static_labels:
            logtype: access
```

## Import dashboards and alerts using Grizzly tool

Install grizzly first: https://grafana.github.io/grizzly/installation/

Set env variables GRAFANA_URL and optionally CORTEX_ADDRESS (see for [details](https://grafana.github.io/grizzly/authentication/)).

Then run to import the dashboards and alerts into Grafana instance:
```bash
make deploy
```

For more advanced uses of mixins, see
https://github.com/monitoring-mixins/docs.
