Dashboards, rules and alerts are in the `yamls` folder. Use them directly in Prometheus & Grafana to monitor Tempo.

To generate dashboards with this mixin use:

```console
jb install && jsonnet -J vendor -S dashboards.jsonnet -m yamls
```

To generate alerts, use:
```console
jsonnet -J vendor -S alerts.jsonnet > yamls/alerts.yaml
```

To generate recording rules, use:
```console
jsonnet -J vendor -S rules.jsonnet > yamls/rules.yaml
```
