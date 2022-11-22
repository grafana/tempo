# Nomad mixin

Nomad mixin is a set of configurable Grafana dashboards.

![screenshot-0](https://storage.googleapis.com/grafanalabs-integration-assets/nomad/screenshots/screenshot0.png)
## Nomad cluster configuration

Add the stanza below in your Nomad client and server configuration files, by default it would be /etc/nomad.d/nomad.hcl:

```
telemetry {
  collection_interval = "1s"
  disable_hostname = true
  prometheus_metrics = true
  publish_allocation_metrics = true
  publish_node_metrics = true
}
```


## Prometheus/grafana agent configuration

In the agent configuration file, the agent must be pointed to each nomad server and nomad client that compose the Nomad cluster, such as `nomad:4646` in the example below, that exposes a `/metrics` endpoint.

```yaml
metrics:
  wal_directory: /tmp/wal
  configs:
    - name: integrations
      scrape_configs:
        - job_name: integrations/nomad
          metrics_path: /v1/metrics
          params:
            format: ['prometheus']
          static_configs:
            - targets: ['nomad1:4646', 'nomad2:4646', 'nomad3:4646', 'nomad-client1:4646']
```

Instead of using static discovery, consul service discovery can be used to discover all the nodes of the Nomad cluster:

```yaml
- job_name: 'integrations/nomad'
  consul_sd_configs:
  - server: 'consul.service.consul:8500'
    services: ['nomad-client', 'nomad']
  metrics_path: /v1/metrics
  params:
    format: ['prometheus']
  relabel_configs:
  - source_labels: ['__meta_consul_tags']
    regex: '(.*)http(.*)'
    action: keep
  - source_labels: [__meta_consul_node]
    target_label: instance

```

For more advanced uses of mixins, see
https://github.com/monitoring-mixins/docs.
