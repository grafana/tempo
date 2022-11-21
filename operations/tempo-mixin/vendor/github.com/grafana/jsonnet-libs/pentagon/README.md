# Pentagon

This is a library to deploy [Grafana's fork of Pentagon](https://github.com/grafana/pentagon).

The library is intended to deploy 1 pentagon instance per namespace.

## Example

```
local pentagon = import 'pentagon/pentagon.libsonnet',

{
  pentagon:
    pentagon
    + pentagon.addPentagonMapping('secret/data/path/to/secret', 'kubernetes_secret_name')
    + pentagon.addPentagonMapping('secret/data/path/to/other_secret', 'kubernetes_other_secret_name')
    + {
      _config+:: {
        namespace: 'default',
        cluster: 'prod',
      },
    },
}
```
