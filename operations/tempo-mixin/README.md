To generate dashboards with this mixin use:

```console
jb install && jsonnet -J vendor -S dashboards.jsonnet -m out
```

To generate alerts, use:
```console
jsonnet -J vendor -S alerts.jsonnet > out/alerts.yaml
```

To generate recording rules, use:
```console
jsonnet -J vendor -S rules.jsonnet > out/rules.yaml
```
