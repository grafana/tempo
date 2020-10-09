local g = import 'grafana-builder/grafana.libsonnet';
local utils = import 'mixin-utils/utils.libsonnet';
local dashboard_utils = import 'dashboard-utils.libsonnet';

dashboard_utils {
  grafanaDashboards+: {
    // 'tempo-operational.json': import './tempo-operational.json',
    'tempo-reads.json':
      $.dashboard('Tempo / Reads')
      .addClusterSelectorTemplates()
      .addRow(
        g.row('Gateway')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route="tempo_api_traces_traceid"}' % $.jobMatcher($._config.jobs.gateway))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route="tempo_api_traces_traceid"}' % $.jobMatcher($._config.jobs.gateway))
        )
      )
      .addRow(
        g.row('Querier')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route="api_traces_traceid"}' % $.jobMatcher($._config.jobs.querier))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route="api_traces_traceid"}' % $.jobMatcher($._config.jobs.querier))
        )
      )
      .addRow(
        g.row('Memcached')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('cortex_memcache_request_duration_seconds_count{%s,method=~"Memcache.Get|Memcache.GetMulti"}' % $.jobMatcher($._config.jobs.querier))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('cortex_memcache_request_duration_seconds', '{%s,method=~"Memcache.Get|Memcache.GetMulti"}' % $.jobMatcher($._config.jobs.querier))
        )
      )
      .addRow(
        g.row('Ingester')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route="/tempopb.Querier/FindTraceByID"}' % $.jobMatcher($._config.jobs.ingester))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route="/tempopb.Querier/FindTraceByID"}' % $.jobMatcher($._config.jobs.ingester))
        )
      )
      .addRow(
        g.row('Querier Disk Cache')
        .addPanel(
          g.panel('Lookups') +
          g.queryPanel('rate(tempodb_disk_cache_total[$__interval])', '{{type}}'),
        )
        .addPanel(
          g.panel('Misses') +
          g.queryPanel('rate(tempodb_disk_cache_miss_total[$__interval])', '{{type}}'),
        )
        .addPanel(
          g.panel('Purges') +
          $.hiddenLegendQueryPanel('rate(tempodb_disk_cache_clean_total[$__interval])', '{{pod}}'),
        )
      )
      .addRow(
        g.row('TempoDB Access')
        .addPanel(
          g.panel('p99') +
          g.queryPanel('histogram_quantile(.99, sum(rate(tempo_query_reads_bucket[$__interval])) by (layer, le))', '{{layer}}'),
        )
        .addPanel(
          g.panel('p50') +
          g.queryPanel('histogram_quantile(.5, sum(rate(tempo_query_reads_bucket[$__interval])) by (layer, le))', '{{layer}}'),
        )
        .addPanel(
          g.panel('Bytes Read') +
          g.queryPanel('histogram_quantile(.99, sum(rate(tempo_query_bytes_read_bucket[$__interval])) by (le))', '0.99') +
          g.queryPanel('histogram_quantile(.9, sum(rate(tempo_query_bytes_read_bucket[$__interval])) by (le))', '0.9') +
          g.queryPanel('histogram_quantile(.5, sum(rate(tempo_query_bytes_read_bucket[$__interval])) by (le))', '0.5'),
        )
      ),
    'tempo-writes.json':
      $.dashboard('Tempo / Writes')
      .addClusterSelectorTemplates()
      .addRow(
        g.row('Gateway')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route="opentelemetry_proto_collector_trace_v1_traceservice_export"}' % $.jobMatcher($._config.jobs.gateway))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route="opentelemetry_proto_collector_trace_v1_traceservice_export"}' % $.jobMatcher($._config.jobs.gateway))
        )
      )
      .addRow(
        g.row('Distributor')
        .addPanel(
          $.panel('Spans/Second') +
          $.queryPanel('sum(rate(tempo_receiver_accepted_spans{%s}[$__interval]))' % $.jobMatcher($._config.jobs.distributor), "accepted") +
          $.queryPanel('sum(rate(tempo_receiver_refused_spans{%s}[$__interval]))' % $.jobMatcher($._config.jobs.distributor), "refused")
        )
      )
      .addRow(
        g.row('Ingester')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route="/tempopb.Pusher/Push"}' % $.jobMatcher($._config.jobs.ingester))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route="/tempopb.Pusher/Push"}' % $.jobMatcher($._config.jobs.ingester))
        )
      ),
    'tempo-resources.json':
      $.dashboard('Tempo / Resources')
      .addClusterSelectorTemplates()
      .addRow(
        g.row('Gateway')
        .addPanel(
          $.containerCPUUsagePanel('CPU', $._config.jobs.gateway),
        )
        .addPanel(
          $.containerMemoryWorkingSetPanel('Memory (workingset)', $._config.jobs.gateway),
        )
        .addPanel(
          $.goHeapInUsePanel('Memory (go heap inuse)', $.jobMatcher($._config.jobs.gateway)),
        )
      )
      .addRow(
        g.row('Querier')
        .addPanel(
          $.containerCPUUsagePanel('CPU', $._config.jobs.querier),
        )
        .addPanel(
          $.containerMemoryWorkingSetPanel('Memory (workingset)', $._config.jobs.querier),
        )
        .addPanel(
          $.goHeapInUsePanel('Memory (go heap inuse)', $.jobMatcher($._config.jobs.querier)),
        )
      )
      .addRow(
        g.row('Ingester')
        .addPanel(
          $.containerCPUUsagePanel('CPU', $._config.jobs.ingester),
        )
        .addPanel(
          $.containerMemoryWorkingSetPanel('Memory (workingset)', $._config.jobs.ingester),
        )
        .addPanel(
          $.goHeapInUsePanel('Memory (go heap inuse)', $.jobMatcher($._config.jobs.ingester)),
        )
      )
      .addRow(
        g.row('Compactor')
        .addPanel(
          $.containerCPUUsagePanel('CPU', $._config.jobs.compactor),
        )
        .addPanel(
          $.containerMemoryWorkingSetPanel('Memory (workingset)', $._config.jobs.compactor),
        )
        .addPanel(
          $.goHeapInUsePanel('Memory (go heap inuse)', $.jobMatcher($._config.jobs.compactor)),
        )
      )
  },
}
