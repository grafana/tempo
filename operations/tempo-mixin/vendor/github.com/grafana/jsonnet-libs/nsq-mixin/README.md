# NSQ Mixin

NSQ mixin is a set of configurable Grafana dashboards and alerts based on the metrics exported by the [NSQ statsd integration](https://nsq.io/components/nsqd.html#statsd--graphite-integration).
In order to use it, you would need https://github.com/prometheus/statsd_exporter or https://grafana.com/docs/grafana-cloud/agent/ with statsd integration enabled.

This NSQ mixin interesting features:
1) Set of two dashboards:
NSQ Topics - view metrics grouped by topics and channels
![screenshot-0](https://storage.googleapis.com/grafanalabs-integration-assets/nsq/screenshots/screenshot0.png)
NSQ Instances - view metrics  grouped by nsqd instances
![screenshot-1](https://storage.googleapis.com/grafanalabs-integration-assets/nsq/screenshots/screenshot1.png)

2) Dashboard links to quickly jump from one dashboard(view) to another.
3) [Data links ](https://grafana.com/docs/grafana/latest/linking/data-links/#data-links) to contextually jump from instance view to topics view and vice versa
![image](https://user-images.githubusercontent.com/14870891/149532730-6fdebd8d-204e-4962-861d-9c78af437afb.png)
4) Alerts for topics and channels depth.

## statsd_exporter/grafana agent configuration

In the Grafana agent configuration file, the agent's statsd_exporter integration must be enabled and configured the following way:

```yaml
metrics:
  wal_directory: /tmp/wal
  configs:
    - name: integrations
      remote_write:
        - url: http://cortex:9009/api/prom/push
integrations:
  statsd_exporter:
    enabled: true
    metric_relabel_configs:
      - source_labels: [exported_job]
        target_label: job
        replacement: 'integrations/$1'
      - source_labels: [exported_instance]
        target_label: instance
      - regex: (exported_instance|exported_job)
        action: labeldrop
    mapping_config:
      defaults:
        match_type: glob
        glob_disable_ordering: false
        ttl: 1m30s
      mappings:
       - match: "nsq.*.topic.*.channel.*.message_count"
         name: "nsq_topic_channel_message_count"
         match_metric_type: counter
         labels:
           instance: "$1"
           job: "nsq"
           topic: "$2"
           channel: "$3"

       - match: "nsq.*.topic.*.channel.*.requeue_count"
         name: "nsq_topic_channel_requeue_count"
         match_metric_type: counter
         labels:
           instance: "$1"
           job: "nsq"
           topic: "$2"
           channel: "$3"

       - match: "nsq.*.topic.*.channel.*.timeout_count"
         name: "nsq_topic_channel_timeout_count"
         match_metric_type: counter
         labels:
           instance: "$1"
           job: "nsq"
           topic: "$2"
           channel: "$3"

       - match: "nsq.*.topic.*.channel.*.*"
         name: "nsq_topic_channel_${4}"
         match_metric_type: gauge
         labels:
           instance: "$1"
           job: "nsq"
           topic: "$2"
           channel: "$3"

      #nsq.<nsq_host>_<nsq_port>.topic.<topic_name>.backend_depth [gauge]
       - match: "nsq.*.topic.*.message_count"
         name: "nsq_topic_message_count"
         help: Total number of messages for the topic
         match_metric_type: counter
         labels:
           instance: "$1"
           job: "nsq"
           topic: "$2"

       - match: "nsq.*.topic.*.message_bytes"
         name: "nsq_topic_message_bytes"
         help: Total number of bytes of all messages
         match_metric_type: counter
         labels:
           instance: "$1"
           job: "nsq"
           topic: "$2"

       - match: "nsq.*.topic.*.*" #depth, backend_depth and e2e_processing_latency_<percent>
         name: "nsq_topic_${3}"
         match_metric_type: gauge
         labels:
           instance: "$1"
           job: "nsq"
           topic: "$2"
      # mem
      # nsq.<nsq_host>_<nsq_port>.mem.gc_runs
       - match: "nsq.*.mem.gc_runs"
         name: "nsq_mem_gc_runs"
         match_metric_type: counter
         labels:
           instance: "$1"
           job: "nsq"

       - match: "nsq.*.mem.*"
         name: "nsq_mem_${2}"
         match_metric_type: gauge
         labels:
           instance: "$1"
           job: "nsq"

```

## Generate config files

You can manually generate dashboards, but first you should install some tools:

```bash
go install github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@latest
go install github.com/google/go-jsonnet/cmd/jsonnet@latest
# or in brew: brew install go-jsonnet
```

For linting and formatting, you would also need `mixtool` and `jsonnetfmt` installed. If you
have a working Go development environment, it's easiest to run the following:

```bash
go install github.com/monitoring-mixins/mixtool/cmd/mixtool@latest
go install github.com/google/go-jsonnet/cmd/jsonnetfmt@latest
```

The files in `dashboards_out` need to be imported
into your Grafana server.  The exact details will be depending on your environment.

`prometheus_alerts.yaml` needs to be imported into Prometheus.

Edit `config.libsonnet` if required and then build JSON dashboard files for Grafana:

```bash
make
```

For more advanced uses of mixins, see
https://github.com/monitoring-mixins/docs.
