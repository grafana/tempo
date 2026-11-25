// This file is generated, do not manually edit.
{
  '#': { help: 'grafonnet.query.googleCloudMonitoring', name: 'googleCloudMonitoring' },
  '#withAliasBy': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Aliases can be set to modify the legend labels. e.g. {{metric.label.xxx}}. See docs for more detail.' } },
  withAliasBy(value): {
    aliasBy: value,
  },
  '#withDatasource': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: "For mixed data sources the selected datasource is on the query level.\nFor non mixed scenarios this is undefined.\nTODO find a better way to do this ^ that's friendly to schema\nTODO this shouldn't be unknown but DataSourceRef | null" } },
  withDatasource(value): {
    datasource: value,
  },
  '#withDatasourceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: "For mixed data sources the selected datasource is on the query level.\nFor non mixed scenarios this is undefined.\nTODO find a better way to do this ^ that's friendly to schema\nTODO this shouldn't be unknown but DataSourceRef | null" } },
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
  '#withHide': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'If hide is set to true, Grafana will filter out the response(s) associated with this query before returning it to the panel.' } },
  withHide(value=true): {
    hide: value,
  },
  '#withIntervalMs': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Time interval in milliseconds.' } },
  withIntervalMs(value): {
    intervalMs: value,
  },
  '#withPromQLQuery': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'PromQL sub-query properties.' } },
  withPromQLQuery(value): {
    promQLQuery: value,
  },
  '#withPromQLQueryMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'PromQL sub-query properties.' } },
  withPromQLQueryMixin(value): {
    promQLQuery+: value,
  },
  promQLQuery+:
    {
      '#withExpr': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'PromQL expression/query to be executed.' } },
      withExpr(value): {
        promQLQuery+: {
          expr: value,
        },
      },
      '#withProjectName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'GCP project to execute the query against.' } },
      withProjectName(value): {
        promQLQuery+: {
          projectName: value,
        },
      },
      '#withStep': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'PromQL min step' } },
      withStep(value): {
        promQLQuery+: {
          step: value,
        },
      },
    },
  '#withQueryType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Specify the query flavor\nTODO make this required and give it a default' } },
  withQueryType(value): {
    queryType: value,
  },
  '#withRefId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'A unique identifier for the query within the list of targets.\nIn server side expressions, the refId is used as a variable name to identify results.\nBy default, the UI will assign A->Z; however setting meaningful names may be useful.' } },
  withRefId(value): {
    refId: value,
  },
  '#withSloQuery': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'SLO sub-query properties.' } },
  withSloQuery(value): {
    sloQuery: value,
  },
  '#withSloQueryMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'SLO sub-query properties.' } },
  withSloQueryMixin(value): {
    sloQuery+: value,
  },
  sloQuery+:
    {
      '#withAlignmentPeriod': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Alignment period to use when regularizing data. Defaults to cloud-monitoring-auto.' } },
      withAlignmentPeriod(value): {
        sloQuery+: {
          alignmentPeriod: value,
        },
      },
      '#withGoal': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'SLO goal value.' } },
      withGoal(value): {
        sloQuery+: {
          goal: value,
        },
      },
      '#withLookbackPeriod': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Specific lookback period for the SLO.' } },
      withLookbackPeriod(value): {
        sloQuery+: {
          lookbackPeriod: value,
        },
      },
      '#withPerSeriesAligner': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Alignment function to be used. Defaults to ALIGN_MEAN.' } },
      withPerSeriesAligner(value): {
        sloQuery+: {
          perSeriesAligner: value,
        },
      },
      '#withProjectName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'GCP project to execute the query against.' } },
      withProjectName(value): {
        sloQuery+: {
          projectName: value,
        },
      },
      '#withSelectorName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'SLO selector.' } },
      withSelectorName(value): {
        sloQuery+: {
          selectorName: value,
        },
      },
      '#withServiceId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'ID for the service the SLO is in.' } },
      withServiceId(value): {
        sloQuery+: {
          serviceId: value,
        },
      },
      '#withServiceName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Name for the service the SLO is in.' } },
      withServiceName(value): {
        sloQuery+: {
          serviceName: value,
        },
      },
      '#withSloId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'ID for the SLO.' } },
      withSloId(value): {
        sloQuery+: {
          sloId: value,
        },
      },
      '#withSloName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Name of the SLO.' } },
      withSloName(value): {
        sloQuery+: {
          sloName: value,
        },
      },
    },
  '#withTimeSeriesList': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Time Series List sub-query properties.' } },
  withTimeSeriesList(value): {
    timeSeriesList: value,
  },
  '#withTimeSeriesListMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Time Series List sub-query properties.' } },
  withTimeSeriesListMixin(value): {
    timeSeriesList+: value,
  },
  timeSeriesList+:
    {
      '#withAlignmentPeriod': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Alignment period to use when regularizing data. Defaults to cloud-monitoring-auto.' } },
      withAlignmentPeriod(value): {
        timeSeriesList+: {
          alignmentPeriod: value,
        },
      },
      '#withCrossSeriesReducer': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Reducer applied across a set of time-series values. Defaults to REDUCE_NONE.' } },
      withCrossSeriesReducer(value): {
        timeSeriesList+: {
          crossSeriesReducer: value,
        },
      },
      '#withFilters': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Array of filters to query data by. Labels that can be filtered on are defined by the metric.' } },
      withFilters(value): {
        timeSeriesList+: {
          filters:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withFiltersMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Array of filters to query data by. Labels that can be filtered on are defined by the metric.' } },
      withFiltersMixin(value): {
        timeSeriesList+: {
          filters+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withGroupBys': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Array of labels to group data by.' } },
      withGroupBys(value): {
        timeSeriesList+: {
          groupBys:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withGroupBysMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Array of labels to group data by.' } },
      withGroupBysMixin(value): {
        timeSeriesList+: {
          groupBys+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withPerSeriesAligner': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Alignment function to be used. Defaults to ALIGN_MEAN.' } },
      withPerSeriesAligner(value): {
        timeSeriesList+: {
          perSeriesAligner: value,
        },
      },
      '#withPreprocessor': { 'function': { args: [{ default: null, enums: ['none', 'rate', 'delta'], name: 'value', type: ['string'] }], help: 'Types of pre-processor available. Defined by the metric.' } },
      withPreprocessor(value): {
        timeSeriesList+: {
          preprocessor: value,
        },
      },
      '#withProjectName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'GCP project to execute the query against.' } },
      withProjectName(value): {
        timeSeriesList+: {
          projectName: value,
        },
      },
      '#withSecondaryAlignmentPeriod': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Only present if a preprocessor is selected. Alignment period to use when regularizing data. Defaults to cloud-monitoring-auto.' } },
      withSecondaryAlignmentPeriod(value): {
        timeSeriesList+: {
          secondaryAlignmentPeriod: value,
        },
      },
      '#withSecondaryCrossSeriesReducer': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Only present if a preprocessor is selected. Reducer applied across a set of time-series values. Defaults to REDUCE_NONE.' } },
      withSecondaryCrossSeriesReducer(value): {
        timeSeriesList+: {
          secondaryCrossSeriesReducer: value,
        },
      },
      '#withSecondaryGroupBys': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Only present if a preprocessor is selected. Array of labels to group data by.' } },
      withSecondaryGroupBys(value): {
        timeSeriesList+: {
          secondaryGroupBys:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withSecondaryGroupBysMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Only present if a preprocessor is selected. Array of labels to group data by.' } },
      withSecondaryGroupBysMixin(value): {
        timeSeriesList+: {
          secondaryGroupBys+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withSecondaryPerSeriesAligner': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Only present if a preprocessor is selected. Alignment function to be used. Defaults to ALIGN_MEAN.' } },
      withSecondaryPerSeriesAligner(value): {
        timeSeriesList+: {
          secondaryPerSeriesAligner: value,
        },
      },
      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Annotation text.' } },
      withText(value): {
        timeSeriesList+: {
          text: value,
        },
      },
      '#withTitle': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Annotation title.' } },
      withTitle(value): {
        timeSeriesList+: {
          title: value,
        },
      },
      '#withView': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Data view, defaults to FULL.' } },
      withView(value): {
        timeSeriesList+: {
          view: value,
        },
      },
    },
  '#withTimeSeriesQuery': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Time Series sub-query properties.' } },
  withTimeSeriesQuery(value): {
    timeSeriesQuery: value,
  },
  '#withTimeSeriesQueryMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Time Series sub-query properties.' } },
  withTimeSeriesQueryMixin(value): {
    timeSeriesQuery+: value,
  },
  timeSeriesQuery+:
    {
      '#withGraphPeriod': { 'function': { args: [{ default: 'disabled', enums: null, name: 'value', type: ['string'] }], help: "To disable the graphPeriod, it should explictly be set to 'disabled'." } },
      withGraphPeriod(value='disabled'): {
        timeSeriesQuery+: {
          graphPeriod: value,
        },
      },
      '#withProjectName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'GCP project to execute the query against.' } },
      withProjectName(value): {
        timeSeriesQuery+: {
          projectName: value,
        },
      },
      '#withQuery': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'MQL query to be executed.' } },
      withQuery(value): {
        timeSeriesQuery+: {
          query: value,
        },
      },
    },
}
+ (import '../custom/query/googleCloudMonitoring.libsonnet')
