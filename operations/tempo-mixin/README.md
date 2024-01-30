# tempo-mixin

Once installed, dashboards, rules, and alerts are in the [`yamls`](./yamls) folder. Use them directly in Prometheus and Grafana to monitor Tempo.

## Build

To regenerate dashboards, rule and alerts, run `make all`.

This requires [jsonnet](https://jsonnet.org/) and [jsonnet-bundler](https://github.com/jsonnet-bundler/jsonnet-bundler) to be installed. 

On macOS, you can install these with the following commands:

```console
brew install jsonnet
brew install jsonnet-bundler 
go install github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@v0.4.0
```

## Use the mixins

Once you run `make all`, the mixins are created in the `tempo-mixin-compiled` folder. 

```
➜   make all                                                                                                                          git:(main|)
jb install
jsonnet -J vendor -S dashboards.jsonnet -m ../tempo-mixin-compiled/dashboards/
../tempo-mixin-compiled/dashboards/tempo-operational.json
../tempo-mixin-compiled/dashboards/tempo-reads.json
../tempo-mixin-compiled/dashboards/tempo-resources.json
../tempo-mixin-compiled/dashboards/tempo-rollout-progress.json
../tempo-mixin-compiled/dashboards/tempo-tenants.json
../tempo-mixin-compiled/dashboards/tempo-writes.json
jsonnet -J vendor -S alerts.jsonnet > ../tempo-mixin-compiled/alerts.yaml
jsonnet -J vendor -S rules.jsonnet > ../tempo-mixin-compiled/rules.yaml
```

Alerts and rules are listed in their matching files: 
* Alerts -> `tempo-mixin-compiled/alerts.yaml`
* Rules -> 'tempo-mixin-compiled/rules.yaml`

For information on using the dashboards, refer to the [mixin runbook](operations/tempo-mixin/runbook.md).
