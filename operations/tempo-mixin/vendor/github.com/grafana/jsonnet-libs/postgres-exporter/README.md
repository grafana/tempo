# postgres-exporter jsonnet library

Jsonnet library for [postgres_exporter](https://github.com/prometheus-community/postgres_exporter).

## Usage

Install it with jsonnet-bundler:

```console
jb install github.com/grafana/jsonnet-libs/postgres-exporter
```

Import into your jsonnet:

```jsonnet
// environments/default/main.jsonnet
local postgres_exporter = import 'github.com/grafana/jsonnet-libs/postgres-exporter/main.libsonnet';
local k = import 'ksonnet-util/kausal.libsonnet';
local envVar = k.core.v1.envVar;

{
  postgres_exporter:
    postgres_exporter.new(
      name='cloudsql-postgres-exporter'
    )
    + postgres_exporter.withEnv([
      envVar.fromSecretRef('USER', 'dbCredentials', 'username'),
      envVar.fromSecretRef('PASSWORD', 'dbCredentials', 'password'),
      envVar.new('HOST', 'host.name.com'),
      envVar.new('PORT', std.toString(5432)),
    ]),
}
```
