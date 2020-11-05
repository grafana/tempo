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

# Tempo / Operational

The Tempo Operational dashboard deserves special mention b/c it probably a stack of dashboard anti-patterns.  It's big and complex, doesn't use jsonnet and displays far too many metrics in one place.  And I love it.  For just getting started the reads, write and resources dashboards are great places to learn how to monitor Tempo in an opaque way.

This dashboard is included in this repo for two reasons:

- It provides a stack of metrics for other operators to consider monitoring while running Tempo.
- We want it in our internal infrastructure and we vendor the tempo-mixin to do this.