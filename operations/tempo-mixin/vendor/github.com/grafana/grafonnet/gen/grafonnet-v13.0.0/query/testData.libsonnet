// This file is generated, do not manually edit.
{
  '#': { help: 'grafonnet.query.testData', name: 'testData' },
  '#withAlias': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
  withAlias(value): {
    alias: value,
  },
  '#withChannel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Used for live query' } },
  withChannel(value): {
    channel: value,
  },
  '#withCsvContent': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
  withCsvContent(value): {
    csvContent: value,
  },
  '#withCsvFileName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
  withCsvFileName(value): {
    csvFileName: value,
  },
  '#withCsvWave': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
  withCsvWave(value): {
    csvWave:
      (if std.isArray(value)
       then value
       else [value]),
  },
  '#withCsvWaveMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
  withCsvWaveMixin(value): {
    csvWave+:
      (if std.isArray(value)
       then value
       else [value]),
  },
  csvWave+:
    {
      '#': { help: '', name: 'csvWave' },
      '#withLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withLabels(value): {
        labels: value,
      },
      '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withName(value): {
        name: value,
      },
      '#withTimeStep': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
      withTimeStep(value): {
        timeStep: value,
      },
      '#withValuesCSV': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withValuesCSV(value): {
        valuesCSV: value,
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
  '#withDropPercent': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Drop percentage (the chance we will lose a point 0-100)' } },
  withDropPercent(value): {
    dropPercent: value,
  },
  '#withErrorSource': { 'function': { args: [{ default: null, enums: ['plugin', 'downstream'], name: 'value', type: ['string'] }], help: 'Possible enum values:\n - `"plugin"` \n - `"downstream"` ' } },
  withErrorSource(value): {
    errorSource: value,
  },
  '#withErrorType': { 'function': { args: [{ default: null, enums: ['frontend_exception', 'frontend_observable', 'server_panic'], name: 'value', type: ['string'] }], help: 'Possible enum values:\n - `"frontend_exception"` \n - `"frontend_observable"` \n - `"server_panic"` ' } },
  withErrorType(value): {
    errorType: value,
  },
  '#withFlamegraphDiff': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
  withFlamegraphDiff(value=true): {
    flamegraphDiff: value,
  },
  '#withHide': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'true if query is disabled (ie should not be returned to the dashboard)\nNOTE: this does not always imply that the query should not be executed since\nthe results from a hidden query may be used as the input to other queries (SSE etc)' } },
  withHide(value=true): {
    hide: value,
  },
  '#withIntervalMs': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Interval is the suggested duration between time points in a time series query.\nNOTE: the values for intervalMs is not saved in the query model.  It is typically calculated\nfrom the interval required to fill a pixels in the visualization' } },
  withIntervalMs(value): {
    intervalMs: value,
  },
  '#withLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
  withLabels(value): {
    labels: value,
  },
  '#withLevelColumn': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
  withLevelColumn(value=true): {
    levelColumn: value,
  },
  '#withLines': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
  withLines(value): {
    lines: value,
  },
  '#withMax': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
  withMax(value): {
    max: value,
  },
  '#withMaxDataPoints': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'MaxDataPoints is the maximum number of data points that should be returned from a time series query.\nNOTE: the values for maxDataPoints is not saved in the query model.  It is typically calculated\nfrom the number of pixels visible in a visualization' } },
  withMaxDataPoints(value): {
    maxDataPoints: value,
  },
  '#withMin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
  withMin(value): {
    min: value,
  },
  '#withNodes': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withNodes(value): {
    nodes: value,
  },
  '#withNodesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withNodesMixin(value): {
    nodes+: value,
  },
  nodes+:
    {
      '#withCount': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
      withCount(value): {
        nodes+: {
          count: value,
        },
      },
      '#withSeed': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
      withSeed(value): {
        nodes+: {
          seed: value,
        },
      },
      '#withType': { 'function': { args: [{ default: null, enums: ['random', 'random edges', 'response_medium', 'response_small', 'feature_showcase'], name: 'value', type: ['string'] }], help: 'Possible enum values:\n - `"random"` \n - `"random edges"` \n - `"response_medium"` \n - `"response_small"` \n - `"feature_showcase"` ' } },
      withType(value): {
        nodes+: {
          type: value,
        },
      },
    },
  '#withNoise': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
  withNoise(value): {
    noise: value,
  },
  '#withPoints': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
  withPoints(value): {
    points:
      (if std.isArray(value)
       then value
       else [value]),
  },
  '#withPointsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
  withPointsMixin(value): {
    points+:
      (if std.isArray(value)
       then value
       else [value]),
  },
  '#withPulseWave': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withPulseWave(value): {
    pulseWave: value,
  },
  '#withPulseWaveMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withPulseWaveMixin(value): {
    pulseWave+: value,
  },
  pulseWave+:
    {
      '#withOffCount': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
      withOffCount(value): {
        pulseWave+: {
          offCount: value,
        },
      },
      '#withOffValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
      withOffValue(value): {
        pulseWave+: {
          offValue: value,
        },
      },
      '#withOnCount': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
      withOnCount(value): {
        pulseWave+: {
          onCount: value,
        },
      },
      '#withOnValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
      withOnValue(value): {
        pulseWave+: {
          onValue: value,
        },
      },
      '#withTimeStep': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
      withTimeStep(value): {
        pulseWave+: {
          timeStep: value,
        },
      },
    },
  '#withQueryType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'QueryType is an optional identifier for the type of query.\nIt can be used to distinguish different types of queries.' } },
  withQueryType(value): {
    queryType: value,
  },
  '#withRawFrameContent': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
  withRawFrameContent(value): {
    rawFrameContent: value,
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
  '#withScenarioId': { 'function': { args: [{ default: null, enums: ['annotations', 'arrow', 'csv_content', 'csv_file', 'csv_metric_values', 'datapoints_outside_range', 'error_with_source', 'exponential_heatmap_bucket_data', 'flame_graph', 'grafana_api', 'linear_heatmap_bucket_data', 'live', 'logs', 'manual_entry', 'no_data_points', 'node_graph', 'predictable_csv_wave', 'predictable_pulse', 'random_walk', 'random_walk_table', 'random_walk_with_error', 'raw_frame', 'server_error_500', 'simulation', 'slow_query', 'streaming_client', 'table_static', 'trace', 'usa', 'variables-query'], name: 'value', type: ['string'] }], help: 'Possible enum values:\n - `"annotations"` \n - `"arrow"` \n - `"csv_content"` \n - `"csv_file"` \n - `"csv_metric_values"` \n - `"datapoints_outside_range"` \n - `"error_with_source"` \n - `"exponential_heatmap_bucket_data"` \n - `"flame_graph"` \n - `"grafana_api"` \n - `"linear_heatmap_bucket_data"` \n - `"live"` \n - `"logs"` \n - `"manual_entry"` \n - `"no_data_points"` \n - `"node_graph"` \n - `"predictable_csv_wave"` \n - `"predictable_pulse"` \n - `"random_walk"` \n - `"random_walk_table"` \n - `"random_walk_with_error"` \n - `"raw_frame"` \n - `"server_error_500"` \n - `"simulation"` \n - `"slow_query"` \n - `"streaming_client"` \n - `"table_static"` \n - `"trace"` \n - `"usa"` \n - `"variables-query"` ' } },
  withScenarioId(value): {
    scenarioId: value,
  },
  '#withSeriesCount': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
  withSeriesCount(value): {
    seriesCount: value,
  },
  '#withSim': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withSim(value): {
    sim: value,
  },
  '#withSimMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withSimMixin(value): {
    sim+: value,
  },
  sim+:
    {
      '#withConfig': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withConfig(value): {
        sim+: {
          config: value,
        },
      },
      '#withConfigMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withConfigMixin(value): {
        sim+: {
          config+: value,
        },
      },
      '#withKey': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withKey(value): {
        sim+: {
          key: value,
        },
      },
      '#withKeyMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withKeyMixin(value): {
        sim+: {
          key+: value,
        },
      },
      key+:
        {
          '#withTick': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
          withTick(value): {
            sim+: {
              key+: {
                tick: value,
              },
            },
          },
          '#withType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
          withType(value): {
            sim+: {
              key+: {
                type: value,
              },
            },
          },
          '#withUid': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
          withUid(value): {
            sim+: {
              key+: {
                uid: value,
              },
            },
          },
        },
      '#withLast': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
      withLast(value=true): {
        sim+: {
          last: value,
        },
      },
      '#withStream': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
      withStream(value=true): {
        sim+: {
          stream: value,
        },
      },
    },
  '#withSpanCount': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
  withSpanCount(value): {
    spanCount: value,
  },
  '#withSpread': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
  withSpread(value): {
    spread: value,
  },
  '#withStartValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
  withStartValue(value): {
    startValue: value,
  },
  '#withStream': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withStream(value): {
    stream: value,
  },
  '#withStreamMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withStreamMixin(value): {
    stream+: value,
  },
  stream+:
    {
      '#withBands': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
      withBands(value): {
        stream+: {
          bands: value,
        },
      },
      '#withNoise': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
      withNoise(value): {
        stream+: {
          noise: value,
        },
      },
      '#withSpeed': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
      withSpeed(value): {
        stream+: {
          speed: value,
        },
      },
      '#withSpread': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
      withSpread(value): {
        stream+: {
          spread: value,
        },
      },
      '#withType': { 'function': { args: [{ default: null, enums: ['fetch', 'logs', 'signal', 'traces'], name: 'value', type: ['string'] }], help: 'Possible enum values:\n - `"fetch"` \n - `"logs"` \n - `"signal"` \n - `"traces"` ' } },
      withType(value): {
        stream+: {
          type: value,
        },
      },
      '#withUrl': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withUrl(value): {
        stream+: {
          url: value,
        },
      },
    },
  '#withStringInput': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'common parameter used by many query types' } },
  withStringInput(value): {
    stringInput: value,
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
  '#withUsa': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withUsa(value): {
    usa: value,
  },
  '#withUsaMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withUsaMixin(value): {
    usa+: value,
  },
  usa+:
    {
      '#withFields': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withFields(value): {
        usa+: {
          fields:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withFieldsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withFieldsMixin(value): {
        usa+: {
          fields+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withMode': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withMode(value): {
        usa+: {
          mode: value,
        },
      },
      '#withPeriod': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withPeriod(value): {
        usa+: {
          period: value,
        },
      },
      '#withStates': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withStates(value): {
        usa+: {
          states:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withStatesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withStatesMixin(value): {
        usa+: {
          states+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
    },
  '#withWithNil': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
  withWithNil(value=true): {
    withNil: value,
  },
}
+ (import '../custom/query/testData.libsonnet')
