# tempo-mixin

Dashboards, rules and alerts are in the [`yamls`](./yamls) folder. Use them directly in Prometheus & Grafana to monitor Tempo.

### Build

To regenerate dashboards, rule and alerts, run `make all`.

This requires [jsonnet](https://jsonnet.org/) and [jsonnet-bundler](https://github.com/jsonnet-bundler/jsonnet-bundler) to be installed, install these with the following commands:

```console
brew install jsonnet
go install github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@v0.4.0
```
