# Kafka Mixin

The Kafka Mixin is a set of configurable, reusable, and extensible dashboards based on the ones discussed on this [blog post from Confluent](https://www.confluent.io/blog/monitor-kafka-clusters-with-prometheus-grafana-and-confluent/), which are fed by a set of JMX Exporters configured with the config files included in this repo, and an additional Lag Overview dashboard based on this [Kafka Overview dashboard](https://grafana.com/grafana/dashboards/7589) which is fed by a [GoLang based exporter](https://github.com/davidmparrott/kafka_exporter).

This mixin includes the following dashboards:
Kafka Overview - Gives an overview of your Kafka cluster resource usage, throughput, and general healthiness 
kafka-topics - Gives informations about the throughput of specific (filterable) topics
Zookeeper Overview - Gives an overview of your Zookeeper nodes resource usage and general healthiness 
Ksqldb Overview - Gives an overview of your ksqldb cluster as queries resource usage, throughput, and general healthiness 
Connect Overview - Gives an overview of your Kafka Connect cluster and tasks resource usage, throughput, and general healthiness
Schema Registry Overview - Gives an overview of your Kafka Schema Registry cluster resource usage, throughput, and general healthiness
Kafka Lag Overview - Gives Lag metrics by time and offset count on topics and partitions of the cluster

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
