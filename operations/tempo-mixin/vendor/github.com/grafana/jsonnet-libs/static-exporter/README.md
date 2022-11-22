# static-exporter jsonnet library

Export metrics that are relatively static.

## Usage

Install it with jsonnet-bundler:

```console
jb install github.com/grafana/jsonnet-libs/static-exporter
```

## Example

```jsonnet
// environments/default/main.jsonnet
local static_exporter = import 'github.com/grafana/jsonnet-libs/static-expoter/main.libsonnet';

{
  team_holiday_exporter:
    static_exporter.new('team-holiday-exporter')
    + static_exporter.withMetrics([
      static_exporter.metric.new(
        'holidays_day_count',
        'Available official holidays',
      )
      + static_exporter.metric.withValue({location: 'EMEA'}, 30)
      + static_exporter.metric.withValue({location: 'NASA'}, 25)
      + static_exporter.metric.withValue({location: 'APAC'}, 37),

      static_exporter.metric.new(
        'team_info',
        'Information about team members',
      )
      + static_exporter.metric.withLabelMapList([
        {name: 'Hiro', team: 'frontend', location:'NASA'},
        {name: 'Baymax', team: 'frontend', location:'EMEA'},
        {name: 'Honey', team: 'platform', location:'EMEA'},
        {name: 'Tomago', team: 'platform', location:'APAC'},
        {name: 'Wasabi', team: 'mobile', location:'NASA'},
        {name: 'Fred', team: 'mobile', location:'APAC'},
      ]),
    ]),
}
```
