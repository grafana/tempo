local dashboard_utils = import 'dashboard-utils.libsonnet';
local g = import 'grafana-builder/grafana.libsonnet';
local filename = 'tempo-slow-queries.json';

dashboard_utils {
  local selector = '{%s=~"$cluster", namespace=~"$namespace", %s=~"%s"}' % [$._config.per_cluster_label, $._config.per_component_loki_label, $._config.jobs.query_frontend],

  // Per-query response lines logged by the search and TraceQL metrics handlers in the query-frontend.
  // duration_seconds is a plain number, so it is converted to a duration to compare it with $min_duration.
  local responseLines = selector + ' |~ `msg="(search response|query range response|query instant results)"` | logfmt | tenant=~"${tenant_id}" | query_type=~"${query_type}" | label_format query_duration=`{{ .duration_seconds }}s` | query_duration > ${min_duration}',

  // Search logs its range in seconds, TraceQL metrics in nanoseconds. Normalize both to seconds.
  local timeSpanLines = responseLines + ' | label_format time_span_seconds=`{{ if .range_nanos }}{{ div .range_nanos 1000000000 }}{{ else }}{{ .range_seconds }}{{ end }}`',

  // Generic per-request lines logged by the query-frontend HTTP handler. Only these carry the response size.
  // They have no query_type field, so it is extracted from the request URL. Other requests, such as
  // trace by ID and tag lookups, are dropped to match the response lines.
  local queryStatsLines = selector + ' |= "query stats" | logfmt | regexp `url="?\\S*/api/(v2/)?(?P<query_type>search|metrics)` | query_type!="" | url!~".*/search/tag.*" | tenant=~"${tenant_id}" | query_type=~"${query_type}" | duration > ${min_duration} | response_size > 0',

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
      description: 'Number of spans the queriers evaluated to answer the query. A search with no span conditions reports zero.',
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
      description: 'Size of the response returned to the client.',
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
      description: 'Number of jobs the query-frontend split the query into. Each job runs on a querier, so more jobs means the query uses more of the read path.',
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

  grafanaDashboards+: {
    [filename]:
      assert std.md5(filename) == '77983290ca4f07ded03bda8305b67013' : 'UID of the dashboard has changed, please update references to dashboard.';
      ($.dashboard('Tempo / Slow queries') + { uid: std.md5(filename) })
      .addClusterSelectorTemplates()
      .addRow(quantilesRow('Across tenants'))
      .addRow(topkRow('Top 10 tenants', 'tenant'))
      .addRow(
        g.row('')
        .addPanel(
          {
            height: '500px',
            title: 'Slow queries',
            type: 'table',
            datasource: '${loki_datasource}',

            targets: [
              {
                expr: timeSpanLines,
                instant: false,
                legendFormat: '',
                range: true,
                refId: 'A',
              },
            ],

            local bytesFields = ['inspected_bytes', 'total_blockBytes'],
            local shortFields = ['inspected_spans', 'inspected_traces', 'total_blocks', 'total_requests', 'completed_requests', 'num_response_series', 'query_weight', 'query_sub_queries', 'query_conditions', 'query_regex_conditions'],
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
                  local hiddenFields = ['app', 'caller', 'cluster', 'container', 'detected_level', 'id', 'job', 'labels', 'level', 'Line', 'msg', 'name', 'namespace', 'pod', 'pod_template_hash', 'query_duration', 'range_nanos', 'range_seconds', 'service_name', 'stream', 'Time', 'topology_kubernetes_io_region', 'topology_kubernetes_io_zone', 'tsNs'],

                  excludeByName: {
                    [field]: true
                    for field in hiddenFields
                  },

                  // Order fields.
                  local orderedFields = ['ts', 'tenant', 'query_type', 'query', 'time_span_seconds', 'duration_seconds', 'inspected_bytes', 'inspected_spans', 'inspected_traces', 'total_blocks', 'total_requests', 'status_code', 'partial_status', 'error', 'traceID'],

                  indexByName: {
                    [orderedFields[i]]: i
                    for i in std.range(0, std.length(orderedFields) - 1)
                  },

                  // Rename fields.
                  renameByName: {
                    ts: 'Completion date',
                    tenant: 'Tenant ID',
                    query_type: 'Type',
                    query: 'Query',
                    time_span_seconds: 'Time span',
                    duration_seconds: 'Duration',
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
                  local numericFields = ['request_throughput'] + bytesFields + shortFields + secondsFields,

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
                    matcher: { id: 'byName', options: 'request_throughput' },
                    properties: [{ id: 'unit', value: 'Bps' }],
                  },
                ],
            },
          },
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
            textboxVariable('tenant_id', 'Tenant ID', '.*'),
            {
              type: 'custom',
              name: 'query_type',
              label: 'Query type',
              multi: false,
              includeAll: true,
              allValue: '.*',
              current: { selected: true, text: 'All', value: '$__all' },
              options: [
                { selected: true, text: 'All', value: '$__all' },
                { selected: false, text: 'search', value: 'search' },
                { selected: false, text: 'metrics', value: 'metrics' },
              ],
              query: 'search, metrics',
            },
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
