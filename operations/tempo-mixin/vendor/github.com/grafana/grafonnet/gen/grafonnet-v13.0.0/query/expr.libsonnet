// This file is generated, do not manually edit.
{
  '#': { help: 'grafonnet.query.expr', name: 'expr' },
  TypeMath+:
    {
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
      '#withExpression': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'General math expression' } },
      withExpression(value): {
        expression: value,
      },
      '#withHide': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'true if query is disabled (ie should not be returned to the dashboard)\nNOTE: this does not always imply that the query should not be executed since\nthe results from a hidden query may be used as the input to other queries (SSE etc)' } },
      withHide(value=true): {
        hide: value,
      },
      '#withIntervalMs': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Interval is the suggested duration between time points in a time series query.\nNOTE: the values for intervalMs is not saved in the query model.  It is typically calculated\nfrom the interval required to fill a pixels in the visualization' } },
      withIntervalMs(value): {
        intervalMs: value,
      },
      '#withMaxDataPoints': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'MaxDataPoints is the maximum number of data points that should be returned from a time series query.\nNOTE: the values for maxDataPoints is not saved in the query model.  It is typically calculated\nfrom the number of pixels visible in a visualization' } },
      withMaxDataPoints(value): {
        maxDataPoints: value,
      },
      '#withQueryType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'QueryType is an optional identifier for the type of query.\nIt can be used to distinguish different types of queries.' } },
      withQueryType(value): {
        queryType: value,
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
      '#withType': { 'function': { args: [], help: '' } },
      withType(): {
        type: 'math',
      },
    },
  TypeReduce+:
    {
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
      '#withExpression': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Reference to single query result' } },
      withExpression(value): {
        expression: value,
      },
      '#withHide': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'true if query is disabled (ie should not be returned to the dashboard)\nNOTE: this does not always imply that the query should not be executed since\nthe results from a hidden query may be used as the input to other queries (SSE etc)' } },
      withHide(value=true): {
        hide: value,
      },
      '#withIntervalMs': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Interval is the suggested duration between time points in a time series query.\nNOTE: the values for intervalMs is not saved in the query model.  It is typically calculated\nfrom the interval required to fill a pixels in the visualization' } },
      withIntervalMs(value): {
        intervalMs: value,
      },
      '#withMaxDataPoints': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'MaxDataPoints is the maximum number of data points that should be returned from a time series query.\nNOTE: the values for maxDataPoints is not saved in the query model.  It is typically calculated\nfrom the number of pixels visible in a visualization' } },
      withMaxDataPoints(value): {
        maxDataPoints: value,
      },
      '#withQueryType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'QueryType is an optional identifier for the type of query.\nIt can be used to distinguish different types of queries.' } },
      withQueryType(value): {
        queryType: value,
      },
      '#withReducer': { 'function': { args: [{ default: null, enums: ['sum', 'mean', 'min', 'max', 'count', 'last', 'median'], name: 'value', type: ['string'] }], help: 'The reducer\nPossible enum values:\n - `"sum"` \n - `"mean"` \n - `"min"` \n - `"max"` \n - `"count"` \n - `"last"` \n - `"median"` ' } },
      withReducer(value): {
        reducer: value,
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
      '#withSettings': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Reducer Options' } },
      withSettings(value): {
        settings: value,
      },
      '#withSettingsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Reducer Options' } },
      withSettingsMixin(value): {
        settings+: value,
      },
      settings+:
        {
          '#withMode': { 'function': { args: [{ default: null, enums: ['dropNN', 'replaceNN'], name: 'value', type: ['string'] }], help: 'Non-number reduce behavior\nPossible enum values:\n - `"dropNN"` Drop non-numbers\n - `"replaceNN"` Replace non-numbers' } },
          withMode(value): {
            settings+: {
              mode: value,
            },
          },
          '#withReplaceWithValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Only valid when mode is replace' } },
          withReplaceWithValue(value): {
            settings+: {
              replaceWithValue: value,
            },
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
      '#withType': { 'function': { args: [], help: '' } },
      withType(): {
        type: 'reduce',
      },
    },
  TypeResample+:
    {
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
      '#withDownsampler': { 'function': { args: [{ default: null, enums: ['sum', 'mean', 'min', 'max', 'count', 'last', 'median'], name: 'value', type: ['string'] }], help: 'The downsample function\nPossible enum values:\n - `"sum"` \n - `"mean"` \n - `"min"` \n - `"max"` \n - `"count"` \n - `"last"` \n - `"median"` ' } },
      withDownsampler(value): {
        downsampler: value,
      },
      '#withExpression': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The math expression' } },
      withExpression(value): {
        expression: value,
      },
      '#withHide': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'true if query is disabled (ie should not be returned to the dashboard)\nNOTE: this does not always imply that the query should not be executed since\nthe results from a hidden query may be used as the input to other queries (SSE etc)' } },
      withHide(value=true): {
        hide: value,
      },
      '#withIntervalMs': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Interval is the suggested duration between time points in a time series query.\nNOTE: the values for intervalMs is not saved in the query model.  It is typically calculated\nfrom the interval required to fill a pixels in the visualization' } },
      withIntervalMs(value): {
        intervalMs: value,
      },
      '#withMaxDataPoints': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'MaxDataPoints is the maximum number of data points that should be returned from a time series query.\nNOTE: the values for maxDataPoints is not saved in the query model.  It is typically calculated\nfrom the number of pixels visible in a visualization' } },
      withMaxDataPoints(value): {
        maxDataPoints: value,
      },
      '#withQueryType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'QueryType is an optional identifier for the type of query.\nIt can be used to distinguish different types of queries.' } },
      withQueryType(value): {
        queryType: value,
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
      '#withType': { 'function': { args: [], help: '' } },
      withType(): {
        type: 'resample',
      },
      '#withUpsampler': { 'function': { args: [{ default: null, enums: ['pad', 'backfilling', 'fillna'], name: 'value', type: ['string'] }], help: 'The upsample function\nPossible enum values:\n - `"pad"` Use the last seen value\n - `"backfilling"` backfill\n - `"fillna"` Do not fill values (nill)' } },
      withUpsampler(value): {
        upsampler: value,
      },
      '#withWindow': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The time duration' } },
      withWindow(value): {
        window: value,
      },
    },
  TypeClassicConditions+:
    {
      '#withConditions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withConditions(value): {
        conditions:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withConditionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withConditionsMixin(value): {
        conditions+:
          (if std.isArray(value)
           then value
           else [value]),
      },
      conditions+:
        {
          '#': { help: '', name: 'conditions' },
          '#withEvaluator': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withEvaluator(value): {
            evaluator: value,
          },
          '#withEvaluatorMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withEvaluatorMixin(value): {
            evaluator+: value,
          },
          evaluator+:
            {
              '#withParams': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withParams(value): {
                evaluator+: {
                  params:
                    (if std.isArray(value)
                     then value
                     else [value]),
                },
              },
              '#withParamsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withParamsMixin(value): {
                evaluator+: {
                  params+:
                    (if std.isArray(value)
                     then value
                     else [value]),
                },
              },
              '#withType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'e.g. "gt"' } },
              withType(value): {
                evaluator+: {
                  type: value,
                },
              },
            },
          '#withOperator': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withOperator(value): {
            operator: value,
          },
          '#withOperatorMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withOperatorMixin(value): {
            operator+: value,
          },
          operator+:
            {
              '#withType': { 'function': { args: [{ default: null, enums: ['and', 'or', 'logic-or'], name: 'value', type: ['string'] }], help: '' } },
              withType(value): {
                operator+: {
                  type: value,
                },
              },
            },
          '#withQuery': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withQuery(value): {
            query: value,
          },
          '#withQueryMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withQueryMixin(value): {
            query+: value,
          },
          query+:
            {
              '#withParams': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withParams(value): {
                query+: {
                  params:
                    (if std.isArray(value)
                     then value
                     else [value]),
                },
              },
              '#withParamsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withParamsMixin(value): {
                query+: {
                  params+:
                    (if std.isArray(value)
                     then value
                     else [value]),
                },
              },
            },
          '#withReducer': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withReducer(value): {
            reducer: value,
          },
          '#withReducerMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withReducerMixin(value): {
            reducer+: value,
          },
          reducer+:
            {
              '#withType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
              withType(value): {
                reducer+: {
                  type: value,
                },
              },
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
      '#withHide': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'true if query is disabled (ie should not be returned to the dashboard)\nNOTE: this does not always imply that the query should not be executed since\nthe results from a hidden query may be used as the input to other queries (SSE etc)' } },
      withHide(value=true): {
        hide: value,
      },
      '#withIntervalMs': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Interval is the suggested duration between time points in a time series query.\nNOTE: the values for intervalMs is not saved in the query model.  It is typically calculated\nfrom the interval required to fill a pixels in the visualization' } },
      withIntervalMs(value): {
        intervalMs: value,
      },
      '#withMaxDataPoints': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'MaxDataPoints is the maximum number of data points that should be returned from a time series query.\nNOTE: the values for maxDataPoints is not saved in the query model.  It is typically calculated\nfrom the number of pixels visible in a visualization' } },
      withMaxDataPoints(value): {
        maxDataPoints: value,
      },
      '#withQueryType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'QueryType is an optional identifier for the type of query.\nIt can be used to distinguish different types of queries.' } },
      withQueryType(value): {
        queryType: value,
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
      '#withType': { 'function': { args: [], help: '' } },
      withType(): {
        type: 'classic_conditions',
      },
    },
  TypeThreshold+:
    {
      '#withConditions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Threshold Conditions' } },
      withConditions(value): {
        conditions:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withConditionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Threshold Conditions' } },
      withConditionsMixin(value): {
        conditions+:
          (if std.isArray(value)
           then value
           else [value]),
      },
      conditions+:
        {
          '#': { help: '', name: 'conditions' },
          '#withEvaluator': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withEvaluator(value): {
            evaluator: value,
          },
          '#withEvaluatorMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withEvaluatorMixin(value): {
            evaluator+: value,
          },
          evaluator+:
            {
              '#withParams': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withParams(value): {
                evaluator+: {
                  params:
                    (if std.isArray(value)
                     then value
                     else [value]),
                },
              },
              '#withParamsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withParamsMixin(value): {
                evaluator+: {
                  params+:
                    (if std.isArray(value)
                     then value
                     else [value]),
                },
              },
              '#withType': { 'function': { args: [{ default: null, enums: ['gt', 'lt', 'eq', 'ne', 'gte', 'lte', 'within_range', 'outside_range', 'within_range_included', 'outside_range_included'], name: 'value', type: ['string'] }], help: 'e.g. "gt"' } },
              withType(value): {
                evaluator+: {
                  type: value,
                },
              },
            },
          '#withLoadedDimensions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withLoadedDimensions(value): {
            loadedDimensions: value,
          },
          '#withLoadedDimensionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withLoadedDimensionsMixin(value): {
            loadedDimensions+: value,
          },
          '#withUnloadEvaluator': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withUnloadEvaluator(value): {
            unloadEvaluator: value,
          },
          '#withUnloadEvaluatorMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withUnloadEvaluatorMixin(value): {
            unloadEvaluator+: value,
          },
          unloadEvaluator+:
            {
              '#withParams': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withParams(value): {
                unloadEvaluator+: {
                  params:
                    (if std.isArray(value)
                     then value
                     else [value]),
                },
              },
              '#withParamsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withParamsMixin(value): {
                unloadEvaluator+: {
                  params+:
                    (if std.isArray(value)
                     then value
                     else [value]),
                },
              },
              '#withType': { 'function': { args: [{ default: null, enums: ['gt', 'lt', 'eq', 'ne', 'gte', 'lte', 'within_range', 'outside_range', 'within_range_included', 'outside_range_included'], name: 'value', type: ['string'] }], help: 'e.g. "gt"' } },
              withType(value): {
                unloadEvaluator+: {
                  type: value,
                },
              },
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
      '#withExpression': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Reference to single query result' } },
      withExpression(value): {
        expression: value,
      },
      '#withHide': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'true if query is disabled (ie should not be returned to the dashboard)\nNOTE: this does not always imply that the query should not be executed since\nthe results from a hidden query may be used as the input to other queries (SSE etc)' } },
      withHide(value=true): {
        hide: value,
      },
      '#withIntervalMs': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Interval is the suggested duration between time points in a time series query.\nNOTE: the values for intervalMs is not saved in the query model.  It is typically calculated\nfrom the interval required to fill a pixels in the visualization' } },
      withIntervalMs(value): {
        intervalMs: value,
      },
      '#withMaxDataPoints': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'MaxDataPoints is the maximum number of data points that should be returned from a time series query.\nNOTE: the values for maxDataPoints is not saved in the query model.  It is typically calculated\nfrom the number of pixels visible in a visualization' } },
      withMaxDataPoints(value): {
        maxDataPoints: value,
      },
      '#withQueryType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'QueryType is an optional identifier for the type of query.\nIt can be used to distinguish different types of queries.' } },
      withQueryType(value): {
        queryType: value,
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
      '#withType': { 'function': { args: [], help: '' } },
      withType(): {
        type: 'threshold',
      },
    },
  TypeSql+:
    {
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
      '#withExpression': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withExpression(value): {
        expression: value,
      },
      '#withFormat': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withFormat(value): {
        format: value,
      },
      '#withHide': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'true if query is disabled (ie should not be returned to the dashboard)\nNOTE: this does not always imply that the query should not be executed since\nthe results from a hidden query may be used as the input to other queries (SSE etc)' } },
      withHide(value=true): {
        hide: value,
      },
      '#withIntervalMs': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Interval is the suggested duration between time points in a time series query.\nNOTE: the values for intervalMs is not saved in the query model.  It is typically calculated\nfrom the interval required to fill a pixels in the visualization' } },
      withIntervalMs(value): {
        intervalMs: value,
      },
      '#withMaxDataPoints': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'MaxDataPoints is the maximum number of data points that should be returned from a time series query.\nNOTE: the values for maxDataPoints is not saved in the query model.  It is typically calculated\nfrom the number of pixels visible in a visualization' } },
      withMaxDataPoints(value): {
        maxDataPoints: value,
      },
      '#withQueryType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'QueryType is an optional identifier for the type of query.\nIt can be used to distinguish different types of queries.' } },
      withQueryType(value): {
        queryType: value,
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
      '#withType': { 'function': { args: [], help: '' } },
      withType(): {
        type: 'sql',
      },
    },
}
+ (import '../custom/query/expr.libsonnet')
