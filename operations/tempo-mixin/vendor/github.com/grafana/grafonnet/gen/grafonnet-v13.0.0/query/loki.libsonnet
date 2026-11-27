// This file is generated, do not manually edit.
{
  '#': { help: 'grafonnet.query.loki', name: 'loki' },
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
  '#withDirection': { 'function': { args: [{ default: null, enums: ['forward', 'backward', 'scan'], name: 'value', type: ['string'] }], help: '' } },
  withDirection(value): {
    direction: value,
  },
  '#withEditorMode': { 'function': { args: [{ default: null, enums: ['code', 'builder'], name: 'value', type: ['string'] }], help: '' } },
  withEditorMode(value): {
    editorMode: value,
  },
  '#withExpr': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The LogQL query.' } },
  withExpr(value): {
    expr: value,
  },
  '#withHide': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'If hide is set to true, Grafana will filter out the response(s) associated with this query before returning it to the panel.' } },
  withHide(value=true): {
    hide: value,
  },
  '#withInstant': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '@deprecated, now use queryType.' } },
  withInstant(value=true): {
    instant: value,
  },
  '#withLegendFormat': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Used to override the name of the series.' } },
  withLegendFormat(value): {
    legendFormat: value,
  },
  '#withMaxLines': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'Used to limit the number of log rows returned.' } },
  withMaxLines(value): {
    maxLines: value,
  },
  '#withQueryType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Specify the query flavor\nTODO make this required and give it a default' } },
  withQueryType(value): {
    queryType: value,
  },
  '#withRange': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '@deprecated, now use queryType.' } },
  withRange(value=true): {
    range: value,
  },
  '#withRefId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'A unique identifier for the query within the list of targets.\nIn server side expressions, the refId is used as a variable name to identify results.\nBy default, the UI will assign A->Z; however setting meaningful names may be useful.' } },
  withRefId(value): {
    refId: value,
  },
  '#withResolution': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '@deprecated, now use step.' } },
  withResolution(value): {
    resolution: value,
  },
  '#withStep': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Used to set step value for range queries.' } },
  withStep(value): {
    step: value,
  },
}
+ (import '../custom/query/loki.libsonnet')
