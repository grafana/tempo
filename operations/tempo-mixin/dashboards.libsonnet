local g = import 'grafana-builder/grafana.libsonnet';
local utils = import 'mixin-utils/utils.libsonnet';
local dashboard_utils = import 'dashboard-utils.libsonnet';

dashboard_utils {
  grafanaDashboards+: {
    'tempo-operational.json': import './tempo-operational.json',
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
        g.row('Jaeger Query')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('jaeger_rpc_http_requests_total{%s, endpoint="/api/traces/-traceID-"}' % $.jobMatcher($._config.jobs.querier))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('jaeger_query_latency', '{%s,operation="get_trace"}' % $.jobMatcher($._config.jobs.querier))
        )
      )
      .addRow(
        g.row('Query Frontend')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route="tempo_api_traces_traceid"}' % $.jobMatcher($._config.jobs.query_frontend))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route="tempo_api_traces_traceid"}' % $.jobMatcher($._config.jobs.query_frontend))
        )
      )
      .addRow(
        g.row('Querier')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route="querier_tempo_api_traces_traceid"}' % $.jobMatcher($._config.jobs.querier))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route="querier_tempo_api_traces_traceid"}' % $.jobMatcher($._config.jobs.querier))
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
        g.row('GCS')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempodb_gcs_request_duration_seconds_count{%s,operation="GET"}' % $.jobMatcher($._config.jobs.querier))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempodb_gcs_request_duration_seconds', '{%s,operation="GET"}' % $.jobMatcher($._config.jobs.querier))
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
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route=~"/tempopb.Pusher/Push.*"}' % $.jobMatcher($._config.jobs.ingester))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route=~"/tempopb.Pusher/Push.*"}' % $.jobMatcher($._config.jobs.ingester))
        )
      )
      .addRow(
        g.row('Memcached - Ingester')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('cortex_memcache_request_duration_seconds_count{%s,method="Memcache.Put"}' % $.jobMatcher($._config.jobs.ingester))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('cortex_memcache_request_duration_seconds', '{%s,method="Memcache.Put"}' % $.jobMatcher($._config.jobs.ingester))
        )
      )
      .addRow(
        g.row('GCS - Ingester')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempodb_gcs_request_duration_seconds_count{%s,operation="POST"}' % $.jobMatcher($._config.jobs.ingester))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempodb_gcs_request_duration_seconds', '{%s,operation="POST"}' % $.jobMatcher($._config.jobs.ingester))
        )
      )
      .addRow(
        g.row('Memcached - Compactor')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('cortex_memcache_request_duration_seconds_count{%s,method="Memcache.Put"}' % $.jobMatcher($._config.jobs.compactor))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('cortex_memcache_request_duration_seconds', '{%s,method="Memcache.Put"}' % $.jobMatcher($._config.jobs.compactor))
        )
      )
      .addRow(
        g.row('GCS - Compactor')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempodb_gcs_request_duration_seconds_count{%s,operation="POST"}' % $.jobMatcher($._config.jobs.compactor))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempodb_gcs_request_duration_seconds', '{%s,operation="POST"}' % $.jobMatcher($._config.jobs.compactor))
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
        g.row('Distributor')
        .addPanel(
          $.containerCPUUsagePanel('CPU', $._config.jobs.distributor),
        )
        .addPanel(
          $.containerMemoryWorkingSetPanel('Memory (workingset)', $._config.jobs.distributor),
        )
        .addPanel(
          $.goHeapInUsePanel('Memory (go heap inuse)', $.jobMatcher($._config.jobs.distributor)),
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
        g.row('Query Frontend')
        .addPanel(
          $.containerCPUUsagePanel('CPU', $._config.jobs.query_frontend),
        )
        .addPanel(
          $.containerMemoryWorkingSetPanel('Memory (workingset)', $._config.jobs.query_frontend),
        )
        .addPanel(
          $.goHeapInUsePanel('Memory (go heap inuse)', $.jobMatcher($._config.jobs.query_frontend)),
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
