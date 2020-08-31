local g = import 'grafana-builder/grafana.libsonnet';
local utils = import 'mixin-utils/utils.libsonnet';

{
  grafanaDashboards+: {
    'tempo-operational.json': import './tempo-operational.json',
    'tempo-reads.json':
      g.dashboard('Tempo / Reads')
      .addRow(
        g.row('Cache')
        .addPanel(
          g.queryPanel('tempodb_disk_cache_total[$interval])', 'Lookups'),
        )
        .addPanel(
          g.queryPanel('tempodb_disk_cache_miss_total[$interval])', 'Misses'),
        )
        .addPanel(
          g.queryPanel('tempodb_disk_cache_clean_total[$interval])', 'Purges'),
        )
      ),
  },
}
