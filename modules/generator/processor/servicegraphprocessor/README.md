# What is this?

This directory is a complete copy of https://github.com/grafana/agent/tree/main/pkg/traces/servicegraphprocessor :(

Since grafana/agent depends on Cortex, Loki and Tempo it's impossible to include it as a dependency of Tempo.
This copy also has some slight changes to create a processor directly and set the prometheus.Registerer 

To see the changes that were made check out both repositories and run:

```shell
diff agent/pkg/traces/servicegraphprocessor tempo/modules/generator/processor/servicegraphprocessor
```
