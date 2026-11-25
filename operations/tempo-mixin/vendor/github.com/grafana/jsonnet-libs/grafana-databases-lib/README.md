# grafana-databases-lib

Shared dashboard panels and rows for the Mimir, Loki, Tempo and Pyroscope
mixins, built on `grafonnet-latest`. Each mixin assembles its own dashboards
from them, so the shared parts stay consistent and the component-specific parts
stay in the mixin.

Panels and rows are dashboard v2 (`dashboard.grafana.app/v2`) elements and
rows. Dashboards still built with
grafana-builder use them through `dblib.legacy`, which converts them to
classic panels; once a dashboard no longer needs `legacy`, it is a v2 dashboard.

## Usage

```jsonnet
local dblib = import 'github.com/grafana/jsonnet-libs/grafana-databases-lib/main.libsonnet';

local databases =
  dblib.new({ clusterLabel: 'cluster', namespaceLabel: 'namespace' })
  + dblib.autoscaling.withConfigMixin({
    components: {
      distributor: { title: 'Distributor', hpaName: 'keda-hpa-distributor', keda: false },
      querier: { title: 'Querier', hpaName: 'keda-hpa-querier.*' },
    },
  });

databases.autoscaling.rows.querier              // a named row (a v2 RowsLayoutRow)
databases.autoscaling.panels.querier.replicas   // a named panel (a v2 Panel element)
```

A v2 dashboard keeps its panels in `spec.elements`, keyed by name, and their
placement in `spec.layout`. Every library panel carries its element name (e.g.
`autoscaling-querier-replicas`) and every row its panels, so `dblib.elements`
and `dblib.layout` build both halves from a list of rows, including custom rows
of named panels (`dblib.row`):

```jsonnet
local rows = [
  databases.autoscaling.rows.querier,
  dblib.row('Ruler', [databases.autoscaling.panels.ruler.replicas]),
];

local dash = (import 'g.libsonnet').apps.dashboard.v2;
dash.new('my-dashboard', 'My dashboard')
+ dash.spec.withElements(dblib.elements(rows))
+ dash.spec.withLayout(dblib.layout(rows))
// + variables, timeSettings, ... see test/dashboard.jsonnet
```

Everything reads `config` through `self`, so a config mixin applies wherever it
is added, and overriding a named panel also changes it in its rows:

```jsonnet
databases
+ dblib.autoscaling.withPanelMixin(
  component='querier',
  panel='replicas',
  patch={ spec+: { title: 'Querier replicas' } },
)
```

In a mixin, build the instance in a hidden field from `$._config`, so that later
`_config+::` overrides still reach it.

## Config

Library-wide, through `dblib.new(config)` or `dblib.withConfigMixin`:

| Field                             | Default                          | Purpose                                      |
| --------------------------------- | -------------------------------- | -------------------------------------------- |
| `datasource`                      | `$datasource`                    | Prometheus datasource variable/uid           |
| `clusterLabel` / `namespaceLabel` | `cluster` / `namespace`          | Labels identifying a cluster and namespace   |
| `selector`                        | derived from the labels          | Matcher applied to kube-state-metrics series |
| `groupBy`                         | `[clusterLabel, namespaceLabel]` | Labels used to join series                   |

Each area keeps its own portion under `config.<area>`, set through
`dblib.<area>.withConfigMixin`.

## autoscaling

HPA/KEDA panels and rows, per component.

| Field          | Default                          | Purpose                                 |
| -------------- | -------------------------------- | --------------------------------------- |
| `hpaPrefix`    | `keda-hpa-`                      | Prefix KEDA gives the HPAs it creates   |
| `kedaSelector` | `clusterLabel` + `exported_namespace` | Matcher applied to KEDA series     |
| `triggerLegend` | `{{ scaledObject }} {{ metric }}` | Legend of per-trigger series; `{{ scaler }}` suits named triggers |
| `axisMin`      | `null` (fit to data)             | Y-axis minimum, e.g. `0`                |
| `noValue`      | `Not auto-scaled`                | Shown instead of "No data" without an HPA; per-scaler panels say "Not scaled on \<scaler\>"; `null` keeps "No data" |
| `components`   | `{}`                             | Component key to its settings, below    |

Each component takes `hpaName` (required, a regex matched against
`horizontalpodautoscaler`), `title` (shown in the row title), `keda` (default
`true`; `false` leaves the KEDA-based panels out of the row), `scalers` (one
desired replicas panel per KEDA scaler, e.g. `{ cpu: { title: 'CPU' } }`) and
`extraReplicaQueries` (`{ expr, legend }` series added to the replicas panel).

Named panels per component: `replicas`, `desiredReplicas`,
`desiredReplicasByScaler.<scaler>` and `failures`. The constructors
(`replicasPanel`, `desiredReplicasPanel`, `failuresPanel`) are exposed too, for
panels a mixin needs to build differently.

## Migrating grafana-builder dashboards

`dblib.legacy` lets dashboards still built with grafana-builder use the
library while they are moved to v2. It converts the v2 panels to the classic
panels grafana-builder rows hold, which lay them out by `span`:

```jsonnet
grafanaBuilderRow.addPanel(dblib.legacy.panel(databases.autoscaling.panels.querier.replicas))

std.foldl(function(row, p) row.addPanel(p),
          dblib.legacy.rowPanels(databases.autoscaling.rows.querier),
          grafana.row('Querier - autoscaling'))
```

## Test

```sh
cd test && make tests
```

`test/` holds testonnet suites for the autoscaling area (`test_autoscaling.libsonnet`),
the building blocks (`test_dashboard.libsonnet`) and the grafana-builder helpers
(`test_legacy.libsonnet`), and an example dashboard (`dashboard.jsonnet`)
built from whole library rows and a custom row of named panels. The root
`make tests` target runs them too.
