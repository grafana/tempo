// This file is generated, do not manually edit.
{
  '#': { help: 'grafonnet.query.prometheus', name: 'prometheus' },
  '#withAdhocFilters': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Additional Ad-hoc filters that take precedence over Scope on conflict.' } },
  withAdhocFilters(value): {
    adhocFilters:
      (if std.isArray(value)
       then value
       else [value]),
  },
  '#withAdhocFiltersMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Additional Ad-hoc filters that take precedence over Scope on conflict.' } },
  withAdhocFiltersMixin(value): {
    adhocFilters+:
      (if std.isArray(value)
       then value
       else [value]),
  },
  adhocFilters+:
    {
      '#': { help: '', name: 'adhocFilters' },
      '#withKey': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withKey(value): {
        key: value,
      },
      '#withOperator': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withOperator(value): {
        operator: value,
      },
      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withValue(value): {
        value: value,
      },
      '#withValues': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withValues(value): {
        values:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withValuesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withValuesMixin(value): {
        values+:
          (if std.isArray(value)
           then value
           else [value]),
      },
    },
  '#withDatasource': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The datasource' } },
  withDatasource(value): {
    datasource: value,
  },
  '#withDatasourceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The datasource' } },
  withDatasourceMixin(value): {
    datasource+: value,
  },
  datasource+:
    {
      '#withType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
      withType(value): {
        datasource+: {
          type: value,
        },
      },
      '#withUid': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Specific datasource instance' } },
      withUid(value): {
        datasource+: {
          uid: value,
        },
      },
    },
  '#withEditorMode': { 'function': { args: [{ default: null, enums: ['builder', 'code'], name: 'value', type: ['string'] }], help: 'what we should show in the editor\nPossible enum values:\n - `"builder"` \n - `"code"` ' } },
  withEditorMode(value): {
    editorMode: value,
  },
  '#withExemplar': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Execute an additional query to identify interesting raw samples relevant for the given expr' } },
  withExemplar(value=true): {
    exemplar: value,
  },
  '#withExpr': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The actual expression/query that will be evaluated by Prometheus' } },
  withExpr(value): {
    expr: value,
  },
  '#withFormat': { 'function': { args: [{ default: null, enums: ['time_series', 'table', 'heatmap'], name: 'value', type: ['string'] }], help: 'The response format\nPossible enum values:\n - `"time_series"` \n - `"table"` \n - `"heatmap"` ' } },
  withFormat(value): {
    format: value,
  },
  '#withGroupByKeys': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Group By parameters to apply to aggregate expressions in the query' } },
  withGroupByKeys(value): {
    groupByKeys:
      (if std.isArray(value)
       then value
       else [value]),
  },
  '#withGroupByKeysMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Group By parameters to apply to aggregate expressions in the query' } },
  withGroupByKeysMixin(value): {
    groupByKeys+:
      (if std.isArray(value)
       then value
       else [value]),
  },
  '#withHide': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'true if query is disabled (ie should not be returned to the dashboard)\nNOTE: this does not always imply that the query should not be executed since\nthe results from a hidden query may be used as the input to other queries (SSE etc)' } },
  withHide(value=true): {
    hide: value,
  },
  '#withInstant': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Returns only the latest value that Prometheus has scraped for the requested time series' } },
  withInstant(value=true): {
    instant: value,
  },
  '#withInterval': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'An additional lower limit for the step parameter of the Prometheus query and for the\n`$__interval` and `$__rate_interval` variables.' } },
  withInterval(value): {
    interval: value,
  },
  '#withIntervalFactor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'Used to specify how many times to divide max data points by. We use max data points under query options\nSee https://github.com/grafana/grafana/issues/48081\nDeprecated: use interval' } },
  withIntervalFactor(value): {
    intervalFactor: value,
  },
  '#withIntervalMs': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Interval is the suggested duration between time points in a time series query.\nNOTE: the values for intervalMs is not saved in the query model.  It is typically calculated\nfrom the interval required to fill a pixels in the visualization' } },
  withIntervalMs(value): {
    intervalMs: value,
  },
  '#withLegendFormat': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Series name override or template. Ex. {{hostname}} will be replaced with label value for hostname' } },
  withLegendFormat(value): {
    legendFormat: value,
  },
  '#withMaxDataPoints': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'MaxDataPoints is the maximum number of data points that should be returned from a time series query.\nNOTE: the values for maxDataPoints is not saved in the query model.  It is typically calculated\nfrom the number of pixels visible in a visualization' } },
  withMaxDataPoints(value): {
    maxDataPoints: value,
  },
  '#withQueryType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'QueryType is an optional identifier for the type of query.\nIt can be used to distinguish different types of queries.' } },
  withQueryType(value): {
    queryType: value,
  },
  '#withRange': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Returns a Range vector, comprised of a set of time series containing a range of data points over time for each time series' } },
  withRange(value=true): {
    range: value,
  },
  '#withRefId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'RefID is the unique identifier of the query, set by the frontend call.' } },
  withRefId(value): {
    refId: value,
  },
  '#withResultAssertions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Optionally define expected query result behavior' } },
  withResultAssertions(value): {
    resultAssertions: value,
  },
  '#withResultAssertionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Optionally define expected query result behavior' } },
  withResultAssertionsMixin(value): {
    resultAssertions+: value,
  },
  resultAssertions+:
    {
      '#withMaxFrames': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'Maximum frame count' } },
      withMaxFrames(value): {
        resultAssertions+: {
          maxFrames: value,
        },
      },
      '#withType': { 'function': { args: [{ default: null, enums: ['', 'timeseries-wide', 'timeseries-long', 'timeseries-many', 'timeseries-multi', 'directory-listing', 'table', 'numeric-wide', 'numeric-multi', 'numeric-long', 'log-lines'], name: 'value', type: ['string'] }], help: 'Type asserts that the frame matches a known type structure.\nPossible enum values:\n - `""` \n - `"timeseries-wide"` \n - `"timeseries-long"` \n - `"timeseries-many"` \n - `"timeseries-multi"` \n - `"directory-listing"` \n - `"table"` \n - `"numeric-wide"` \n - `"numeric-multi"` \n - `"numeric-long"` \n - `"log-lines"` ' } },
      withType(value): {
        resultAssertions+: {
          type: value,
        },
      },
      '#withTypeVersion': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'TypeVersion is the version of the Type property. Versions greater than 0.0 correspond to the dataplane\ncontract documentation https://grafana.github.io/dataplane/contract/.' } },
      withTypeVersion(value): {
        resultAssertions+: {
          typeVersion:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withTypeVersionMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'TypeVersion is the version of the Type property. Versions greater than 0.0 correspond to the dataplane\ncontract documentation https://grafana.github.io/dataplane/contract/.' } },
      withTypeVersionMixin(value): {
        resultAssertions+: {
          typeVersion+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
    },
  '#withScopes': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'A set of filters applied to apply to the query' } },
  withScopes(value): {
    scopes:
      (if std.isArray(value)
       then value
       else [value]),
  },
  '#withScopesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'A set of filters applied to apply to the query' } },
  withScopesMixin(value): {
    scopes+:
      (if std.isArray(value)
       then value
       else [value]),
  },
  scopes+:
    {
      '#': { help: '', name: 'scopes' },
      '#withDefaultPath': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withDefaultPath(value): {
        defaultPath:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withDefaultPathMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withDefaultPathMixin(value): {
        defaultPath+:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withFilters': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withFilters(value): {
        filters:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withFiltersMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withFiltersMixin(value): {
        filters+:
          (if std.isArray(value)
           then value
           else [value]),
      },
      filters+:
        {
          '#': { help: '', name: 'filters' },
          '#withKey': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
          withKey(value): {
            key: value,
          },
          '#withOperator': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
          withOperator(value): {
            operator: value,
          },
          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
          withValue(value): {
            value: value,
          },
          '#withValues': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
          withValues(value): {
            values:
              (if std.isArray(value)
               then value
               else [value]),
          },
          '#withValuesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
          withValuesMixin(value): {
            values+:
              (if std.isArray(value)
               then value
               else [value]),
          },
        },
      '#withTitle': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withTitle(value): {
        title: value,
      },
    },
  '#withTimeRange': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TimeRange represents the query range\nNOTE: unlike generic /ds/query, we can now send explicit time values in each query\nNOTE: the values for timeRange are not saved in a dashboard, they are constructed on the fly' } },
  withTimeRange(value): {
    timeRange: value,
  },
  '#withTimeRangeMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TimeRange represents the query range\nNOTE: unlike generic /ds/query, we can now send explicit time values in each query\nNOTE: the values for timeRange are not saved in a dashboard, they are constructed on the fly' } },
  withTimeRangeMixin(value): {
    timeRange+: value,
  },
  timeRange+:
    {
      '#withFrom': { 'function': { args: [{ default: 'now-6h', enums: null, name: 'value', type: ['string'] }], help: 'From is the start time of the query.' } },
      withFrom(value='now-6h'): {
        timeRange+: {
          from: value,
        },
      },
      '#withTo': { 'function': { args: [{ default: 'now', enums: null, name: 'value', type: ['string'] }], help: 'To is the end time of the query.' } },
      withTo(value='now'): {
        timeRange+: {
          to: value,
        },
      },
    },
}
+ (import '../custom/query/prometheus.libsonnet')
