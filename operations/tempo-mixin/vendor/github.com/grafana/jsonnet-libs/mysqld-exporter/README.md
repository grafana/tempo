# mysqld-exporter jsonnet library

Jsonnet library for [mysqld_exporter](https://github.com/prometheus/mysqld_exporter).

## Usage

Install it with jsonnet-bundler:

```console
jb install github.com/grafana/jsonnet-libs/mysqld-exporter
```

Import into your jsonnet:

```jsonnet
// environments/default/main.jsonnet
local mysqld_exporter = import 'github.com/grafana/jsonnet-libs/mysqld-exporter/main.libsonnet';

{
  mysqld_exporter:
    mysqld_exporter.new(
      name='cloudsql-mysqld-exporter',
      user='admin',
      host='mysql',
    )
    + mysqld_exporter.withPassword(error 'requires superSecurePassword'),
}
```
