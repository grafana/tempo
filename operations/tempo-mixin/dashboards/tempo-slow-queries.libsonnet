local dashboard_utils = import 'dashboard-utils.libsonnet';
local g = import 'grafana-builder/grafana.libsonnet';
local filename = 'tempo-slow-queries.json';

dashboard_utils {
  local selector = '{%s=~"$cluster", namespace=~"$namespace", %s=~"%s"}' % [$._config.per_cluster_label, $._config.per_component_loki_label, $._config.jobs.query_frontend],

  // Per-query response lines logged by the query-frontend for trace by ID, search, tag, and TraceQL metrics
  // queries, including the "- no resp" and "- no metrics" variants. query_kind names the query type.
  // Trace by ID logs its error as err, the other query types as error. Copy err to error so the table shows one Error column.
  // A query without an error logs null, which is dropped.
  // A query is slow when it is slower than $min_duration and its throughput is below $throughput, the same as
  // the query SLO. Trace by ID has no throughput SLO. Lines without a throughput, such as failed queries, count as 0.
  local responseLines = selector + ' |~ `msg="(trace id response|search tag response|search tag values response|search response|query range response|query instant results)( - no (resp|metrics))?"` | logfmt | tenant=~"${tenant_id}" | label_format query_kind=`{{ if hasPrefix "trace id" .msg }}id_lookup{{ else if hasPrefix "search tag" .msg }}metadata{{ else if hasPrefix "search" .msg }}search{{ else if hasPrefix "query range" .msg }}query_range{{ else }}instant_query{{ end }}`, request_throughput=`{{ or .request_throughput "0" }}`, error=`{{ if ne .error "null" }}{{ .error }}{{ end }}{{ if ne .err "null" }}{{ .err }}{{ end }}`, query_duration=`{{ .duration_seconds }}s` | query_kind=~"${query_type}" | query_duration > ${min_duration} | query_kind="id_lookup" or request_throughput < ${throughput}',

  // Search logs its range in seconds, TraceQL metrics in nanoseconds. Normalize both to seconds.
  local timeSpanLines = responseLines + ' | label_format time_span_seconds=`{{ if .range_nanos }}{{ div .range_nanos 1000000000 }}{{ else }}{{ .range_seconds }}{{ end }}`',

  // Generic per-request lines logged by the query-frontend HTTP handler. Only these carry the response size.
  // They have no query type and no throughput, so query_kind is extracted from the request URL and only
  // $min_duration applies. Trace diff requests are dropped. The URL path ends at "?", so the query string is not matched.
  local queryStatsLines = selector + ' |= "query stats" | logfmt | url!~".*/traces/diff.*" | regexp `url="?[^?\\s"]*/api/(v2/)?(?P<api_path>traces|search/tags?|search|metrics/query_range|metrics/query)` | label_format query_kind=`{{ if eq .api_path "traces" }}id_lookup{{ else if hasPrefix "search/tag" .api_path }}metadata{{ else if eq .api_path "search" }}search{{ else if eq .api_path "metrics/query_range" }}query_range{{ else if eq .api_path "metrics/query" }}instant_query{{ end }}` | query_kind!="" | tenant=~"${tenant_id}" | query_kind=~"${query_type}" | duration > ${min_duration} | response_size > 0',

  // Slow TraceQL metrics queries near the series limit, the same as the Tempo slow customer searches dashboard:
  // the query-frontend stopped at the limit and returned a partial response, or the query finished with at least
  // query_frontend_max_response_series series. A tenant with a higher limit can finish with more series.
  local seriesLimitedLines = timeSpanLines + (
    if $._config.query_frontend_max_response_series > 0
    then ' | partial_status="PARTIAL" or num_response_series >= %d' % $._config.query_frontend_max_response_series
    else ' | partial_status="PARTIAL"'
  ),

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

  // A table column: the log field to keep, its header, and optionally a unit, a width in pixels, and data links.
  // A column with a unit holds a number. Query and Error can be long, so their cells open in an inspect view.
  local column(field, name, unit=null, width=null, links=[]) = { field: field, name: name, unit: unit, width: width, links: links },

  // ts is the time the query-frontend logged the response, so when the query finished. It is a label, because the
  // table keeps only labels, and the log timestamp field has a different name in each Grafana version.
  local timeColumn = column('ts', 'time', width=160),

  // Explore URL that opens the trace ID in the cell with the Tempo data source that holds Tempo's own traces,
  // over the dashboard time range. The panes value is URL-encoded JSON. The ${...} variables stay as they are,
  // so Grafana fills them in when the link opens.
  local traceExploreUrl =
    local escapes = { '{': '%7B', '}': '%7D', '"': '%22', '[': '%5B', ']': '%5D', ':': '%3A', ',': '%2C' };
    local panes = std.manifestJsonMinified({
      trace: {
        datasource: $._config.tempo_datasource_uid,
        queries: [{
          refId: 'A',
          datasource: { type: 'tempo', uid: $._config.tempo_datasource_uid },
          queryType: 'traceql',
          query: '@trace_id@',
        }],
        range: { from: '@from@', to: '@to@' },
      },
    });
    local encoded = std.join('', [
      if std.objectHas(escapes, c) then escapes[c] else c
      for c in std.stringChars(panes)
    ]);
    std.foldl(
      function(url, v) std.strReplace(url, v[0], v[1]),
      [['@trace_id@', '${__value.raw}'], ['@from@', '${__from}'], ['@to@', '${__to}']],
      '/explore?schemaVersion=1&panes=' + encoded,
    ),

  // Columns follow the Tempo slow customer searches dashboard: who ran the query and what it cost, then the
  // query with its trace ID exemplar next to it. Query has no fixed width, so it takes the remaining width.
  local leadingColumns = [
    timeColumn,
    column($._config.per_cluster_label, 'cluster', width=100),
    column('tenant', 'tenant id', width=90),
    column('query_kind', 'type', width=100),
  ],
  local traceIdColumn = column('traceID', 'trace id', width=250, links=if $._config.tempo_datasource_uid != '' then [
    { title: 'View trace', url: traceExploreUrl, targetBlank: true },
  ] else []),
  local queryColumn = column('query', 'query'),

  // error and tag come last, because only some queries fill them.
  local slowQueryColumns = leadingColumns + [
    column('duration_seconds', 'duration', 's', 80),
    column('inspected_bytes', 'inspected', 'bytes', 100),
    column('request_throughput', 'throughput', 'Bps', 100),
    column('time_span_seconds', 'range', 's', 90),
    queryColumn,
    traceIdColumn,
    column('error', 'error', width=120),
    column('tag', 'tag', width=100),
  ],

  // Same order as the series-limited table of the Tempo slow customer searches dashboard.
  local seriesLimitedColumns = leadingColumns + [
    column('num_response_series', 'series', 'none', 80),
    column('partial_status', 'partial status', width=110),
    column('duration_seconds', 'duration', 's', 80),
    column('time_span_seconds', 'range', 's', 90),
    queryColumn,
    traceIdColumn,
  ],

  local queriesTable(title, lines, columns) = {
    height: '500px',
    title: title,
    type: 'table',
    datasource: '${loki_datasource}',

    targets: [
      {
        // Drop lines whose filters failed to parse, such as a duration in scientific notation, which Loki keeps otherwise.
        // Trace by ID has no query, so show the request path, which holds the trace ID it looked up.
        // keep drops every other label, so the table shows only these columns. drop removes empty labels, so a column
        // that only some query types fill, such as range or tag, is shown only when a query in the table has a value.
        expr: lines + ' | __error__="" | label_format query=`{{ or .query .path }}` | keep ' + std.join(', ', [c.field for c in columns]) +
              ' | drop ' + std.join(', ', ['%s=""' % c.field for c in columns]),
        instant: false,
        legendFormat: '',
        range: true,
        refId: 'A',
      },
    ],

    local numericColumns = [c for c in columns if c.unit != null],

    // Use Grafana transformations to display fields in a table.
    transformations: [
      {
        // Convert labels to fields, and drop the log line and the other fields.
        id: 'extractFields',
        options: {
          source: 'labels',
          replace: true,
        },
      },
      {
        id: 'organize',
        options: {
          indexByName: {
            [columns[i].field]: i
            for i in std.range(0, std.length(columns) - 1)
          },
          renameByName: {
            [c.field]: c.name
            for c in columns
          },
        },
      },
      {
        // Transform the time and the numbers so sorting in the table doesn't sort them lexicographically.
        id: 'convertFieldType',
        options: {
          conversions: [{ targetField: timeColumn.name, destinationType: 'time' }] + [
            {
              targetField: c.name,
              destinationType: 'number',
            }
            for c in numericColumns
          ],
        },
      },
    ],

    fieldConfig: {
      // Give the short columns a fixed width, so the query and its trace ID fit without scrolling.
      overrides:
        [
          {
            matcher: { id: 'byName', options: c.name },
            properties:
              (if c.unit != null then [{ id: 'unit', value: c.unit }] else []) +
              (if c.width != null then [{ id: 'custom.width', value: c.width }] else []) +
              (if c.field == 'query' || c.field == 'error' then [{ id: 'custom.inspect', value: true }] else []) +
              (if c.links != [] then [{ id: 'links', value: c.links }] else []),
          }
          for c in columns
        ],
    },
  },

  local seriesLimitedDescription =
    'Slow TraceQL metrics queries, range and instant, that hit the max series limit. With partial status PARTIAL, the query-frontend stopped at the limit and returned a partial response, so the results are incomplete.' +
    (if $._config.query_frontend_max_response_series > 0 then ' With COMPLETE, the query finished but returned at least %d series, the default limit. A tenant with a higher limit can return more series without a partial response.' % $._config.query_frontend_max_response_series else ''),

  // Number of slow queries in each interval, per query type, to show when they happen. Lines whose filters
  // failed to parse are dropped, the same as in the tables.
  // Slow queries are sparse, and Loki returns no point for an interval without one, so a line can't join a lone
  // point to anything and it isn't drawn. Stacked bars from 0 show every interval that has a slow query.
  // Intervals are at least 1m, so each bar is wide enough to see.
  local countPanel(title, lines, description) =
    $.timeseriesPanel(title) +
    $.lokiMetricsQueryPanel('sum by (query_kind) (count_over_time(%s | __error__="" [$__auto]))' % lines, '{{query_kind}}') +
    $.panelDescription(title, description) +
    {
      height: '250px',
      interval: '1m',
      fieldConfig+: {
        defaults+: {
          min: 0,
          decimals: 0,
          custom+: {
            drawStyle: 'bars',
            fillOpacity: 80,
            stacking+: { mode: 'normal' },
          },
        },
      },
      options+: { tooltip+: { mode: 'multi' } },
    },

  // A row with the time series above the table. grafana-builder splits a row's width between its panels,
  // so both panels are set to the full width, and Grafana stacks them.
  local stackedRow(title, panels) = g.row(title) {
    panels: [p { span: 12 } for p in panels],
  },

  grafanaDashboards+: {
    [filename]:
      assert std.md5(filename) == '77983290ca4f07ded03bda8305b67013' : 'UID of the dashboard has changed, please update references to dashboard.';
      ($.dashboard('Tempo / Slow queries') + { uid: std.md5(filename) })
      .addClusterSelectorTemplates()
      .addRow(quantilesRow('Across tenants'))
      .addRow(topkRow('Top 10 tenants', 'tenant'))
      .addRow(
        stackedRow('Slow queries', [
          countPanel('Slow queries over time', responseLines, 'Number of slow queries in each interval, by query type. The table below lists them.'),
          queriesTable('Slow queries', timeSpanLines, slowQueryColumns),
        ])
      )
      .addRow(
        stackedRow('Series-limited queries', [
          countPanel('Series-limited queries over time', seriesLimitedLines, 'Number of series-limited queries in each interval, by query type. The table below lists them.'),
          queriesTable('Series-limited queries', seriesLimitedLines, seriesLimitedColumns) +
          $.panelDescription('Series-limited queries', seriesLimitedDescription),
        ]) { collapse: true }
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
