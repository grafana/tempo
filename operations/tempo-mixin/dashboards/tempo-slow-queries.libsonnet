local dashboard_utils = import 'dashboard-utils.libsonnet';
local g = import 'grafana-builder/grafana.libsonnet';
local filename = 'tempo-slow-queries.json';

dashboard_utils {
  local selector = '{%s=~"$cluster", namespace=~"$namespace", %s=~"%s"}' % [$._config.per_cluster_label, $._config.per_component_loki_label, $._config.jobs.query_frontend],

  // Per-query response lines logged by the query-frontend for trace by ID, search, tag, and TraceQL metrics
  // queries, including the "- no resp" and "- no metrics" variants. query_kind names the query type.
  // A query is slow when it is slower than $min_duration and its throughput is below $throughput, the same as
  // the query SLO. Trace by ID has no throughput SLO. Lines without a throughput, such as failed queries, count as 0.
  local responseLines = selector + ' |~ `msg="(trace id response|search tag response|search tag values response|search response|query range response|query instant results)( - no (resp|metrics))?"` | logfmt | tenant=~"${tenant_id}" | label_format query_kind=`{{ if hasPrefix "trace id" .msg }}id_lookup{{ else if hasPrefix "search tag" .msg }}metadata{{ else if hasPrefix "search" .msg }}search{{ else if hasPrefix "query range" .msg }}query_range{{ else }}instant_query{{ end }}`, request_throughput=`{{ or .request_throughput "0" }}`, query_duration=`{{ .duration_seconds }}s` | query_kind=~"${query_type}" | query_duration > ${min_duration} | query_kind="id_lookup" or request_throughput < ${throughput}',

  // Search logs its range in seconds, TraceQL metrics in nanoseconds. Normalize both to seconds.
  local timeSpanLines = responseLines + ' | label_format time_span_seconds=`{{ if .range_nanos }}{{ div .range_nanos 1000000000 }}{{ else }}{{ .range_seconds }}{{ end }}`',

  // Generic per-request lines logged by the query-frontend HTTP handler. Only these carry the response size.
  // They have no query type and no throughput, so query_kind is extracted from the request URL and only
  // $min_duration applies. Trace diff requests are dropped.
  local queryStatsLines = selector + ' |= "query stats" | logfmt | url!~".*/traces/diff.*" | regexp `url="?\\S*/api/(v2/)?(?P<api_path>traces|search/tags?|search|metrics/query_range|metrics/query)` | label_format query_kind=`{{ if eq .api_path "traces" }}id_lookup{{ else if hasPrefix "search/tag" .api_path }}metadata{{ else if eq .api_path "search" }}search{{ else if eq .api_path "metrics/query_range" }}query_range{{ else if eq .api_path "metrics/query" }}instant_query{{ end }}` | query_kind!="" | tenant=~"${tenant_id}" | query_kind=~"${query_type}" | duration > ${min_duration} | response_size > 0',

  // Slow TraceQL metrics queries that hit the series limit and returned a partial response.
  local seriesLimitedLines = timeSpanLines + ' | partial_status="PARTIAL"',

  local quantiles(lines, field) = [
    'quantile_over_time(%s, %s | unwrap %s | __error__="" [$__auto]) by ()' % [quantile, lines, field]
    for quantile in ['0.99', '0.5']
  ],

  local topk(lines, field, label) =
    'topk(10, quantile_over_time(0.99, %s | unwrap %s | __error__="" [$__auto]) by (%s))' % [lines, field, label],

  local textboxVariable(name, label, defaultValue) = {
    type: 'textbox',
    name: name,
    label: label,
    hide: 0,
    options: [{ selected: true, text: defaultValue, value: defaultValue }],
    current: { selected: true, text: defaultValue, value: defaultValue },
    query: defaultValue,
  },

  local customVariable(name, label, values, includeAll=false) = {
    type: 'custom',
    name: name,
    label: label,
    hide: 0,
    multi: false,
    includeAll: includeAll,
    [if includeAll then 'allValue']: '.*',
    local first = if includeAll then { text: 'All', value: '$__all' } else values[0],
    current: first { selected: true },
    options: (if includeAll then [{ selected: true, text: 'All', value: '$__all' }] else []) + [
      v { selected: !includeAll && v == values[0] }
      for v in values
    ],
    query: std.join(', ', ['%s : %s' % [v.text, v.value] for v in values]),
  },

  local searchAndMetricsOnly = ' Only query types: search, query range, and instant query report this value.',

  // Measurements shown in each row, in display order.
  local measurements = [
    {
      title: 'Response time',
      lines: responseLines,
      field: 'duration_seconds',
      unit: 's',
      description: 'Time the query-frontend took to answer the query, from request to response.',
    },
    {
      title: 'Inspected spans',
      lines: responseLines,
      field: 'inspected_spans',
      unit: 'short',
      description: 'Number of spans the queriers evaluated to answer the query. A search with no span conditions reports zero.' + searchAndMetricsOnly,
    },
    {
      title: 'Inspected bytes',
      lines: responseLines,
      field: 'inspected_bytes',
      unit: 'bytes',
      description: 'Bytes the queriers read from blocks and live-stores to answer the query, including bytes served from the cache.',
    },
    {
      title: 'Response size',
      lines: queryStatsLines,
      field: 'response_size',
      unit: 'bytes',
      description: 'Size of the response returned to the client. Only the Min duration filter applies.',
    },
    {
      title: 'Time span',
      lines: timeSpanLines,
      field: 'time_span_seconds',
      unit: 's',
      description: 'Time range that the query covers, from its start time to its end time.',
    },
    {
      title: 'Jobs per query',
      lines: responseLines,
      field: 'total_requests',
      unit: 'short',
      description: 'Number of jobs the query-frontend split the query into. Each job runs on a querier, so more jobs means the query uses more of the read path.' + searchAndMetricsOnly,
    },
  ],

  local quantilesRow(title) = std.foldl(
    function(row, m)
      row.addPanel(
        $.timeseriesPanel(m.title) +
        $.lokiMetricsQueryPanel(quantiles(m.lines, m.field), ['p99', 'p50'], unit=m.unit) +
        $.panelDescription(m.title, m.description)
      ),
    measurements,
    g.row(title),
  ),

  local topkRow(title, label) = std.foldl(
    function(row, m)
      row.addPanel(
        $.timeseriesPanel('P99 ' + std.asciiLower(m.title)) +
        $.lokiMetricsQueryPanel(topk(m.lines, m.field, label), '{{%s}}' % label, unit=m.unit) +
        $.panelDescription(m.title, m.description)
      ),
    measurements,
    g.row(title) { collapse: true },
  ),

  local queriesTable(title, expr) = {
    height: '500px',
    title: title,
    type: 'table',
    datasource: '${loki_datasource}',

    targets: [
      {
        expr: expr,
        instant: false,
        legendFormat: '',
        range: true,
        refId: 'A',
      },
    ],

    local bytesFields = ['inspected_bytes', 'total_blockBytes'],
    local shortFields = ['inspected_spans', 'inspected_traces', 'total_blocks', 'total_requests', 'completed_requests', 'num_response_series', 'max_series', 'query_weight', 'query_sub_queries', 'query_conditions', 'query_regex_conditions'],
    local secondsFields = ['Time span', 'Duration'],

    // Use Grafana transformations to display fields in a table.
    transformations: [
      {
        // Convert labels to fields.
        id: 'extractFields',
        options: {
          source: 'labels',
        },
      },
      {
        id: 'organize',
        options: {
          // Hide stream labels and fields that duplicate other columns.
          local hiddenFields = ['app', 'caller', 'cluster', 'container', 'detected_level', 'id', 'job', 'labels', 'level', 'Line', 'msg', 'name', 'namespace', 'pod', 'pod_template_hash', 'query_duration', 'query_type', 'range', 'range_nanos', 'range_seconds', 'service_name', 'stream', 'Time', 'topology_kubernetes_io_region', 'topology_kubernetes_io_zone', 'tsNs'],

          excludeByName: {
            [field]: true
            for field in hiddenFields
          },

          // Order fields.
          local orderedFields = ['ts', 'tenant', 'query_kind', 'query', 'handler', 'tag', 'path', 'time_span_seconds', 'duration_seconds', 'request_throughput', 'inspected_bytes', 'inspected_spans', 'inspected_traces', 'total_blocks', 'total_requests', 'num_response_series', 'max_series', 'status_code', 'partial_status', 'error', 'traceID'],

          indexByName: {
            [orderedFields[i]]: i
            for i in std.range(0, std.length(orderedFields) - 1)
          },

          // Rename fields.
          renameByName: {
            ts: 'Completion date',
            tenant: 'Tenant ID',
            query_kind: 'Type',
            query: 'Query',
            time_span_seconds: 'Time span',
            duration_seconds: 'Duration',
            request_throughput: 'Throughput',
            status_code: 'Status',
            partial_status: 'Partial status',
            'error': 'Error',
          },
        },
      },
      {
        // Transform some fields into numbers so sorting in the table doesn't sort them lexicographically.
        id: 'convertFieldType',
        options: {
          local numericFields = ['Throughput'] + bytesFields + shortFields + secondsFields,

          conversions: [
            {
              targetField: fieldName,
              destinationType: 'number',
            }
            for fieldName in numericFields
          ],
        },
      },
    ],

    fieldConfig: {
      // Configure overrides to nicely format field values.
      overrides:
        [
          {
            matcher: { id: 'byName', options: fieldName },
            properties: [{ id: 'unit', value: 'bytes' }],
          }
          for fieldName in bytesFields
        ] +
        [
          {
            matcher: { id: 'byName', options: fieldName },
            properties: [{ id: 'unit', value: 'short' }],
          }
          for fieldName in shortFields
        ] +
        [
          {
            matcher: { id: 'byName', options: fieldName },
            properties: [{ id: 'unit', value: 's' }],
          }
          for fieldName in secondsFields
        ] +
        [
          {
            matcher: { id: 'byName', options: 'Throughput' },
            properties: [{ id: 'unit', value: 'Bps' }],
          },
        ],
    },
  },

  grafanaDashboards+: {
    [filename]:
      assert std.md5(filename) == '77983290ca4f07ded03bda8305b67013' : 'UID of the dashboard has changed, please update references to dashboard.';
      ($.dashboard('Tempo / Slow queries') + { uid: std.md5(filename) })
      .addClusterSelectorTemplates()
      .addRow(quantilesRow('Across tenants'))
      .addRow(topkRow('Top 10 tenants', 'tenant'))
      .addRow(
        g.row('Slow queries')
        .addPanel(queriesTable('Slow queries', timeSpanLines))
      )
      .addRow(
        g.row('Series-limited queries') { collapse: true }
        .addPanel(
          queriesTable('Series-limited queries', seriesLimitedLines) +
          $.panelDescription('Series-limited queries', 'Slow TraceQL metrics queries, range and instant, that produced more series than the max series limit allows. The query-frontend stops at the limit and returns a partial response, so the results of these queries are incomplete.')
        )
      )
      + {
        templating+: {
          list+: [
            {
              type: 'datasource',
              name: 'loki_datasource',
              label: 'Loki data source',
              query: 'loki',
              hide: 0,
              includeAll: false,
              multi: false,
            },
            textboxVariable('min_duration', 'Min duration', '5s'),
            // Queries at or above this throughput meet the query SLO, whatever their duration.
            customVariable('throughput', 'Throughput', [
              { text: '1 GB/s', value: '1e+09' },
              { text: '500 MB/s', value: '5e+08' },
              { text: 'off', value: '1e+18' },
            ]),
            textboxVariable('tenant_id', 'Tenant ID', '.*'),
            customVariable('query_type', 'Query type', [
              { text: 'ID lookups', value: 'id_lookup' },
              { text: 'search', value: 'search' },
              { text: 'metadata', value: 'metadata' },
              { text: 'query range', value: 'query_range' },
              { text: 'instant query', value: 'instant_query' },
            ], includeAll=true),
          ],
        },
      } + {
        templating+: {
          list: [
            // Do not allow to include all namespaces, otherwise the Loki queries scan every Tempo cell.
            l + (if (l.name == 'namespace') then { includeAll: false } else {})
            for l in super.list
          ],
        },
      },
  },
}
